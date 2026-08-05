package backup

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestEncryptDecryptData round-trips in-memory encryption and verifies a wrong
// password fails with ErrInvalidPassword.
func TestEncryptDecryptData(t *testing.T) {
	e := NewEncryptionService()
	plaintext := []byte("the quick brown fox jumps over the lazy dog")

	enc, err := e.EncryptData(plaintext, "correct horse battery staple")
	if err != nil {
		t.Fatalf("EncryptData: %v", err)
	}
	if bytes.Equal(enc, plaintext) {
		t.Fatal("ciphertext equals plaintext")
	}

	dec, err := e.DecryptData(enc, "correct horse battery staple")
	if err != nil {
		t.Fatalf("DecryptData: %v", err)
	}
	if !bytes.Equal(dec, plaintext) {
		t.Fatalf("round-trip mismatch: got %q", dec)
	}

	if _, err := e.DecryptData(enc, "wrong password"); err != ErrInvalidPassword {
		t.Fatalf("wrong password: expected ErrInvalidPassword, got %v", err)
	}

	if _, err := e.DecryptData([]byte("tooshort"), "x"); err != ErrInvalidBackupFile {
		t.Fatalf("short data: expected ErrInvalidBackupFile, got %v", err)
	}
}

// TestEncryptDecryptFile round-trips file encryption and checks the wrong
// password path.
func TestEncryptDecryptFile(t *testing.T) {
	e := NewEncryptionService()
	dir := t.TempDir()
	src := filepath.Join(dir, "data.tar.gz")
	content := []byte("archive contents here")
	if err := os.WriteFile(src, content, 0600); err != nil {
		t.Fatal(err)
	}

	encPath, err := e.EncryptFile(src, "pw")
	if err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}
	if filepath.Ext(encPath) != ".enc" {
		t.Fatalf("expected .enc output, got %q", encPath)
	}

	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0700); err != nil {
		t.Fatal(err)
	}
	decPath, err := e.DecryptFile(encPath, "pw", outDir)
	if err != nil {
		t.Fatalf("DecryptFile: %v", err)
	}
	got, err := os.ReadFile(decPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("file round-trip mismatch: got %q", got)
	}

	if _, err := e.DecryptFile(encPath, "bad", outDir); err != ErrInvalidPassword {
		t.Fatalf("wrong password: expected ErrInvalidPassword, got %v", err)
	}
}

// TestHashPassword verifies determinism and hex-length of the SHA-256 hash.
func TestHashPassword(t *testing.T) {
	h1 := HashPassword("secret")
	h2 := HashPassword("secret")
	if h1 != h2 {
		t.Fatal("HashPassword not deterministic")
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(h1))
	}
	if HashPassword("secret") == HashPassword("other") {
		t.Fatal("distinct passwords hashed equal")
	}
}

// TestGenerateRandomBytes verifies length and non-repetition.
func TestGenerateRandomBytes(t *testing.T) {
	b, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("GenerateRandomBytes: %v", err)
	}
	if len(b) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(b))
	}
	b2, _ := GenerateRandomBytes(32)
	if bytes.Equal(b, b2) {
		t.Fatal("two random reads returned identical bytes")
	}
}
