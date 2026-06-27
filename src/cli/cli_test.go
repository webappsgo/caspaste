
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package cli

import (
	"os"
	"testing"
	"time"
)

func TestGetEnvVar_Found(t *testing.T) {
	os.Setenv("CASPB_DB_DRIVER", "sqlite")
	defer os.Unsetenv("CASPB_DB_DRIVER")

	val, found := getEnvVar("db-driver")
	if !found {
		t.Fatal("expected found=true")
	}
	if val != "sqlite" {
		t.Fatalf("expected %q, got %q", "sqlite", val)
	}
}

func TestGetEnvVar_NotFound(t *testing.T) {
	os.Unsetenv("CASPB_MISSING_TEST_CLI_VAR")

	val, found := getEnvVar("missing-test-cli-var")
	if found {
		t.Fatal("expected found=false")
	}
	if val != "" {
		t.Fatalf("expected empty string, got %q", val)
	}
}

func TestGetEnvVar_UnderscoreConversion(t *testing.T) {
	os.Setenv("CASPB_SOME_TEST_FLAG", "flagvalue")
	defer os.Unsetenv("CASPB_SOME_TEST_FLAG")

	val, found := getEnvVar("some-test-flag")
	if !found {
		t.Fatal("expected found=true")
	}
	if val != "flagvalue" {
		t.Fatalf("expected %q, got %q", "flagvalue", val)
	}
}

func TestNormalizeFlag(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"--debug", "-debug"},
		{"--port", "-port"},
		{"--host-name", "-host-name"},
		{"-port", "-port"},
		{"-debug", "-debug"},
		{"noflag", "noflag"},
		{"--", "-"},
	}

	for _, tc := range tests {
		got := normalizeFlag(tc.in)
		if got != tc.want {
			t.Errorf("normalizeFlag(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestWriteVar_String(t *testing.T) {
	var s string
	err := writeVar("hello", &s, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "hello" {
		t.Fatalf("expected %q, got %q", "hello", s)
	}
}

func TestWriteVar_StringWithPreHook(t *testing.T) {
	var s string
	hook := func(val string) (string, error) {
		return val + "_hooked", nil
	}
	err := writeVar("input", &s, hook)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "input_hooked" {
		t.Fatalf("expected %q, got %q", "input_hooked", s)
	}
}

func TestWriteVar_Int(t *testing.T) {
	var i int
	err := writeVar("42", &i, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i != 42 {
		t.Fatalf("expected 42, got %d", i)
	}
}

func TestWriteVar_IntInvalid(t *testing.T) {
	var i int
	err := writeVar("notanint", &i, nil)
	if err == nil {
		t.Fatal("expected error for invalid int")
	}
}

func TestWriteVar_Bool(t *testing.T) {
	var b bool
	err := writeVar("anything", &b, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !b {
		t.Fatal("expected bool to be set to true")
	}
}

func TestWriteVar_Uint(t *testing.T) {
	var u uint
	err := writeVar("100", &u, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != 100 {
		t.Fatalf("expected 100, got %d", u)
	}
}

func TestWriteVar_UintInvalid(t *testing.T) {
	var u uint
	err := writeVar("notauint", &u, nil)
	if err == nil {
		t.Fatal("expected error for invalid uint")
	}
}

func TestWriteVar_Duration(t *testing.T) {
	var d time.Duration
	err := writeVar("1h", &d, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != time.Hour {
		t.Fatalf("expected 1h, got %v", d)
	}
}

func TestWriteVar_DurationInvalid(t *testing.T) {
	var d time.Duration
	err := writeVar("notaduration", &d, nil)
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestWriteVar_UnknownTypePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unknown type")
		}
	}()
	var f float64
	writeVar("1.0", &f, nil)
}

func TestNew(t *testing.T) {
	c := New("2.0.0")
	if c == nil {
		t.Fatal("expected non-nil CLI")
	}
	if c.version != "2.0.0" {
		t.Fatalf("expected version %q, got %q", "2.0.0", c.version)
	}
	if len(c.vars) != 0 {
		t.Fatalf("expected empty vars slice, got %d", len(c.vars))
	}
}

func TestAddStringVar(t *testing.T) {
	c := New("1.0.0")
	val := c.AddStringVar("host", "localhost", "Host to bind to", nil)
	if val == nil {
		t.Fatal("expected non-nil *string")
	}
	if *val != "localhost" {
		t.Fatalf("expected %q, got %q", "localhost", *val)
	}
	if len(c.vars) != 1 {
		t.Fatalf("expected 1 var registered, got %d", len(c.vars))
	}
}

func TestAddStringVar_WithPreHook(t *testing.T) {
	c := New("1.0.0")
	hook := func(val string) (string, error) {
		return val + ":3000", nil
	}
	val := c.AddStringVar("addr", "localhost", "server address", &FlagOptions{PreHook: hook})
	if *val != "localhost:3000" {
		t.Fatalf("expected %q, got %q", "localhost:3000", *val)
	}
}

func TestAddBoolVar(t *testing.T) {
	c := New("1.0.0")
	val := c.AddBoolVar("verbose", "Enable verbose output")
	if val == nil {
		t.Fatal("expected non-nil *bool")
	}
	if *val {
		t.Fatal("expected bool default to be false")
	}
}

func TestAddIntVar(t *testing.T) {
	c := New("1.0.0")
	val := c.AddIntVar("port", 8080, "Port number", nil)
	if val == nil {
		t.Fatal("expected non-nil *int")
	}
	if *val != 8080 {
		t.Fatalf("expected 8080, got %d", *val)
	}
}

func TestAddUintVar(t *testing.T) {
	c := New("1.0.0")
	val := c.AddUintVar("workers", 4, "Number of worker threads", nil)
	if val == nil {
		t.Fatal("expected non-nil *uint")
	}
	if *val != 4 {
		t.Fatalf("expected 4, got %d", *val)
	}
}

func TestAddDurationVar(t *testing.T) {
	c := New("1.0.0")
	val := c.AddDurationVar("timeout", "30m", "Request timeout duration", nil)
	if val == nil {
		t.Fatal("expected non-nil *time.Duration")
	}
	if *val != 30*time.Minute {
		t.Fatalf("expected 30m, got %v", *val)
	}
}

func TestAddDurationVar_InvalidDefaultPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid duration default")
		}
	}()
	c := New("1.0.0")
	c.AddDurationVar("bad", "notaduration", "bad duration", nil)
}

func TestAddVar_EmptyNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty variable name")
		}
	}()
	c := New("1.0.0")
	c.addVar("", new(string), "", "usage text", nil)
}

func TestAddVar_EmptyUsagePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty usage string")
		}
	}()
	c := New("1.0.0")
	c.addVar("myvar", new(string), "default", "", nil)
}

func TestAddVar_FlagNameFormat(t *testing.T) {
	c := New("1.0.0")
	c.addVar("myvar", new(string), "default", "some usage", nil)
	if len(c.vars) != 1 {
		t.Fatalf("expected 1 var, got %d", len(c.vars))
	}
	if c.vars[0].cliFlagName != "-myvar" {
		t.Fatalf("expected cliFlagName=%q, got %q", "-myvar", c.vars[0].cliFlagName)
	}
}

func TestParse_StringFlag(t *testing.T) {
	c := New("1.0.0")
	host := c.AddStringVar("clitt-host", "localhost", "host flag", nil)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog", "-clitt-host", "example.com"}

	c.Parse()

	if *host != "example.com" {
		t.Errorf("expected %q, got %q", "example.com", *host)
	}
}

func TestParse_DoubleDashNormalized(t *testing.T) {
	c := New("1.0.0")
	host := c.AddStringVar("clitt-host2", "localhost", "host flag 2", nil)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog", "--clitt-host2", "example.com"}

	c.Parse()

	if *host != "example.com" {
		t.Errorf("expected %q, got %q", "example.com", *host)
	}
}

func TestParse_BoolFlag(t *testing.T) {
	c := New("1.0.0")
	debug := c.AddBoolVar("clitt-debug", "enable debug mode")

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog", "-clitt-debug"}

	c.Parse()

	if !*debug {
		t.Error("expected debug=true after flag set")
	}
}

func TestParse_IntFlag(t *testing.T) {
	c := New("1.0.0")
	port := c.AddIntVar("clitt-port", 8080, "port number", nil)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog", "-clitt-port", "9090"}

	c.Parse()

	if *port != 9090 {
		t.Errorf("expected 9090, got %d", *port)
	}
}

func TestParse_UintFlag(t *testing.T) {
	c := New("1.0.0")
	workers := c.AddUintVar("clitt-workers", 2, "worker count", nil)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog", "-clitt-workers", "8"}

	c.Parse()

	if *workers != 8 {
		t.Errorf("expected 8, got %d", *workers)
	}
}

func TestParse_DurationFlag(t *testing.T) {
	c := New("1.0.0")
	timeout := c.AddDurationVar("clitt-timeout", "5m", "request timeout", nil)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog", "-clitt-timeout", "1h"}

	c.Parse()

	if *timeout != time.Hour {
		t.Errorf("expected 1h, got %v", *timeout)
	}
}

func TestParse_EnvVarRead(t *testing.T) {
	os.Setenv("CASPB_CLITT_ENVHOST", "fromenv")
	defer os.Unsetenv("CASPB_CLITT_ENVHOST")

	c := New("1.0.0")
	host := c.AddStringVar("clitt-envhost", "default", "env var host", nil)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog"}

	c.Parse()

	if *host != "fromenv" {
		t.Errorf("expected %q from env, got %q", "fromenv", *host)
	}
}

func TestParse_CLIOverridesEnv(t *testing.T) {
	os.Setenv("CASPB_CLITT_OVERRIDE", "envvalue")
	defer os.Unsetenv("CASPB_CLITT_OVERRIDE")

	c := New("1.0.0")
	val := c.AddStringVar("clitt-override", "default", "overridable flag", nil)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog", "-clitt-override", "clivalue"}

	c.Parse()

	if *val != "clivalue" {
		t.Errorf("expected CLI value %q to override env, got %q", "clivalue", *val)
	}
}

func TestParse_DefaultsRetainedWhenNotSet(t *testing.T) {
	c := New("1.0.0")
	host := c.AddStringVar("clitt-default", "mydefault", "host with default", nil)
	port := c.AddIntVar("clitt-defaultport", 1234, "port with default", nil)

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog"}

	c.Parse()

	if *host != "mydefault" {
		t.Errorf("expected default %q, got %q", "mydefault", *host)
	}
	if *port != 1234 {
		t.Errorf("expected default port 1234, got %d", *port)
	}
}

func TestParse_MultipleFlags(t *testing.T) {
	c := New("1.0.0")
	host := c.AddStringVar("mf-host", "localhost", "host", nil)
	port := c.AddIntVar("mf-port", 80, "port", nil)
	debug := c.AddBoolVar("mf-debug", "debug mode")

	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"prog", "-mf-host", "myhost", "-mf-port", "443", "-mf-debug"}

	c.Parse()

	if *host != "myhost" {
		t.Errorf("host: expected %q, got %q", "myhost", *host)
	}
	if *port != 443 {
		t.Errorf("port: expected 443, got %d", *port)
	}
	if !*debug {
		t.Error("debug: expected true")
	}
}
