
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package caspasswd

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// testArgon2Hash is computed once in TestMain to keep the test suite fast.
var testArgon2Hash string

// testBcryptHash is computed once for migration tests.
var testBcryptHash string

const testPassword = "hunter2_correct_horse"

func TestMain(m *testing.M) {
	var err error
	testArgon2Hash, err = HashPassword(testPassword)
	if err != nil {
		panic("TestMain: failed to hash test password: " + err.Error())
	}

	bcryptBytes, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		panic("TestMain: failed to generate bcrypt hash: " + err.Error())
	}
	testBcryptHash = string(bcryptBytes)

	os.Exit(m.Run())
}

func TestHashPassword_Format(t *testing.T) {
	if !strings.HasPrefix(testArgon2Hash, "$argon2id$") {
		t.Errorf("expected hash to start with $argon2id$, got: %q", testArgon2Hash)
	}
}

func TestHashPassword_Unique(t *testing.T) {
	hash2, err := HashPassword(testPassword)
	if err != nil {
		t.Fatalf("HashPassword: unexpected error: %v", err)
	}
	if testArgon2Hash == hash2 {
		t.Error("expected two hashes of the same password to differ (different salts)")
	}
}

func TestHashPassword_Short(t *testing.T) {
	h, err := HashPassword("x")
	if err != nil {
		t.Fatalf("HashPassword short password: unexpected error: %v", err)
	}
	if !strings.HasPrefix(h, "$argon2id$") {
		t.Errorf("expected argon2id hash, got: %q", h)
	}
}

func TestDataCheck_ValidArgon2id(t *testing.T) {
	data := Data{"admin": testArgon2Hash}
	if !data.Check("admin", testPassword) {
		t.Error("Check: expected true for correct argon2id password")
	}
}

func TestDataCheck_WrongPassword(t *testing.T) {
	data := Data{"admin": testArgon2Hash}
	if data.Check("admin", "wrongpassword") {
		t.Error("Check: expected false for wrong password")
	}
}

func TestDataCheck_NonExistentUser(t *testing.T) {
	data := Data{"admin": testArgon2Hash}
	if data.Check("nosuchuser", testPassword) {
		t.Error("Check: expected false for non-existent user")
	}
}

func TestDataCheck_BcryptHash(t *testing.T) {
	data := Data{"legacy": testBcryptHash}
	if !data.Check("legacy", testPassword) {
		t.Error("Check: expected true for correct bcrypt password (migration path)")
	}
}

func TestDataCheck_BcryptWrongPassword(t *testing.T) {
	data := Data{"legacy": testBcryptHash}
	if data.Check("legacy", "wrongpassword") {
		t.Error("Check: expected false for wrong bcrypt password")
	}
}

func TestDataCheck_UnknownHashFormat(t *testing.T) {
	data := Data{"user": "plaintext-password"}
	if data.Check("user", "plaintext-password") {
		t.Error("Check: expected false for unrecognized hash format (plaintext rejected)")
	}
}

func TestDataCheck_MalformedArgon2id(t *testing.T) {
	data := Data{"user": "$argon2id$malformed"}
	if data.Check("user", testPassword) {
		t.Error("Check: expected false for malformed argon2id hash")
	}
}

func TestDataNeedsRehash_Argon2id(t *testing.T) {
	data := Data{"admin": testArgon2Hash}
	if data.NeedsRehash("admin") {
		t.Error("NeedsRehash: expected false for argon2id hash")
	}
}

func TestDataNeedsRehash_Bcrypt(t *testing.T) {
	data := Data{"legacy": testBcryptHash}
	if !data.NeedsRehash("legacy") {
		t.Error("NeedsRehash: expected true for bcrypt hash (needs migration)")
	}
}

func TestDataNeedsRehash_UnknownFormat(t *testing.T) {
	data := Data{"user": "plaintext"}
	if data.NeedsRehash("user") {
		t.Error("NeedsRehash: expected false for unknown format")
	}
}

func TestDataNeedsRehash_NonExistentUser(t *testing.T) {
	data := Data{}
	if data.NeedsRehash("nobody") {
		t.Error("NeedsRehash: expected false for non-existent user")
	}
}

func TestGenerateRandomPassword_Length(t *testing.T) {
	for _, length := range []int{8, 16, 32, 64} {
		p, err := GenerateRandomPassword(length)
		if err != nil {
			t.Fatalf("GenerateRandomPassword(%d): unexpected error: %v", length, err)
		}
		if len(p) != length {
			t.Errorf("GenerateRandomPassword(%d): expected length %d, got %d", length, length, len(p))
		}
	}
}

func TestGenerateRandomPassword_Charset(t *testing.T) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	p, err := GenerateRandomPassword(100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, c := range p {
		if !strings.ContainsRune(charset, c) {
			t.Errorf("GenerateRandomPassword: character %q at position %d not in charset", c, i)
		}
	}
}

func TestGenerateRandomPassword_Unique(t *testing.T) {
	p1, err := GenerateRandomPassword(32)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := GenerateRandomPassword(32)
	if err != nil {
		t.Fatal(err)
	}
	if p1 == p2 {
		t.Error("expected two generated passwords to differ")
	}
}

func TestLoadFile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "admin:" + testArgon2Hash + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	data, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: unexpected error: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("LoadFile: expected 1 user, got %d", len(data))
	}
	if data["admin"] != testArgon2Hash {
		t.Error("LoadFile: unexpected hash value for admin")
	}
}

func TestLoadFile_MultiUser(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "alice:" + testArgon2Hash + "\nbob:" + testBcryptHash + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	data, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: unexpected error: %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("LoadFile: expected 2 users, got %d", len(data))
	}
}

func TestLoadFile_MissingFile(t *testing.T) {
	_, err := LoadFile("/nonexistent/passwd/file")
	if err == nil {
		t.Error("LoadFile: expected error for missing file")
	}
}

func TestLoadFile_MalformedLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	if err := os.WriteFile(path, []byte("malformed-no-colon\n"), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Error("LoadFile: expected error for malformed line")
	}
}

func TestLoadFile_DuplicateUser(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "admin:hash1\nadmin:hash2\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Error("LoadFile: expected error for duplicate user")
	}
}

func TestLoadFile_EmptyLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "\nadmin:" + testArgon2Hash + "\n\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	data, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: unexpected error: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("LoadFile: expected 1 user, got %d", len(data))
	}
}

func TestFileExistsAndHasUsers_NonExistent(t *testing.T) {
	if FileExistsAndHasUsers("/nonexistent/path") {
		t.Error("expected false for non-existent file")
	}
}

func TestFileExistsAndHasUsers_EmptyPath(t *testing.T) {
	if FileExistsAndHasUsers("") {
		t.Error("expected false for empty path")
	}
}

func TestFileExistsAndHasUsers_WithUsers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "admin:" + testArgon2Hash + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	if !FileExistsAndHasUsers(path) {
		t.Error("expected true for file with users")
	}
}

func TestLoadAndCheck_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "admin:" + testArgon2Hash + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	ok, err := LoadAndCheck(path, "admin", testPassword)
	if err != nil {
		t.Fatalf("LoadAndCheck: unexpected error: %v", err)
	}
	if !ok {
		t.Error("LoadAndCheck: expected true for valid credentials")
	}
}

func TestLoadAndCheck_InvalidPassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "admin:" + testArgon2Hash + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	ok, err := LoadAndCheck(path, "admin", "wrongpassword")
	if err != nil {
		t.Fatalf("LoadAndCheck: unexpected error: %v", err)
	}
	if ok {
		t.Error("LoadAndCheck: expected false for wrong password")
	}
}

func TestLoadAndCheckWithRehash_Argon2id(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "admin:" + testArgon2Hash + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	ok, needsRehash, err := LoadAndCheckWithRehash(path, "admin", testPassword)
	if err != nil {
		t.Fatalf("LoadAndCheckWithRehash: unexpected error: %v", err)
	}
	if !ok {
		t.Error("LoadAndCheckWithRehash: expected valid=true")
	}
	if needsRehash {
		t.Error("LoadAndCheckWithRehash: expected needsRehash=false for argon2id hash")
	}
}

func TestLoadAndCheckWithRehash_Bcrypt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "legacy:" + testBcryptHash + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	ok, needsRehash, err := LoadAndCheckWithRehash(path, "legacy", testPassword)
	if err != nil {
		t.Fatalf("LoadAndCheckWithRehash: unexpected error: %v", err)
	}
	if !ok {
		t.Error("LoadAndCheckWithRehash: expected valid=true for bcrypt")
	}
	if !needsRehash {
		t.Error("LoadAndCheckWithRehash: expected needsRehash=true for bcrypt hash")
	}
}

func TestLoadAndCheckWithRehash_Invalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "admin:" + testArgon2Hash + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	ok, needsRehash, err := LoadAndCheckWithRehash(path, "admin", "wrongpassword")
	if err != nil {
		t.Fatalf("LoadAndCheckWithRehash: unexpected error: %v", err)
	}
	if ok {
		t.Error("LoadAndCheckWithRehash: expected valid=false for wrong password")
	}
	if needsRehash {
		t.Error("LoadAndCheckWithRehash: expected needsRehash=false when check failed")
	}
}

func TestRehashPassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")
	content := "user:" + testBcryptHash + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test passwd file: %v", err)
	}

	if err := RehashPassword(path, "user", testPassword); err != nil {
		t.Fatalf("RehashPassword: unexpected error: %v", err)
	}

	data, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile after rehash: %v", err)
	}

	if !strings.HasPrefix(data["user"], "$argon2id$") {
		t.Error("RehashPassword: expected new hash to be argon2id format")
	}
}

func TestGenerateCredentialsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "passwd")

	username, password, err := GenerateCredentialsFile(path)
	if err != nil {
		t.Fatalf("GenerateCredentialsFile: unexpected error: %v", err)
	}
	if username != "admin" {
		t.Errorf("expected username 'admin', got %q", username)
	}
	if len(password) != 16 {
		t.Errorf("expected 16-char password, got %d chars", len(password))
	}

	data, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile after GenerateCredentialsFile: %v", err)
	}
	if !data.Check("admin", password) {
		t.Error("generated password does not verify against stored hash")
	}
}

func TestBruteForceProtection_InitiallyNotBlocked(t *testing.T) {
	bfp := NewBruteForceProtection(5, time.Second)
	ip := net.ParseIP("192.168.1.1")
	if bfp.CheckBlocked(ip) {
		t.Error("expected IP to not be blocked initially")
	}
}

func TestBruteForceProtection_NilIP(t *testing.T) {
	bfp := NewBruteForceProtection(5, time.Second)
	if bfp.CheckBlocked(nil) {
		t.Error("expected CheckBlocked(nil) to return false")
	}
	bfp.RecordFailure(nil)
	bfp.RecordSuccess(nil)
	d := bfp.GetRemainingLockout(nil)
	if d != 0 {
		t.Errorf("expected GetRemainingLockout(nil) = 0, got %v", d)
	}
}

func TestBruteForceProtection_LockAfterMaxAttempts(t *testing.T) {
	const maxAttempts = 3
	bfp := NewBruteForceProtection(maxAttempts, time.Minute)
	ip := net.ParseIP("10.0.0.1")

	for i := 0; i < maxAttempts; i++ {
		bfp.RecordFailure(ip)
	}

	if !bfp.CheckBlocked(ip) {
		t.Error("expected IP to be blocked after max attempts")
	}
}

func TestBruteForceProtection_NotBlockedBeforeMax(t *testing.T) {
	const maxAttempts = 5
	bfp := NewBruteForceProtection(maxAttempts, time.Minute)
	ip := net.ParseIP("10.0.0.2")

	for i := 0; i < maxAttempts-1; i++ {
		bfp.RecordFailure(ip)
	}

	if bfp.CheckBlocked(ip) {
		t.Errorf("expected IP to not be blocked with %d failures (max=%d)", maxAttempts-1, maxAttempts)
	}
}

func TestBruteForceProtection_RecordSuccess_ClearsBlock(t *testing.T) {
	const maxAttempts = 3
	bfp := NewBruteForceProtection(maxAttempts, time.Minute)
	ip := net.ParseIP("10.0.0.3")

	for i := 0; i < maxAttempts; i++ {
		bfp.RecordFailure(ip)
	}
	if !bfp.CheckBlocked(ip) {
		t.Fatal("precondition: IP should be blocked")
	}

	bfp.RecordSuccess(ip)
	if bfp.CheckBlocked(ip) {
		t.Error("expected block to be cleared after RecordSuccess")
	}
}

func TestBruteForceProtection_GetRemainingLockout(t *testing.T) {
	const maxAttempts = 3
	lockout := 10 * time.Minute
	bfp := NewBruteForceProtection(maxAttempts, lockout)
	ip := net.ParseIP("10.0.0.4")

	d := bfp.GetRemainingLockout(ip)
	if d != 0 {
		t.Errorf("expected 0 remaining lockout before any failures, got %v", d)
	}

	for i := 0; i < maxAttempts; i++ {
		bfp.RecordFailure(ip)
	}

	remaining := bfp.GetRemainingLockout(ip)
	if remaining <= 0 {
		t.Error("expected positive remaining lockout after being blocked")
	}
	if remaining > lockout {
		t.Errorf("remaining lockout %v exceeds max lockout %v", remaining, lockout)
	}
}

func TestBruteForceProtection_LockedIPFailureIgnored(t *testing.T) {
	const maxAttempts = 3
	bfp := NewBruteForceProtection(maxAttempts, time.Minute)
	ip := net.ParseIP("10.0.0.5")

	for i := 0; i < maxAttempts; i++ {
		bfp.RecordFailure(ip)
	}
	bfp.RecordFailure(ip)
	bfp.RecordFailure(ip)

	if !bfp.CheckBlocked(ip) {
		t.Error("expected IP to still be blocked after extra failures")
	}
}

func TestBruteForceProtection_Cleanup(t *testing.T) {
	bfp := &BruteForceProtection{
		attempts:      make(map[string]*loginAttempts),
		maxAttempts:   3,
		lockoutTime:   time.Millisecond,
		cleanupPeriod: time.Minute,
		lastCleanup:   time.Now(),
	}

	ip := net.ParseIP("10.0.0.6")
	key := ip.String()

	bfp.attempts[key] = &loginAttempts{
		count:       2,
		lockedUntil: time.Now().Add(-time.Second),
	}

	bfp.cleanup()

	bfp.mu.RLock()
	_, exists := bfp.attempts[key]
	bfp.mu.RUnlock()

	if exists {
		t.Error("expected expired entry to be cleaned up")
	}
}

func TestBruteForceProtection_DifferentIPs(t *testing.T) {
	const maxAttempts = 3
	bfp := NewBruteForceProtection(maxAttempts, time.Minute)

	ip1 := net.ParseIP("10.0.1.1")
	ip2 := net.ParseIP("10.0.1.2")

	for i := 0; i < maxAttempts; i++ {
		bfp.RecordFailure(ip1)
	}

	if !bfp.CheckBlocked(ip1) {
		t.Error("expected ip1 to be blocked")
	}
	if bfp.CheckBlocked(ip2) {
		t.Error("expected ip2 to remain unblocked")
	}
}
