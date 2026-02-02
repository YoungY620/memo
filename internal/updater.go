// Package internal provides shared utilities for memo
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	// GitHub API endpoint for latest release
	releaseAPI = "https://api.github.com/repos/YoungY620/memo/releases/latest"
	// Timeout for update check
	updateTimeout = 2 * time.Second
)

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
	UpdateCommand  string
}

// githubRelease represents the GitHub API response for a release
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckUpdate checks if a newer version is available
// Returns nil if check fails or no update available
func CheckUpdate(currentVersion string) *UpdateInfo {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()

	latest, err := fetchLatestVersion(ctx)
	if err != nil {
		LogDebug("Update check failed: %v", err)
		return nil
	}

	// Normalize versions (remove 'v' prefix for comparison)
	current := normalizeVersion(currentVersion)
	latestNorm := normalizeVersion(latest)

	if !isNewerVersion(latestNorm, current) {
		return nil
	}

	return &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latest,
		HasUpdate:      true,
		UpdateCommand:  getUpdateCommand(),
	}
}

// CheckUpdateAsync checks for updates asynchronously
// Returns a channel that will receive the result (or nil if no update/error)
func CheckUpdateAsync(currentVersion string) <-chan *UpdateInfo {
	ch := make(chan *UpdateInfo, 1)
	go func() {
		ch <- CheckUpdate(currentVersion)
	}()
	return ch
}

// fetchLatestVersion fetches the latest version from GitHub API
func fetchLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", releaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "memo-update-checker")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

// normalizeVersion removes 'v' prefix and trims whitespace
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return v
}

// isNewerVersion returns true if latest is newer than current
// Uses simple string comparison for semver (works for most cases)
func isNewerVersion(latest, current string) bool {
	// Handle dev/dirty versions
	if current == "dev" || strings.Contains(current, "dirty") {
		return false // Don't prompt for dev builds
	}

	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	// Compare each part numerically
	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		l := parseVersionPart(latestParts[i])
		c := parseVersionPart(currentParts[i])
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}

	// If all compared parts are equal, longer version is newer
	return len(latestParts) > len(currentParts)
}

// parseVersionPart extracts numeric part from version component
func parseVersionPart(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break // Stop at first non-digit (e.g., "1-beta" -> 1)
		}
	}
	return n
}

// getUpdateCommand returns the appropriate update command based on OS and install location
func getUpdateCommand() string {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}

	switch runtime.GOOS {
	case "windows":
		// Check if installed via scoop
		if strings.Contains(exe, "\\scoop\\") {
			return "scoop update memo"
		}
		return `irm https://raw.githubusercontent.com/YoungY620/memo/main/install.ps1 | iex`

	case "darwin", "linux":
		// Check if installed via homebrew
		if strings.Contains(exe, "/Cellar/") || strings.Contains(exe, "/homebrew/") {
			return "brew upgrade memo"
		}
		return "curl -fsSL https://raw.githubusercontent.com/YoungY620/memo/main/install.sh | sh"

	default:
		return "curl -fsSL https://raw.githubusercontent.com/YoungY620/memo/main/install.sh | sh"
	}
}
