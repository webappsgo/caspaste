package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// ProjectName is the project name for backup files
	ProjectName = "caspaste"
)

// Service provides backup and restore functionality
type Service struct {
	configDir  string
	dataDir    string
	backupDir  string
	db         *sql.DB
	encryption *EncryptionService
	retention  *RetentionService
	compliance bool
	appVersion string
}

// BackupOptions contains options for creating a backup
type BackupOptions struct {
	IncludeSSL     bool
	IncludeData    bool
	IncludeUploads bool
	Password       string
	CustomFilename string
}

// RestoreOptions contains options for restoring a backup
type RestoreOptions struct {
	Password   string
	SkipVerify bool
	DryRun     bool
}

// BackupResult contains the result of a backup operation
type BackupResult struct {
	Filename  string    `json:"filename"`
	FilePath  string    `json:"file_path"`
	Size      int64     `json:"size"`
	Checksum  string    `json:"checksum"`
	CreatedAt time.Time `json:"created_at"`
	Encrypted bool      `json:"encrypted"`
	Manifest  *Manifest `json:"manifest"`
}

// RestoreResult contains the result of a restore operation
type RestoreResult struct {
	BackupFile    string    `json:"backup_file"`
	RestoredAt    time.Time `json:"restored_at"`
	FilesRestored int       `json:"files_restored"`
	Manifest      *Manifest `json:"manifest"`
	SetupRequired bool      `json:"setup_required"`
	SetupToken    string    `json:"setup_token,omitempty"`
}

// NewService creates a new backup service. appVersion is the running
// binary's version string (injected via -ldflags at build time) and is
// embedded in every backup manifest so restores can detect
// version-skew. Pass "dev" when the version is not known.
func NewService(configDir, dataDir, backupDir, appVersion string, db *sql.DB) *Service {
	if appVersion == "" {
		appVersion = "dev"
	}
	return &Service{
		configDir:  configDir,
		dataDir:    dataDir,
		backupDir:  backupDir,
		db:         db,
		encryption: NewEncryptionService(),
		retention:  NewRetentionService(backupDir),
		appVersion: appVersion,
	}
}

// SetCompliance sets whether compliance mode is enabled
func (s *Service) SetCompliance(enabled bool) {
	s.compliance = enabled
}

// diskFullThresholdPct is the disk-usage percentage above which a scheduled
// backup is skipped, per AI.md PART 22 backup_daily step 2 (default 90%).
const diskFullThresholdPct = 90.0

// CheckDiskSpace evaluates whether there is enough free space to safely run
// a *scheduled* backup, per AI.md PART 22 backup_daily step 2: skip if free
// space is less than 2x the size of the most recent backup, OR disk usage
// exceeds 90%. Manual (CLI/API-triggered) backups never call this — only
// the backup_daily / backup_hourly scheduler tasks do. A true skip return
// is not an error; the caller should log backup.skipped_disk_full and abort
// without attempting Create.
func (s *Service) CheckDiskSpace() (skip bool, reason string, err error) {
	total, err := diskTotalBytes(s.backupDir)
	if err != nil {
		return false, "", fmt.Errorf("failed to stat backup disk: %w", err)
	}
	free, err := diskFreeBytes(s.backupDir)
	if err != nil {
		return false, "", fmt.Errorf("failed to stat backup disk free space: %w", err)
	}

	if total > 0 {
		usedPct := float64(total-free) / float64(total) * 100
		if usedPct > diskFullThresholdPct {
			return true, fmt.Sprintf("disk usage %.1f%% exceeds %.0f%% threshold (free %s of %s)",
				usedPct, diskFullThresholdPct, FormatBytes(int64(free)), FormatBytes(int64(total))), nil
		}
	}

	backups, err := s.retention.ListBackups()
	if err != nil {
		return false, "", fmt.Errorf("failed to list existing backups: %w", err)
	}
	if len(backups) > 0 {
		// ListBackups returns newest first.
		lastSize := backups[0].Size
		needed := lastSize * 2
		if int64(free) < needed {
			return true, fmt.Sprintf("free space %s is less than 2x last backup size (%s needed)",
				FormatBytes(int64(free)), FormatBytes(needed)), nil
		}
	}

	return false, "", nil
}

// Create creates a new backup
func (s *Service) Create(ctx context.Context, opts BackupOptions) (*BackupResult, error) {
	// Check compliance requirements
	if s.compliance && opts.Password == "" {
		return nil, ErrComplianceRequiresEncryption
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(s.backupDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02_150405")
	filename := opts.CustomFilename
	if filename == "" {
		filename = fmt.Sprintf("%s_backup_%s", ProjectName, timestamp)
	}

	// Create manifest
	manifest := NewManifest()
	manifest.AppVersion = s.appVersion
	manifest.Encrypted = opts.Password != ""

	// Create temporary directory for staging
	tempDir, err := os.MkdirTemp("", "caspaste-backup-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Stage files for backup
	if err := s.stageBackupFiles(tempDir, opts, manifest); err != nil {
		return nil, fmt.Errorf("failed to stage backup files: %w", err)
	}

	// Calculate the content checksum over the staged files (before the manifest
	// is written), so it can be recomputed and verified from an extracted archive.
	checksum, err := computeContentChecksum(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}
	manifest.Checksum = checksum

	// Save manifest to the staging directory so it is included in the archive
	manifestPath := filepath.Join(tempDir, "manifest.json")
	if err := manifest.Save(manifestPath); err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	// Create the tar.gz archive with the manifest embedded
	archivePath := filepath.Join(tempDir, filename+".tar.gz")
	if err := s.createArchive(tempDir, archivePath, manifest); err != nil {
		return nil, fmt.Errorf("failed to create archive: %w", err)
	}

	// Final filename and path
	finalFilename := filename + ".tar.gz"
	finalPath := filepath.Join(s.backupDir, finalFilename)

	// Encrypt if password provided
	if opts.Password != "" {
		encryptedPath, err := s.encryption.EncryptFile(archivePath, opts.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt backup: %w", err)
		}
		archivePath = encryptedPath
		finalFilename = filename + ".tar.gz.enc"
		finalPath = filepath.Join(s.backupDir, finalFilename)
	}

	// Move to backup directory
	if err := s.moveFile(archivePath, finalPath); err != nil {
		return nil, fmt.Errorf("failed to move backup to destination: %w", err)
	}

	// Get file info
	info, err := os.Stat(finalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	// Verify backup
	verifyOpts := VerifyOptions{
		Password: opts.Password,
	}
	if err := s.Verify(finalPath, verifyOpts); err != nil {
		// Delete failed backup
		os.Remove(finalPath)
		return nil, fmt.Errorf("backup verification failed: %w", err)
	}

	// Apply retention policy
	if err := s.retention.Apply(); err != nil {
		log.Printf("backup: failed to apply retention policy: %v", err)
	}

	log.Printf("backup: created %s (size: %d bytes)", finalFilename, info.Size())

	return &BackupResult{
		Filename:  finalFilename,
		FilePath:  finalPath,
		Size:      info.Size(),
		Checksum:  manifest.Checksum,
		CreatedAt: time.Now(),
		Encrypted: opts.Password != "",
		Manifest:  manifest,
	}, nil
}

// Restore restores from a backup file
func (s *Service) Restore(ctx context.Context, backupPath string, opts RestoreOptions) (*RestoreResult, error) {
	// Check if file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return nil, ErrBackupNotFound
	}

	// Check if encrypted
	isEncrypted := strings.HasSuffix(backupPath, ".enc")
	if isEncrypted && opts.Password == "" {
		return nil, ErrPasswordRequired
	}

	// Create temp directory for extraction
	tempDir, err := os.MkdirTemp("", "caspaste-restore-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	archivePath := backupPath

	// Decrypt if necessary
	if isEncrypted {
		decryptedPath, err := s.encryption.DecryptFile(backupPath, opts.Password, tempDir)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt backup: %w", err)
		}
		archivePath = decryptedPath
	}

	// Verify if not skipped
	if !opts.SkipVerify {
		verifyOpts := VerifyOptions{
			Password: opts.Password,
		}
		if err := s.Verify(backupPath, verifyOpts); err != nil {
			return nil, fmt.Errorf("backup verification failed: %w", err)
		}
	}

	// Extract archive
	extractDir := filepath.Join(tempDir, "extracted")
	if err := s.extractArchive(archivePath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Load manifest
	manifestPath := filepath.Join(extractDir, "manifest.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	// Dry run - just verify
	if opts.DryRun {
		return &RestoreResult{
			BackupFile:    backupPath,
			RestoredAt:    time.Now(),
			FilesRestored: len(manifest.Contents),
			Manifest:      manifest,
		}, nil
	}

	// Restore files
	filesRestored := 0
	for _, content := range manifest.Contents {
		srcPath := filepath.Join(extractDir, content)
		var dstPath string

		// Determine destination based on file type
		switch {
		case content == "server.yml":
			dstPath = filepath.Join(s.configDir, "server.yml")
		case content == "server.db":
			dstPath = filepath.Join(s.dataDir, "db", "server.db")
		case content == "users.db":
			dstPath = filepath.Join(s.dataDir, "db", "users.db")
		case strings.HasPrefix(content, "template/"):
			dstPath = filepath.Join(s.configDir, content)
		case strings.HasPrefix(content, "theme/"):
			dstPath = filepath.Join(s.configDir, content)
		case strings.HasPrefix(content, "ssl/"):
			dstPath = filepath.Join(s.configDir, content)
		case strings.HasPrefix(content, "uploads/"):
			dstPath = filepath.Join(s.dataDir, content)
		default:
			continue
		}

		// Check if source exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		// Ensure destination directory exists
		dstDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			log.Printf("backup: failed to create directory %s: %v", dstDir, err)
			continue
		}

		// Copy file
		if err := s.copyFile(srcPath, dstPath); err != nil {
			log.Printf("backup: failed to restore %s: %v", content, err)
			continue
		}
		filesRestored++
	}

	// Generate new setup token for security
	setupToken := generateSetupToken()

	log.Printf("backup: restored %s (%d files)", backupPath, filesRestored)

	return &RestoreResult{
		BackupFile:    backupPath,
		RestoredAt:    time.Now(),
		FilesRestored: filesRestored,
		Manifest:      manifest,
		SetupRequired: true,
		SetupToken:    setupToken,
	}, nil
}

// List returns all available backups
func (s *Service) List() ([]BackupInfo, error) {
	return s.retention.ListBackups()
}

// Delete deletes a specific backup
func (s *Service) Delete(filename string) error {
	backupPath := filepath.Join(s.backupDir, filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return ErrBackupNotFound
	}
	return os.Remove(backupPath)
}

// GetBackupDir returns the backup directory path
func (s *Service) GetBackupDir() string {
	return s.backupDir
}

// GetRetentionConfig returns the current backup retention configuration.
func (s *Service) GetRetentionConfig() RetentionConfig {
	return s.retention.GetConfig()
}

// SetRetentionConfig replaces the backup retention configuration.
func (s *Service) SetRetentionConfig(cfg RetentionConfig) {
	s.retention.SetConfig(cfg)
}

// stageBackupFiles stages files for backup
func (s *Service) stageBackupFiles(tempDir string, opts BackupOptions, manifest *Manifest) error {
	// Always include server.yml
	serverYml := filepath.Join(s.configDir, "server.yml")
	if _, err := os.Stat(serverYml); err == nil {
		dst := filepath.Join(tempDir, "server.yml")
		if err := s.copyFile(serverYml, dst); err != nil {
			return fmt.Errorf("failed to copy server.yml: %w", err)
		}
		manifest.Contents = append(manifest.Contents, "server.yml")
	}

	// Always include server.db
	serverDB := filepath.Join(s.dataDir, "db", "server.db")
	if _, err := os.Stat(serverDB); err == nil {
		dst := filepath.Join(tempDir, "server.db")
		if err := s.copyFile(serverDB, dst); err != nil {
			return fmt.Errorf("failed to copy server.db: %w", err)
		}
		manifest.Contents = append(manifest.Contents, "server.db")
	}

	// Include users.db if exists
	usersDB := filepath.Join(s.dataDir, "db", "users.db")
	if _, err := os.Stat(usersDB); err == nil {
		dst := filepath.Join(tempDir, "users.db")
		if err := s.copyFile(usersDB, dst); err != nil {
			return fmt.Errorf("failed to copy users.db: %w", err)
		}
		manifest.Contents = append(manifest.Contents, "users.db")
	}

	// Include template directory if exists
	templateDir := filepath.Join(s.configDir, "template")
	if _, err := os.Stat(templateDir); err == nil {
		dst := filepath.Join(tempDir, "template")
		if err := s.copyDir(templateDir, dst); err != nil {
			return fmt.Errorf("failed to copy template directory: %w", err)
		}
		manifest.Contents = append(manifest.Contents, "template/")
	}

	// Include theme directory if exists
	themeDir := filepath.Join(s.configDir, "theme")
	if _, err := os.Stat(themeDir); err == nil {
		dst := filepath.Join(tempDir, "theme")
		if err := s.copyDir(themeDir, dst); err != nil {
			return fmt.Errorf("failed to copy theme directory: %w", err)
		}
		manifest.Contents = append(manifest.Contents, "theme/")
	}

	// Include SSL if requested
	if opts.IncludeSSL {
		sslDir := filepath.Join(s.configDir, "ssl")
		if _, err := os.Stat(sslDir); err == nil {
			dst := filepath.Join(tempDir, "ssl")
			if err := s.copyDir(sslDir, dst); err != nil {
				return fmt.Errorf("failed to copy ssl directory: %w", err)
			}
			manifest.Contents = append(manifest.Contents, "ssl/")
		}
	}

	// Include uploads if requested
	if opts.IncludeUploads {
		uploadsDir := filepath.Join(s.dataDir, "uploads")
		if _, err := os.Stat(uploadsDir); err == nil {
			dst := filepath.Join(tempDir, "uploads")
			if err := s.copyDir(uploadsDir, dst); err != nil {
				return fmt.Errorf("failed to copy uploads directory: %w", err)
			}
			manifest.Contents = append(manifest.Contents, "uploads/")
		}
	}

	return nil
}

// createArchive creates a tar.gz archive
func (s *Service) createArchive(sourceDir, archivePath string, manifest *Manifest) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk the source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the archive file itself
		if path == archivePath {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Create header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file content if not a directory
		if !info.IsDir() {
			fileContent, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fileContent.Close()

			if _, err := io.Copy(tarWriter, fileContent); err != nil {
				return err
			}
		}

		return nil
	})
}

// extractArchive extracts a tar.gz archive
// Archive extraction safety limits per AI.md PART 11 → "Archive Extraction
// Safety". These guard against compression bombs and runaway extraction.
const (
	// maxExtractFiles caps the number of entries extracted from one archive.
	maxExtractFiles = 200000
	// maxExtractSingleFile caps the uncompressed size of any single entry.
	maxExtractSingleFile int64 = 4 << 30 // 4 GiB
	// maxExtractTotalSize caps the total uncompressed size of the archive.
	maxExtractTotalSize int64 = 20 << 30 // 20 GiB
	// maxCompressionRatio caps expanded/compressed ratio (bomb detection).
	maxCompressionRatio int64 = 200
)

func (s *Service) extractArchive(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	archiveInfo, err := file.Stat()
	if err != nil {
		return err
	}
	// Compression-bomb ceiling derived from the on-disk archive size.
	ratioCeiling := archiveInfo.Size() * maxCompressionRatio
	if ratioCeiling < maxExtractSingleFile {
		ratioCeiling = maxExtractSingleFile
	}

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var (
		fileCount int
		totalSize int64
	)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fileCount++
		if fileCount > maxExtractFiles {
			return fmt.Errorf("archive exceeds max file count (%d)", maxExtractFiles)
		}

		// Reject absolute/empty names and path traversal before touching disk.
		if header.Name == "" || filepath.IsAbs(header.Name) || strings.Contains(header.Name, "..") {
			return fmt.Errorf("invalid file name in archive: %q", header.Name)
		}

		// Sanitize path to prevent directory traversal
		targetPath := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Reject an oversized single entry up front via the header size.
			if header.Size > maxExtractSingleFile {
				return fmt.Errorf("archive entry %q exceeds max single-file size", header.Name)
			}

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}

			// Bound the copy so a lying header cannot exhaust the disk, and
			// stop when the cumulative total or compression ratio is exceeded.
			limit := maxExtractSingleFile + 1
			written, err := io.Copy(outFile, io.LimitReader(tarReader, limit))
			outFile.Close()
			if err != nil {
				return err
			}
			if written >= limit {
				return fmt.Errorf("archive entry %q exceeds max single-file size", header.Name)
			}

			totalSize += written
			if totalSize > maxExtractTotalSize {
				return fmt.Errorf("archive exceeds max total uncompressed size")
			}
			if totalSize > ratioCeiling {
				return fmt.Errorf("archive exceeds max compression ratio (possible bomb)")
			}

			// Set permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		default:
			// Reject symlinks, hard links, device/special files per PART 11.
			return fmt.Errorf("archive entry %q has unsupported type %d", header.Name, header.Typeflag)
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func (s *Service) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir recursively copies a directory
func (s *Service) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return s.copyFile(path, dstPath)
	})
}

// moveFile moves a file from src to dst
func (s *Service) moveFile(src, dst string) error {
	// Try rename first (works if same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fall back to copy + delete
	if err := s.copyFile(src, dst); err != nil {
		return err
	}

	return os.Remove(src)
}

// generateSetupToken generates a random setup token
func generateSetupToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based token
		return fmt.Sprintf("%x", sha256.Sum256([]byte(time.Now().String())))[:32]
	}
	return hex.EncodeToString(bytes)
}
