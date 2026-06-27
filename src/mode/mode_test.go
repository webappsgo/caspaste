
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package mode

import (
	"os"
	"testing"
)

// resetState restores the package-level variables to their defaults after each test.
func resetState(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		currentMode = Production
		debugEnabled = false
	})
}

func TestAppModeString(t *testing.T) {
	tests := []struct {
		mode AppMode
		want string
	}{
		{Production, "production"},
		{Development, "development"},
		// Any other int value should default to "production"
		{AppMode(99), "production"},
	}

	for _, tc := range tests {
		got := tc.mode.String()
		if got != tc.want {
			t.Errorf("AppMode(%d).String() = %q, want %q", int(tc.mode), got, tc.want)
		}
	}
}

func TestSetAppMode_Production(t *testing.T) {
	resetState(t)

	SetAppMode("production")
	if GetCurrentAppMode() != Production {
		t.Errorf("expected Production, got %v", GetCurrentAppMode())
	}
}

func TestSetAppMode_ProductionShorthand(t *testing.T) {
	resetState(t)

	SetAppMode("prod")
	if GetCurrentAppMode() != Production {
		t.Errorf("expected Production for 'prod', got %v", GetCurrentAppMode())
	}
}

func TestSetAppMode_Development(t *testing.T) {
	resetState(t)

	SetAppMode("development")
	if GetCurrentAppMode() != Development {
		t.Errorf("expected Development, got %v", GetCurrentAppMode())
	}
}

func TestSetAppMode_DevelopmentShorthand(t *testing.T) {
	resetState(t)

	SetAppMode("dev")
	if GetCurrentAppMode() != Development {
		t.Errorf("expected Development for 'dev', got %v", GetCurrentAppMode())
	}
}

func TestSetAppMode_CaseInsensitive(t *testing.T) {
	resetState(t)

	SetAppMode("DEVELOPMENT")
	if GetCurrentAppMode() != Development {
		t.Errorf("expected Development for 'DEVELOPMENT', got %v", GetCurrentAppMode())
	}
}

func TestSetAppMode_Unknown(t *testing.T) {
	resetState(t)

	SetAppMode("staging")
	if GetCurrentAppMode() != Production {
		t.Errorf("expected Production for unknown mode 'staging', got %v", GetCurrentAppMode())
	}
}

func TestSet_Alias(t *testing.T) {
	resetState(t)

	Set("development")
	if GetCurrentAppMode() != Development {
		t.Errorf("Set alias: expected Development, got %v", GetCurrentAppMode())
	}
}

func TestIsAppModeDev(t *testing.T) {
	resetState(t)

	SetAppMode("production")
	if IsAppModeDev() {
		t.Error("expected IsAppModeDev=false in production mode")
	}

	SetAppMode("development")
	if !IsAppModeDev() {
		t.Error("expected IsAppModeDev=true in development mode")
	}
}

func TestIsAppModeProd(t *testing.T) {
	resetState(t)

	SetAppMode("development")
	if IsAppModeProd() {
		t.Error("expected IsAppModeProd=false in development mode")
	}

	SetAppMode("production")
	if !IsAppModeProd() {
		t.Error("expected IsAppModeProd=true in production mode")
	}
}

func TestSetDebugEnabled(t *testing.T) {
	resetState(t)

	if IsDebugEnabled() {
		t.Error("expected debug disabled by default")
	}

	SetDebugEnabled(true)
	if !IsDebugEnabled() {
		t.Error("expected debug enabled after SetDebugEnabled(true)")
	}

	SetDebugEnabled(false)
	if IsDebugEnabled() {
		t.Error("expected debug disabled after SetDebugEnabled(false)")
	}
}

func TestSetDebug_Alias(t *testing.T) {
	resetState(t)

	SetDebug(true)
	if !IsDebugEnabled() {
		t.Error("SetDebug alias: expected debug enabled")
	}
}

func TestGetAppModeString_Production(t *testing.T) {
	resetState(t)

	SetAppMode("production")
	SetDebugEnabled(false)
	got := GetAppModeString()
	if got != "production" {
		t.Errorf("expected %q, got %q", "production", got)
	}
}

func TestGetAppModeString_Development(t *testing.T) {
	resetState(t)

	SetAppMode("development")
	SetDebugEnabled(false)
	got := GetAppModeString()
	if got != "development" {
		t.Errorf("expected %q, got %q", "development", got)
	}
}

func TestGetAppModeString_WithDebug(t *testing.T) {
	resetState(t)

	SetAppMode("production")
	SetDebugEnabled(true)
	got := GetAppModeString()
	want := "production [debugging]"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestGetAppModeString_DevWithDebug(t *testing.T) {
	resetState(t)

	SetAppMode("development")
	SetDebugEnabled(true)
	got := GetAppModeString()
	want := "development [debugging]"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestFromEnv_Mode(t *testing.T) {
	resetState(t)

	os.Setenv("MODE", "development")
	os.Unsetenv("DEBUG")
	defer func() {
		os.Unsetenv("MODE")
	}()

	FromEnv()

	if GetCurrentAppMode() != Development {
		t.Errorf("FromEnv: expected Development from MODE=development, got %v", GetCurrentAppMode())
	}
}

func TestFromEnv_Debug(t *testing.T) {
	resetState(t)

	os.Unsetenv("MODE")
	os.Setenv("DEBUG", "true")
	defer func() {
		os.Unsetenv("DEBUG")
	}()

	FromEnv()

	if !IsDebugEnabled() {
		t.Error("FromEnv: expected debug enabled from DEBUG=true")
	}
}

func TestFromEnv_Empty(t *testing.T) {
	resetState(t)

	os.Unsetenv("MODE")
	os.Unsetenv("DEBUG")

	FromEnv()

	if GetCurrentAppMode() != Production {
		t.Errorf("FromEnv: expected Production when MODE not set, got %v", GetCurrentAppMode())
	}
	if IsDebugEnabled() {
		t.Error("FromEnv: expected debug disabled when DEBUG not set")
	}
}

func TestGetCurrentAppMode(t *testing.T) {
	resetState(t)

	if GetCurrentAppMode() != Production {
		t.Errorf("default mode should be Production, got %v", GetCurrentAppMode())
	}

	SetAppMode("development")
	if GetCurrentAppMode() != Development {
		t.Errorf("after SetAppMode(development), got %v", GetCurrentAppMode())
	}
}
