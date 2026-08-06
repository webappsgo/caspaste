package backup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RetentionConfig contains backup retention settings
type RetentionConfig struct {
	// Daily full backups to keep (default: 1)
	MaxBackups int `yaml:"max_backups" json:"max_backups"`
	// Weekly backups to keep (0 = disabled)
	KeepWeekly int `yaml:"keep_weekly" json:"keep_weekly"`
	// Monthly backups to keep (0 = disabled)
	KeepMonthly int `yaml:"keep_monthly" json:"keep_monthly"`
	// Yearly backups to keep (0 = disabled)
	KeepYearly int `yaml:"keep_yearly" json:"keep_yearly"`
	// Hard size cap applied last: percent ("10%") or absolute ("50G"); "" or "0" = disabled
	MaxTotalSize string `yaml:"max_total_size" json:"max_total_size"`
}

// DefaultRetentionConfig returns default retention settings
func DefaultRetentionConfig() RetentionConfig {
	return RetentionConfig{
		MaxBackups:   1,
		KeepWeekly:   0,
		KeepMonthly:  0,
		KeepYearly:   0,
		MaxTotalSize: "10%",
	}
}

// sizeSuffixBytes maps a single-letter size suffix to its byte multiplier
var sizeSuffixBytes = map[byte]int64{
	'K': 1 << 10,
	'M': 1 << 20,
	'G': 1 << 30,
	'T': 1 << 40,
}

// parseMaxTotalSize resolves a "max_total_size" value against the total capacity
// of the filesystem containing backupDir. Returns 0 (no cap) for "", "0", or an
// unparseable value (logged as a warning per AI.md config-validation rules).
func parseMaxTotalSize(value, backupDir string) int64 {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" {
		return 0
	}

	if strings.HasSuffix(value, "%") {
		pct, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
		if err != nil || pct <= 0 || pct > 100 {
			log.Printf("backup: max_total_size: %q invalid percentage, disabling size cap", value)
			return 0
		}

		total, err := diskTotalBytes(backupDir)
		if err != nil {
			log.Printf("backup: max_total_size: failed to stat disk for %s: %v, disabling size cap", backupDir, err)
			return 0
		}

		return int64(float64(total) * pct / 100)
	}

	upper := strings.ToUpper(value)
	if len(upper) >= 2 {
		suffix := upper[len(upper)-1]
		if mult, ok := sizeSuffixBytes[suffix]; ok {
			num, err := strconv.ParseFloat(strings.TrimSpace(upper[:len(upper)-1]), 64)
			if err != nil || num <= 0 {
				log.Printf("backup: max_total_size: %q invalid, disabling size cap", value)
				return 0
			}
			return int64(num * float64(mult))
		}
	}

	// Bare number = bytes
	bytes, err := strconv.ParseInt(upper, 10, 64)
	if err != nil || bytes <= 0 {
		log.Printf("backup: max_total_size: %q invalid, disabling size cap", value)
		return 0
	}
	return bytes
}

// FormatBytes renders a byte count as a human-readable string (base 1024)
func FormatBytes(n int64) string {
	units := []string{"B", "K", "M", "G", "T"}
	f := float64(n)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	return fmt.Sprintf("%.1f%s", f, units[i])
}

// BackupInfo contains information about a backup file
type BackupInfo struct {
	Filename  string    `json:"filename"`
	FilePath  string    `json:"file_path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Encrypted bool      `json:"encrypted"`
	// daily, weekly, monthly, yearly, incremental
	Type    string `json:"type"`
	IsValid bool   `json:"is_valid"`
}

// RetentionService manages backup retention
type RetentionService struct {
	backupDir string
	config    RetentionConfig
}

// NewRetentionService creates a new retention service
func NewRetentionService(backupDir string) *RetentionService {
	return &RetentionService{
		backupDir: backupDir,
		config:    DefaultRetentionConfig(),
	}
}

// SetConfig sets the retention configuration
func (r *RetentionService) SetConfig(config RetentionConfig) {
	// Validate and apply defaults for invalid values
	if config.MaxBackups < 1 {
		log.Printf("backup: max_backups: %d invalid, using default 1", config.MaxBackups)
		config.MaxBackups = 1
	}
	if config.KeepWeekly < 0 {
		log.Printf("backup: keep_weekly: %d invalid, using default 0", config.KeepWeekly)
		config.KeepWeekly = 0
	}
	if config.KeepMonthly < 0 {
		log.Printf("backup: keep_monthly: %d invalid, using default 0", config.KeepMonthly)
		config.KeepMonthly = 0
	}
	if config.KeepYearly < 0 {
		log.Printf("backup: keep_yearly: %d invalid, using default 0", config.KeepYearly)
		config.KeepYearly = 0
	}

	// Warn for high values
	if config.MaxBackups > 7 {
		log.Printf("backup: max_backups: %d exceeds recommended 7 (%d days of daily backups)", config.MaxBackups, config.MaxBackups)
	}
	if config.KeepWeekly > 8 {
		log.Printf("backup: keep_weekly: %d exceeds recommended 8 (%d weeks of weekly backups)", config.KeepWeekly, config.KeepWeekly)
	}
	if config.KeepMonthly > 12 {
		log.Printf("backup: keep_monthly: %d exceeds recommended 12 (%d months of monthly backups)", config.KeepMonthly, config.KeepMonthly)
	}
	if config.KeepYearly > 2 {
		log.Printf("backup: keep_yearly: %d exceeds recommended 2 (%d years of yearly backups)", config.KeepYearly, config.KeepYearly)
	}

	r.config = config
}

// GetConfig returns the current retention configuration
func (r *RetentionService) GetConfig() RetentionConfig {
	return r.config
}

// ListBackups returns all backup files in the backup directory
func (r *RetentionService) ListBackups() ([]BackupInfo, error) {
	var backups []BackupInfo

	entries, err := os.ReadDir(r.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return backups, nil
		}
		return nil, err
	}

	// Pattern for full backups: caspaste_backup_YYYY-MM-DD_HHMMSS.tar.gz[.enc]
	fullPattern := regexp.MustCompile(`^caspaste_backup_(\d{4}-\d{2}-\d{2})_(\d{6})\.tar\.gz(\.enc)?$`)
	// Pattern for incremental backups: caspaste-daily.tar.gz[.enc], caspaste-hourly.tar.gz[.enc]
	incrPattern := regexp.MustCompile(`^caspaste-(daily|hourly)\.tar\.gz(\.enc)?$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		info, err := entry.Info()
		if err != nil {
			continue
		}

		backup := BackupInfo{
			Filename:  filename,
			FilePath:  filepath.Join(r.backupDir, filename),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			Encrypted: strings.HasSuffix(filename, ".enc"),
			// Assume valid until verified
			IsValid: true,
		}

		// Determine backup type
		if matches := fullPattern.FindStringSubmatch(filename); matches != nil {
			dateStr := matches[1]
			backupDate, _ := time.Parse("2006-01-02", dateStr)

			// Categorize based on date
			if backupDate.YearDay() == 1 {
				backup.Type = "yearly"
			} else if backupDate.Day() == 1 {
				backup.Type = "monthly"
			} else if backupDate.Weekday() == time.Sunday {
				backup.Type = "weekly"
			} else {
				backup.Type = "daily"
			}
		} else if matches := incrPattern.FindStringSubmatch(filename); matches != nil {
			backup.Type = "incremental"
		} else {
			// Skip unrecognized files
			continue
		}

		backups = append(backups, backup)
	}

	// Sort by creation time, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// Apply applies the retention policy and removes old backups
func (r *RetentionService) Apply() error {
	backups, err := r.ListBackups()
	if err != nil {
		return err
	}

	// Separate backups by type
	var yearly, monthly, weekly, daily []BackupInfo

	for _, backup := range backups {
		switch backup.Type {
		case "yearly":
			yearly = append(yearly, backup)
		case "monthly":
			monthly = append(monthly, backup)
		case "weekly":
			weekly = append(weekly, backup)
		case "daily":
			daily = append(daily, backup)
		case "incremental":
			// Always keep incrementals (only 1 of each type exists)
			continue
		}
	}

	// Determine which backups to delete based on priority
	toDelete := make([]BackupInfo, 0)

	// Keep yearly backups (highest priority)
	if r.config.KeepYearly > 0 && len(yearly) > r.config.KeepYearly {
		toDelete = append(toDelete, yearly[r.config.KeepYearly:]...)
	}

	// Keep monthly backups
	if r.config.KeepMonthly > 0 && len(monthly) > r.config.KeepMonthly {
		toDelete = append(toDelete, monthly[r.config.KeepMonthly:]...)
	}

	// Keep weekly backups
	if r.config.KeepWeekly > 0 && len(weekly) > r.config.KeepWeekly {
		toDelete = append(toDelete, weekly[r.config.KeepWeekly:]...)
	}

	// Keep daily backups (lowest priority)
	if len(daily) > r.config.MaxBackups {
		toDelete = append(toDelete, daily[r.config.MaxBackups:]...)
	}

	// Apply max_total_size hard cap last, oldest deleted first, across whatever
	// survives the priority-based selection above (per AI.md PART 22 retention order)
	toDelete = append(toDelete, r.applySizeCap(backups, toDelete)...)

	// Delete old backups
	for _, backup := range toDelete {
		if err := os.Remove(backup.FilePath); err != nil {
			log.Printf("backup: failed to delete old backup %s: %v", backup.Filename, err)
		} else {
			log.Printf("backup: deleted old backup: %s (type: %s)", backup.Filename, backup.Type)
		}
	}

	if len(toDelete) > 0 {
		log.Printf("backup: retention policy applied: deleted %d backups", len(toDelete))
	}

	return nil
}

// applySizeCap returns additional backups to delete (oldest first) so that the
// total size of survivors (all backups minus alreadyDeleted) fits under the
// configured max_total_size. Incrementals are never deleted by the size cap —
// they're each a single always-replaced file, not part of the countable set.
func (r *RetentionService) applySizeCap(all, alreadyDeleted []BackupInfo) []BackupInfo {
	sizeCap := parseMaxTotalSize(r.config.MaxTotalSize, r.backupDir)
	if sizeCap <= 0 {
		return nil
	}

	deleted := make(map[string]bool, len(alreadyDeleted))
	for _, b := range alreadyDeleted {
		deleted[b.FilePath] = true
	}

	// Survivors, oldest first (ListBackups returns newest first)
	survivors := make([]BackupInfo, 0, len(all))
	var total int64
	for _, b := range all {
		if deleted[b.FilePath] || b.Type == "incremental" {
			continue
		}
		survivors = append(survivors, b)
		total += b.Size
	}
	sort.Slice(survivors, func(i, j int) bool {
		return survivors[i].CreatedAt.Before(survivors[j].CreatedAt)
	})

	if total <= sizeCap {
		return nil
	}

	var extra []BackupInfo
	for _, b := range survivors {
		if total <= sizeCap {
			break
		}
		extra = append(extra, b)
		total -= b.Size
	}

	if len(extra) > 0 {
		log.Printf("backup: max_total_size %s exceeded, deleting %d oldest backup(s) to reclaim space", FormatBytes(sizeCap), len(extra))
	}

	return extra
}

// CleanupFailed removes backups that failed verification
func (r *RetentionService) CleanupFailed(filename string) error {
	backupPath := filepath.Join(r.backupDir, filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(backupPath)
}

// GetStorageInfo returns storage statistics for backups
func (r *RetentionService) GetStorageInfo() (int64, int, error) {
	backups, err := r.ListBackups()
	if err != nil {
		return 0, 0, err
	}

	var totalSize int64
	for _, backup := range backups {
		totalSize += backup.Size
	}

	return totalSize, len(backups), nil
}

// EstimateStorage estimates storage based on current backup sizes and retention settings
func (r *RetentionService) EstimateStorage() (int64, error) {
	backups, err := r.ListBackups()
	if err != nil {
		return 0, err
	}

	if len(backups) == 0 {
		return 0, nil
	}

	// Calculate average backup size
	var totalSize int64
	var fullCount int
	for _, backup := range backups {
		if backup.Type != "incremental" {
			totalSize += backup.Size
			fullCount++
		}
	}

	if fullCount == 0 {
		return 0, nil
	}

	avgSize := totalSize / int64(fullCount)

	// Estimate based on retention settings
	expectedCount := r.config.MaxBackups + r.config.KeepWeekly + r.config.KeepMonthly + r.config.KeepYearly
	// Add incrementals (daily + hourly)
	expectedCount += 2

	return avgSize * int64(expectedCount), nil
}
