package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// VerifyOptions contains options for backup verification
type VerifyOptions struct {
	Password string
	// If set, extracts to this directory as part of verification
	ExtractToDir string
}

// VerifyResult contains the result of backup verification
type VerifyResult struct {
	Valid          bool     `json:"valid"`
	Encrypted      bool     `json:"encrypted"`
	FileExists     bool     `json:"file_exists"`
	SizeValid      bool     `json:"size_valid"`
	ChecksumValid  bool     `json:"checksum_valid"`
	DecryptValid   bool     `json:"decrypt_valid"`
	ManifestValid  bool     `json:"manifest_valid"`
	ExtractValid   bool     `json:"extract_valid"`
	DatabaseValid  bool     `json:"database_valid"`
	Errors         []string `json:"errors,omitempty"`
	FilesInArchive int      `json:"files_in_archive"`
}

// Verify verifies a backup file
func (s *Service) Verify(backupPath string, opts VerifyOptions) error {
	result := s.VerifyDetailed(backupPath, opts)
	if !result.Valid {
		if len(result.Errors) > 0 {
			return fmt.Errorf("%s", result.Errors[0])
		}
		return ErrVerificationFailed
	}
	return nil
}

// VerifyDetailed performs detailed verification of a backup
func (s *Service) VerifyDetailed(backupPath string, opts VerifyOptions) *VerifyResult {
	result := &VerifyResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// Check file exists
	info, err := os.Stat(backupPath)
	if err != nil {
		result.Valid = false
		result.FileExists = false
		result.Errors = append(result.Errors, fmt.Sprintf("file not found: %s", backupPath))
		return result
	}
	result.FileExists = true

	// Check size > 0
	if info.Size() == 0 {
		result.Valid = false
		result.SizeValid = false
		result.Errors = append(result.Errors, "backup file is empty")
		return result
	}
	result.SizeValid = true

	// Check if encrypted
	result.Encrypted = strings.HasSuffix(backupPath, ".enc")

	// If encrypted, verify decryption
	archivePath := backupPath
	tempDir := ""

	if result.Encrypted {
		if opts.Password == "" {
			result.Valid = false
			result.DecryptValid = false
			result.Errors = append(result.Errors, "encrypted backup requires password")
			return result
		}

		// Create temp directory for decryption
		tempDir, err = os.MkdirTemp("", "caspaste-verify-*")
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create temp dir: %v", err))
			return result
		}
		defer os.RemoveAll(tempDir)

		// Decrypt
		decryptedPath, err := s.encryption.DecryptFile(backupPath, opts.Password, tempDir)
		if err != nil {
			result.Valid = false
			result.DecryptValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("decryption failed: %v", err))
			return result
		}
		result.DecryptValid = true
		archivePath = decryptedPath
	} else {
		result.DecryptValid = true
	}

	// Create temp directory for extraction if not already created
	if tempDir == "" {
		tempDir, err = os.MkdirTemp("", "caspaste-verify-*")
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create temp dir: %v", err))
			return result
		}
		defer os.RemoveAll(tempDir)
	}

	// Verify archive structure and extract
	extractDir := filepath.Join(tempDir, "extracted")
	filesExtracted, err := s.verifyAndExtractArchive(archivePath, extractDir)
	if err != nil {
		result.Valid = false
		result.ExtractValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("extraction failed: %v", err))
		return result
	}
	result.ExtractValid = true
	result.FilesInArchive = filesExtracted

	// Verify manifest
	manifestPath := filepath.Join(extractDir, "manifest.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		result.Valid = false
		result.ManifestValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid manifest: %v", err))
		return result
	}

	if err := manifest.Validate(); err != nil {
		result.Valid = false
		result.ManifestValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("manifest validation failed: %v", err))
		return result
	}
	result.ManifestValid = true

	// Verify checksum: recompute the content checksum over the extracted files
	// (excluding manifest.json, which is not part of the staged content that was
	// hashed at creation time) and compare against the value in the manifest.
	if manifest.Checksum != "" {
		actual, err := computeContentChecksum(extractDir, "manifest.json")
		if err != nil {
			result.Valid = false
			result.ChecksumValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("failed to compute checksum: %v", err))
			return result
		}
		if actual != manifest.Checksum {
			result.Valid = false
			result.ChecksumValid = false
			result.Errors = append(result.Errors, "backup corrupted: checksum mismatch")
			return result
		}
		result.ChecksumValid = true
	} else {
		result.ChecksumValid = true
	}

	// Verify database integrity
	serverDBPath := filepath.Join(extractDir, "server.db")
	if _, err := os.Stat(serverDBPath); err == nil {
		if err := s.verifyDatabase(serverDBPath); err != nil {
			result.Valid = false
			result.DatabaseValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("database integrity check failed: %v", err))
			return result
		}
		result.DatabaseValid = true
	} else {
		// No server.db - might be okay for minimal backups
		result.DatabaseValid = true
	}

	return result
}

// verifyAndExtractArchive verifies and extracts a tar.gz archive
func (s *Service) verifyAndExtractArchive(archivePath, extractDir string) (int, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return 0, fmt.Errorf("invalid gzip format: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Create extraction directory
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create extract dir: %w", err)
	}

	filesExtracted := 0

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return filesExtracted, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Sanitize path to prevent directory traversal
		targetPath := filepath.Join(extractDir, header.Name)
		if !strings.HasPrefix(targetPath, filepath.Clean(extractDir)+string(os.PathSeparator)) {
			return filesExtracted, fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return filesExtracted, fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return filesExtracted, fmt.Errorf("failed to create parent dir: %w", err)
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return filesExtracted, fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return filesExtracted, fmt.Errorf("failed to extract file: %w", err)
			}
			outFile.Close()

			filesExtracted++
		}
	}

	return filesExtracted, nil
}

// verifyDatabase verifies SQLite database integrity
func (s *Service) verifyDatabase(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Run integrity check
	var result string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		return fmt.Errorf("integrity check query failed: %w", err)
	}

	if result != "ok" {
		return fmt.Errorf("integrity check failed: %s", result)
	}

	return nil
}

// CalculateFileChecksum calculates SHA256 checksum of a file
func CalculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

// computeContentChecksum computes a deterministic combined SHA256 over every
// regular file under root (relative path + content, in sorted path order),
// skipping any file whose base name is listed in skip. Because the value is
// derived from staged content rather than the wrapping tar/gzip stream, the
// same checksum can be recomputed from an extracted archive on the verify side,
// which makes real (non-fake) checksum verification possible.
func computeContentChecksum(root string, skip ...string) (string, error) {
	skipSet := make(map[string]bool, len(skip))
	for _, name := range skip {
		skipSet[name] = true
	}

	var relPaths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		if skipSet[filepath.Base(path)] {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relPaths = append(relPaths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(relPaths)

	hash := sha256.New()
	for _, rel := range relPaths {
		if _, err := io.WriteString(hash, rel+"\n"); err != nil {
			return "", err
		}
		f, err := os.Open(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(hash, f); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
	}

	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
