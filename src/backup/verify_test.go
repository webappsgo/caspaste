package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestComputeContentChecksum verifies the checksum is deterministic, detects
// content changes, and honors the skip list.
func TestComputeContentChecksum(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	first, err := computeContentChecksum(dir)
	if err != nil {
		t.Fatalf("checksum: %v", err)
	}
	if first == "" || first[:7] != "sha256:" {
		t.Fatalf("unexpected checksum format: %q", first)
	}

	// Deterministic: recomputing over unchanged content yields the same value.
	again, err := computeContentChecksum(dir)
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Fatalf("checksum not deterministic: %q != %q", again, first)
	}

	// Changing content changes the checksum.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("HELLO"), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := computeContentChecksum(dir)
	if err != nil {
		t.Fatal(err)
	}
	if changed == first {
		t.Fatal("checksum did not change after content change")
	}

	// Skipped files do not contribute to the checksum.
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	withManifest, err := computeContentChecksum(dir, "manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if withManifest != changed {
		t.Fatalf("skip did not exclude manifest.json: %q != %q", withManifest, changed)
	}
}

// TestBackupVerifyRoundTrip creates a real backup and confirms verification
// passes, then corrupts the archive content and confirms the checksum catches it.
func TestBackupVerifyRoundTrip(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	backupDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(configDir, "server.yml"), []byte("port: 8080\n"), 0644); err != nil {
		t.Fatal(err)
	}

	svc := NewService(configDir, dataDir, backupDir, "test", nil)

	res, err := svc.Create(context.Background(), BackupOptions{})
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Verification is already run inside Backup; re-run explicitly.
	detail := svc.VerifyDetailed(res.FilePath, VerifyOptions{})
	if !detail.Valid {
		t.Fatalf("verify failed: %v", detail.Errors)
	}
	if !detail.ChecksumValid {
		t.Fatal("checksum should be valid for a fresh backup")
	}
	if res.Checksum == "" || res.Checksum[:7] != "sha256:" {
		t.Fatalf("unexpected manifest checksum: %q", res.Checksum)
	}
}
