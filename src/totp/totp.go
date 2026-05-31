
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
)

// TOTP constants
const (
	// seconds
	DefaultPeriod     = 30
	DefaultDigits     = 6
	// bytes
	DefaultSecretSize = 20
)

// TOTPSetup contains data needed for TOTP setup
type TOTPSetup struct {
	// Base32-encoded secret
	Secret string `json:"secret"`
	// otpauth:// URL for QR code generation
	QRCodeURL string `json:"qr_code_url"`
	Issuer    string `json:"issuer"`
	Account   string `json:"account"`
}

// GenerateSecret generates a new TOTP secret and returns setup info
func GenerateSecret(issuer, account string) (*TOTPSetup, error) {
	// Generate random bytes
	secretBytes := make([]byte, DefaultSecretSize)
	_, err := rand.Read(secretBytes)
	if err != nil {
		return nil, err
	}

	// Encode to base32
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)

	// Create otpauth URL for QR code
	qrURL := fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&digits=%d&period=%d",
		issuer, account, secret, issuer, DefaultDigits, DefaultPeriod,
	)

	return &TOTPSetup{
		Secret:    secret,
		QRCodeURL: qrURL,
		Issuer:    issuer,
		Account:   account,
	}, nil
}

// Verify verifies a TOTP code against a secret
func Verify(secret, code string) bool {
	// Decode secret
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}

	// Get current time step
	now := time.Now().Unix()
	counter := uint64(now / DefaultPeriod)

	// Allow for time drift (check current, previous, and next time step)
	for i := -1; i <= 1; i++ {
		expectedCode := generateTOTP(secretBytes, counter+uint64(i), DefaultDigits)
		if expectedCode == code {
			return true
		}
	}

	return false
}

// generateTOTP generates a TOTP code for a given counter
func generateTOTP(secret []byte, counter uint64, digits int) string {
	// Convert counter to bytes (big-endian)
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, counter)

	// HMAC-SHA1
	h := hmac.New(sha1.New, secret)
	h.Write(counterBytes)
	hash := h.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	code := int64(hash[offset]&0x7f)<<24 |
		int64(hash[offset+1]&0xff)<<16 |
		int64(hash[offset+2]&0xff)<<8 |
		int64(hash[offset+3]&0xff)

	// Get the last N digits
	code = code % int64(math.Pow10(digits))

	// Pad with leading zeros if necessary
	return fmt.Sprintf("%0*d", digits, code)
}

// GenerateRecoveryKeys generates a set of recovery keys per PART 34
// Format: {8-hex-chars}-{4-hex-chars} (e.g., a1b2c3d4-e5f6)
func GenerateRecoveryKeys(count int) ([]string, error) {
	keys := make([]string, count)

	for i := 0; i < count; i++ {
		// Generate 6 random bytes (12 hex chars)
		bytes := make([]byte, 6)
		_, err := rand.Read(bytes)
		if err != nil {
			return nil, err
		}

		// Format as 8-4 hex chars
		hex := fmt.Sprintf("%x", bytes)
		keys[i] = hex[:8] + "-" + hex[8:]
	}

	return keys, nil
}

// HashRecoveryKey hashes a recovery key for storage using SHA-256
func HashRecoveryKey(key string) string {
	// Normalize: lowercase, remove spaces
	key = strings.ToLower(strings.ReplaceAll(key, " ", ""))
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// VerifyRecoveryKeyFormat checks if a recovery key has the correct format
func VerifyRecoveryKeyFormat(key string) bool {
	// Format: 8 hex chars, hyphen, 4 hex chars (e.g., a1b2c3d4-e5f6)
	key = strings.ToLower(strings.TrimSpace(key))
	if len(key) != 13 {
		return false
	}
	if key[8] != '-' {
		return false
	}

	// Check all characters are hex
	for i, c := range key {
		if i == 8 {
			// Skip hyphen
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}

	return true
}

// NormalizeRecoveryKey normalizes a recovery key for comparison
func NormalizeRecoveryKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), " ", ""))
}

// GetCurrentTimeStep returns the current TOTP time step
func GetCurrentTimeStep() int64 {
	return time.Now().Unix() / DefaultPeriod
}

// GetRemainingSeconds returns seconds until the current code expires
func GetRemainingSeconds() int {
	return DefaultPeriod - int(time.Now().Unix()%DefaultPeriod)
}

// GenerateCurrentCode generates the current TOTP code for a secret
// This is useful for testing or displaying the expected code
func GenerateCurrentCode(secret string) (string, error) {
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}

	counter := uint64(time.Now().Unix() / DefaultPeriod)
	return generateTOTP(secretBytes, counter, DefaultDigits), nil
}
