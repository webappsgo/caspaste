
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package validation

import (
	"os"
	"testing"
)

func TestParseBool_Truthy(t *testing.T) {
	truthy := []string{
		"1", "yes", "true", "enable", "enabled", "on",
		"yep", "yup", "yeah", "affirmative", "aye",
		"si", "oui", "da", "hai", "totally", "sure",
		"ok", "okay", "accept", "allow", "grant",
		"y", "t", "active",
		"YES", "TRUE", "Enable",
	}

	for _, s := range truthy {
		val, wasSet := ParseBool(s)
		if !wasSet {
			t.Errorf("ParseBool(%q): expected wasSet=true", s)
		}
		if !val {
			t.Errorf("ParseBool(%q): expected val=true", s)
		}
	}
}

func TestParseBool_Falsey(t *testing.T) {
	falsey := []string{
		"0", "no", "false", "disable", "disabled", "off",
		"nope", "nah", "nay", "negative", "nein",
		"non", "niet", "iie", "lie", "noway", "never",
		"deny", "reject", "block", "revoke",
		"n", "f", "inactive",
		"NO", "FALSE", "Disable",
	}

	for _, s := range falsey {
		val, wasSet := ParseBool(s)
		if !wasSet {
			t.Errorf("ParseBool(%q): expected wasSet=true", s)
		}
		if val {
			t.Errorf("ParseBool(%q): expected val=false", s)
		}
	}
}

func TestParseBool_Empty(t *testing.T) {
	val, wasSet := ParseBool("")
	if wasSet {
		t.Error("ParseBool(\"\"): expected wasSet=false")
	}
	if val {
		t.Error("ParseBool(\"\"): expected val=false")
	}
}

func TestParseBool_Unknown(t *testing.T) {
	unknowns := []string{"maybe", "unknown", "2", "tRuE_ish", "yessir"}
	for _, s := range unknowns {
		val, wasSet := ParseBool(s)
		if wasSet {
			t.Errorf("ParseBool(%q): expected wasSet=false for unknown value", s)
		}
		if val {
			t.Errorf("ParseBool(%q): expected val=false for unknown value", s)
		}
	}
}

func TestParseBool_Whitespace(t *testing.T) {
	val, wasSet := ParseBool("  true  ")
	if !wasSet {
		t.Error("expected wasSet=true for trimmed truthy value")
	}
	if !val {
		t.Error("expected val=true for trimmed truthy value")
	}
}

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"yes", true},
		{"true", true},
		{"1", true},
		{"no", false},
		{"false", false},
		{"0", false},
		{"", false},
		{"unknown", false},
	}

	for _, tc := range tests {
		got := IsTruthy(tc.in)
		if got != tc.want {
			t.Errorf("IsTruthy(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsFalsey(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"no", true},
		{"false", true},
		{"0", true},
		{"yes", false},
		{"true", false},
		{"1", false},
		{"", false},
		{"unknown", false},
	}

	for _, tc := range tests {
		got := IsFalsey(tc.in)
		if got != tc.want {
			t.Errorf("IsFalsey(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestDetectDriver(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"sqlite:///path/to/db.sqlite", "sqlite", false},
		{"sqlite3:///path/to/db.sqlite", "sqlite", false},
		{"postgres://user:pass@host:5432/db", "postgres", false},
		{"postgresql://user:pass@host:5432/db", "postgres", false},
		{"mysql://user:pass@host:3306/db", "mysql", false},
		{"mariadb://user:pass@host:3306/db", "mysql", false},
		{"libsql://host/db", "libsql", false},
		{"file:/path/to/db.db", "libsql", false},
		{"/path/to/database.db", "sqlite", false},
		{"relative/path/db.db", "sqlite", false},
		{"./path/db.sqlite", "sqlite", false},
		// Contains "/" so falls through to path-based SQLite detection
		{"unknownscheme://host/db", "sqlite", false},
		// No "/" and no ".db" suffix → error
		{"noslash", "", true},
	}

	for _, tc := range tests {
		got, err := DetectDriver(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("DetectDriver(%q): expected error, got driver=%q", tc.in, got)
			}
		} else {
			if err != nil {
				t.Errorf("DetectDriver(%q): unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("DetectDriver(%q) = %q, want %q", tc.in, got, tc.want)
			}
		}
	}
}

func TestDetectDriver_CaseInsensitive(t *testing.T) {
	got, err := DetectDriver("SQLITE:///path.db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sqlite" {
		t.Fatalf("expected sqlite, got %q", got)
	}
}

func TestNormalizeDriver(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"mariadb", "mysql"},
		{"maria", "mysql"},
		{"sqlite3", "sqlite"},
		{"postgresql", "postgres"},
		{"turso", "libsql"},
		{"mysql", "mysql"},
		{"postgres", "postgres"},
		{"sqlite", "sqlite"},
		{"libsql", "libsql"},
		{"MYSQL", "mysql"},
		{"Postgres", "postgres"},
	}

	for _, tc := range tests {
		got := NormalizeDriver(tc.in)
		if got != tc.want {
			t.Errorf("NormalizeDriver(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeConnectionString(t *testing.T) {
	tests := []struct {
		driver string
		source string
		want   string
	}{
		{"sqlite", "sqlite:///path/to/db.sqlite", "/path/to/db.sqlite"},
		{"sqlite", "sqlite3:///path/to/db.sqlite", "/path/to/db.sqlite"},
		{"sqlite", "/path/to/db.sqlite", "/path/to/db.sqlite"},
		{"postgres", "postgres://user:pass@host:5432/db", "postgres://user:pass@host:5432/db"},
		{"libsql", "libsql://host/db", "libsql://host/db"},
		{"libsql", "file:/local.db", "file:/local.db"},
		{"mysql", "mysql://user:pass@host:3306/mydb", "user:pass@tcp(host:3306)/mydb"},
		{"mysql", "mariadb://user:pass@host:3306/mydb", "user:pass@tcp(host:3306)/mydb"},
		{"mysql", "user:pass@tcp(host:3306)/mydb", "user:pass@tcp(host:3306)/mydb"},
		{"other", "some://connection", "some://connection"},
	}

	for _, tc := range tests {
		got := NormalizeConnectionString(tc.driver, tc.source)
		if got != tc.want {
			t.Errorf("NormalizeConnectionString(%q, %q) = %q, want %q", tc.driver, tc.source, got, tc.want)
		}
	}
}

func TestValidateFQDN_Valid(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.example.com",
		"deep.sub.example.com",
		"example.org",
		"my-server.example.net",
		"xn--nxasmq6b.com",
	}

	for _, fqdn := range valid {
		err := ValidateFQDN(fqdn)
		if err != nil {
			t.Errorf("ValidateFQDN(%q): unexpected error: %v", fqdn, err)
		}
	}
}

func TestValidateFQDN_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"localhost",
		"localhost.localdomain",
		"localdomain",
		"local",
		"192.168.1.1",
		"127.0.0.1",
		".example.com",
		"example.com.",
		"INVALID SPACE.com",
	}

	for _, fqdn := range invalid {
		err := ValidateFQDN(fqdn)
		if err == nil {
			t.Errorf("ValidateFQDN(%q): expected error, got nil", fqdn)
		}
	}
}

func TestValidateFQDN_WithPort(t *testing.T) {
	err := ValidateFQDN("example.com:8080")
	if err != nil {
		t.Errorf("ValidateFQDN with port: unexpected error: %v", err)
	}
}

func TestValidateFQDN_NoDot(t *testing.T) {
	err := ValidateFQDN("nodot")
	if err == nil {
		t.Error("ValidateFQDN(\"nodot\"): expected error for missing dot")
	}
}

func TestValidateFQDN_IPAddress(t *testing.T) {
	err := ValidateFQDN("192.168.1.1")
	if err == nil {
		t.Error("ValidateFQDN(\"192.168.1.1\"): expected error for IP address")
	}
}

func TestValidateEmail_Valid(t *testing.T) {
	valid := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.org",
		"admin@sub.example.net",
	}

	for _, email := range valid {
		err := ValidateEmail(email)
		if err != nil {
			t.Errorf("ValidateEmail(%q): unexpected error: %v", email, err)
		}
	}
}

func TestValidateEmail_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"notanemail",
		"@example.com",
		"user@",
		"user@@example.com",
		"user@localhost",
	}

	for _, email := range invalid {
		err := ValidateEmail(email)
		if err == nil {
			t.Errorf("ValidateEmail(%q): expected error, got nil", email)
		}
	}
}

func TestDetermineFQDN_FromConfig(t *testing.T) {
	got, err := DetermineFQDN("", "example.com")
	if err != nil {
		t.Fatalf("DetermineFQDN with config: unexpected error: %v", err)
	}
	if got != "example.com" {
		t.Errorf("expected %q, got %q", "example.com", got)
	}
}

func TestDetermineFQDN_FromConfigWithPort(t *testing.T) {
	got, err := DetermineFQDN("", "example.com:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "example.com" {
		t.Errorf("expected %q (port stripped), got %q", "example.com", got)
	}
}

func TestDetermineFQDN_FromProxy(t *testing.T) {
	got, err := DetermineFQDN("proxy.example.com", "")
	if err != nil {
		t.Fatalf("DetermineFQDN with proxy: unexpected error: %v", err)
	}
	if got != "proxy.example.com" {
		t.Errorf("expected %q, got %q", "proxy.example.com", got)
	}
}

func TestDetermineFQDN_FromProxyWithPort(t *testing.T) {
	got, err := DetermineFQDN("proxy.example.com:443", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "proxy.example.com" {
		t.Errorf("expected %q (port stripped), got %q", "proxy.example.com", got)
	}
}

func TestDetermineFQDN_ConfigTakesPriorityOverProxy(t *testing.T) {
	got, err := DetermineFQDN("proxy.example.com", "config.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "config.example.com" {
		t.Errorf("expected config %q to win over proxy, got %q", "config.example.com", got)
	}
}

func TestValidateTLSCerts_MissingCertFile(t *testing.T) {
	err := ValidateTLSCerts("/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Error("expected error for missing certificate file")
	}
}

func TestValidateTLSCerts_MissingKeyFile(t *testing.T) {
	certFile := t.TempDir() + "/cert.pem"
	if err := createDummyFile(certFile); err != nil {
		t.Fatalf("failed to create temp cert file: %v", err)
	}
	err := ValidateTLSCerts(certFile, "/nonexistent/key.pem")
	if err == nil {
		t.Error("expected error for missing key file")
	}
}

func TestFindLetsEncryptCerts_MissingDirectory(t *testing.T) {
	_, err := FindLetsEncryptCerts("example.com")
	if err == nil {
		t.Error("expected error when /etc/letsencrypt/live does not exist")
	}
}

func createDummyFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}
