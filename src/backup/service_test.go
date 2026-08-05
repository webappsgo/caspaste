package backup

import (
	"os"
	"path/filepath"
	"testing"
)

// TestServiceGettersSetters exercises the small accessors on Service and its
// retention configuration passthrough.
func TestServiceGettersSetters(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		t.Fatal(err)
	}
	s := NewService(dir, dir, backupDir, "", nil)

	if s.appVersion != "dev" {
		t.Fatalf("empty appVersion should default to dev, got %q", s.appVersion)
	}
	if s.GetBackupDir() != backupDir {
		t.Fatalf("GetBackupDir mismatch: %q", s.GetBackupDir())
	}

	s.SetCompliance(true)
	if !s.compliance {
		t.Fatal("SetCompliance(true) did not set field")
	}

	cfg := DefaultRetentionConfig()
	cfg.MaxBackups = 3
	s.SetRetentionConfig(cfg)
	if got := s.GetRetentionConfig(); got.MaxBackups != 3 {
		t.Fatalf("retention passthrough failed: %d", got.MaxBackups)
	}
}

// TestServiceListDelete verifies List reflects files on disk and Delete removes
// a backup, returning ErrBackupNotFound when absent.
func TestServiceListDelete(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		t.Fatal(err)
	}
	name := "caspaste_backup_2026-06-15_000000.tar.gz"
	if err := os.WriteFile(filepath.Join(backupDir, name), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	s := NewService(dir, dir, backupDir, "dev", nil)

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Filename != name {
		t.Fatalf("List did not return the backup: %+v", list)
	}

	if err := s.Delete(name); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(backupDir, name)); !os.IsNotExist(err) {
		t.Fatal("Delete did not remove file")
	}
	if err := s.Delete("missing.tar.gz"); err != ErrBackupNotFound {
		t.Fatalf("Delete missing: expected ErrBackupNotFound, got %v", err)
	}
}

// TestCopyDirAndMoveFile exercises the recursive copy and move helpers.
func TestCopyDirAndMoveFile(t *testing.T) {
	dir := t.TempDir()
	s := NewService(dir, dir, dir, "dev", nil)

	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "top.txt"), []byte("top"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "nested", "deep.txt"), []byte("deep"), 0644); err != nil {
		t.Fatal(err)
	}

	dstDir := filepath.Join(dir, "dst")
	if err := s.copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dstDir, "nested", "deep.txt")); err != nil || string(b) != "deep" {
		t.Fatalf("copyDir did not copy nested file: %v %q", err, b)
	}

	moveSrc := filepath.Join(dir, "move-src.txt")
	if err := os.WriteFile(moveSrc, []byte("move"), 0644); err != nil {
		t.Fatal(err)
	}
	moveDst := filepath.Join(dir, "move-dst.txt")
	if err := s.moveFile(moveSrc, moveDst); err != nil {
		t.Fatalf("moveFile: %v", err)
	}
	if _, err := os.Stat(moveSrc); !os.IsNotExist(err) {
		t.Fatal("moveFile left source behind")
	}
	if b, _ := os.ReadFile(moveDst); string(b) != "move" {
		t.Fatalf("moveFile content mismatch: %q", b)
	}
}

// TestErrorClassifiers verifies the IsPasswordError/IsVerificationError helpers.
func TestErrorClassifiers(t *testing.T) {
	if !IsPasswordError(ErrInvalidPassword) || !IsPasswordError(ErrPasswordRequired) {
		t.Fatal("IsPasswordError missed a password error")
	}
	if IsPasswordError(ErrVerificationFailed) {
		t.Fatal("IsPasswordError matched a non-password error")
	}
	if !IsVerificationError(ErrChecksumMismatch) || !IsVerificationError(ErrDatabaseIntegrity) {
		t.Fatal("IsVerificationError missed a verification error")
	}
	if IsVerificationError(ErrInvalidPassword) {
		t.Fatal("IsVerificationError matched a password error")
	}
}

// TestRetentionConfigValidation checks SetConfig clamps invalid values to
// defaults per PART 22.
func TestRetentionConfigValidation(t *testing.T) {
	r := NewRetentionService(t.TempDir())
	r.SetConfig(RetentionConfig{MaxBackups: 0, KeepWeekly: -1, KeepMonthly: -5, KeepYearly: -1})
	got := r.GetConfig()
	if got.MaxBackups != 1 || got.KeepWeekly != 0 || got.KeepMonthly != 0 || got.KeepYearly != 0 {
		t.Fatalf("SetConfig did not clamp invalid values: %+v", got)
	}
}
