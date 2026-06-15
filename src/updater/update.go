
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package updater provides self-update functionality per AI.md PART 23.
// Supports checking for updates, downloading, and in-place binary replacement.
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Config holds update configuration
type Config struct {
	// CurrentVersion is the current running version
	CurrentVersion string
	// Branch is the update channel: stable, beta, or daily
	Branch string
	// GithubOwner is the GitHub repository owner
	GithubOwner string
	// GithubRepo is the GitHub repository name
	GithubRepo string
	// BinaryName is the base name of the binary (without platform suffix)
	BinaryName string
}

// DefaultConfig returns default update configuration
func DefaultConfig(version string) Config {
	return Config{
		CurrentVersion: version,
		Branch:         "stable",
		GithubOwner:    "casjay-forks",
		GithubRepo:     "caspb",
		BinaryName:     "caspb",
	}
}

// Release represents a GitHub release
type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Prerelease bool    `json:"prerelease"`
	Draft      bool    `json:"draft"`
	Assets     []Asset `json:"assets"`
	Body       string  `json:"body"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

// UpdateResult holds the result of an update check
type UpdateResult struct {
	Available      bool
	CurrentVersion string
	NewVersion     string
	Release        *Release
}

// CheckForUpdate checks GitHub releases for updates per AI.md PART 23
func CheckForUpdate(ctx context.Context, cfg Config) (*UpdateResult, error) {
	result := &UpdateResult{
		CurrentVersion: cfg.CurrentVersion,
	}

	// Determine API endpoint based on branch
	var url string
	switch cfg.Branch {
	case "stable":
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
			cfg.GithubOwner, cfg.GithubRepo)
	default:
		// For beta/daily, get all releases and filter
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases",
			cfg.GithubOwner, cfg.GithubRepo)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", cfg.BinaryName, cfg.CurrentVersion))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	// HTTP 404 means no updates available (already current)
	if resp.StatusCode == 404 {
		return result, nil
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	if cfg.Branch == "stable" {
		var release Release
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		if release.Draft {
			return result, nil
		}
		if normalizeVersion(release.TagName) == normalizeVersion(cfg.CurrentVersion) {
			return result, nil
		}
		result.Available = true
		result.NewVersion = release.TagName
		result.Release = &release
		return result, nil
	}

	// For beta/daily, filter releases
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	for _, r := range releases {
		if r.Draft {
			continue
		}
		if matchesBranch(r, cfg.Branch) && normalizeVersion(r.TagName) != normalizeVersion(cfg.CurrentVersion) {
			result.Available = true
			result.NewVersion = r.TagName
			result.Release = &r
			return result, nil
		}
	}

	return result, nil
}

// normalizeVersion strips leading 'v' from version string
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// matchesBranch checks if a release matches the specified branch
func matchesBranch(r Release, branch string) bool {
	switch branch {
	case "beta":
		return strings.HasSuffix(r.TagName, "-beta")
	case "daily":
		// Daily builds are timestamps: YYYYMMDDHHMMSS
		return len(r.TagName) == 14 && !strings.Contains(r.TagName, ".")
	default:
		// stable: non-prerelease versions
		return !r.Prerelease
	}
}

// DoUpdate downloads and installs the update per AI.md PART 23
func DoUpdate(ctx context.Context, cfg Config, release *Release) error {
	// Find the right asset for this platform
	assetName := getBinaryName(cfg.BinaryName)
	var downloadURL string
	var checksumURL string

	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
		}
		if asset.Name == assetName+".sha256" {
			checksumURL = asset.BrowserDownloadURL
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s (looking for %s)",
			runtime.GOOS, runtime.GOARCH, assetName)
	}

	fmt.Printf("Downloading %s...\n", assetName)

	// Download to temp file
	tmpFile, err := os.CreateTemp("", cfg.BinaryName+"-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	// Clean up on error (will be renamed on success)
	defer func() {
		if _, err := os.Stat(tmpPath); err == nil {
			os.Remove(tmpPath)
		}
	}()

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", cfg.BinaryName, cfg.CurrentVersion))

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		tmpFile.Close()
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Copy with progress
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to download: %w", err)
	}
	tmpFile.Close()

	fmt.Printf("Downloaded %d bytes\n", written)

	// Verify checksum if available
	if checksumURL != "" {
		fmt.Println("Verifying checksum...")
		if err := verifyChecksumFromURL(ctx, tmpPath, checksumURL, cfg); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("Checksum verified")
	}

	// Make executable (Unix)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	// Get current binary path
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	fmt.Printf("Replacing %s...\n", currentPath)

	// Replace binary (platform-specific)
	if err := ReplaceBinary(currentPath, tmpPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Println("Update complete!")
	return nil
}

// getBinaryName returns the expected binary name for this platform
func getBinaryName(baseName string) string {
	name := baseName + "-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// verifyChecksumFromURL downloads and verifies checksum
func verifyChecksumFromURL(ctx context.Context, filePath, checksumURL string, cfg Config) error {
	req, err := http.NewRequestWithContext(ctx, "GET", checksumURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", cfg.BinaryName, cfg.CurrentVersion))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download checksum: %d", resp.StatusCode)
	}

	// Read checksum file (format: "hash  filename" or just "hash")
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse checksum (first field)
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return fmt.Errorf("empty checksum file")
	}
	expectedHash := strings.ToLower(parts[0])

	return verifyChecksum(filePath, expectedHash)
}

// verifyChecksum verifies SHA256 checksum per AI.md PART 23
func verifyChecksum(filePath, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

// PrintHelp prints update command help
func PrintHelp(binaryName string) {
	fmt.Printf(`%s --update - Check and perform self-updates

Usage:
  %s --update [command]

Commands:
  check              Check for updates without installing
  yes                Check and install updates (default)
  branch {name}      Set update branch (stable, beta, daily)
  --help             Show this help

Examples:
  %s --update check          # Check for updates
  %s --update                # Install updates (same as 'yes')
  %s --update yes            # Install updates
  %s --update branch beta    # Switch to beta channel
  %s --update branch stable  # Switch to stable channel

Update Branches:
  stable    Release versions (v1.0.0, v2.0.0, etc.)
  beta      Pre-release beta builds (*-beta)
  daily     Daily development builds (YYYYMMDDHHMMSS)

Exit Codes:
  0  Success (updated or no updates available)
  1  Error
`, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName)
}
