package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	currentVersion = "v2.0.0"
	repoOwner      = "Adrian95"
	repoName       = "graphite-tui"
	githubAPI      = "https://api.github.com/repos/%s/%s/releases/latest"
)

// --- Version Messages ---

type versionCheckMsg struct {
	latestVersion string
	err           error
}

type updateCompleteMsg struct {
	success bool
	err     error
	message string
}

type uninstallCompleteMsg struct {
	success bool
	err     error
}

// --- Version Checking ---

// checkForUpdates queries GitHub API for the latest release
func checkForUpdates() tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf(githubAPI, repoOwner, repoName)

		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return versionCheckMsg{err: err}
		}

		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "graphite-tui")

		resp, err := client.Do(req)
		if err != nil {
			return versionCheckMsg{err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return versionCheckMsg{err: fmt.Errorf("GitHub API returned %d", resp.StatusCode)}
		}

		var release struct {
			TagName string `json:"tag_name"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return versionCheckMsg{err: err}
		}

		return versionCheckMsg{latestVersion: release.TagName}
	}
}

// performUpdate updates the tool to the latest version
func performUpdate() tea.Cmd {
	return func() tea.Msg {
		// Check if Go is installed
		goPath, err := exec.LookPath("go")
		if err != nil {
			return updateCompleteMsg{
				success: false,
				err:     err,
				message: `Go is not installed!

To install graphite-tui, you need Go first:

  1. Install Go: https://go.dev/dl/
  2. Restart your terminal
  3. Run: go install github.com/Adrian95/graphite-tui@latest
  4. Make sure ~/go/bin is in your PATH`,
			}
		}

		// Use go install to update
		installPath := fmt.Sprintf("github.com/%s/%s@latest", repoOwner, repoName)
		cmd := exec.Command(goPath, "install", installPath)

		output, err := cmd.CombinedOutput()
		if err != nil {
			errMsg := string(output)
			if errMsg == "" {
				errMsg = err.Error()
			}
			return updateCompleteMsg{
				success: false,
				err:     err,
				message: fmt.Sprintf("Update failed: %s\n\nTry manually:\n  go install %s", errMsg, installPath),
			}
		}

		return updateCompleteMsg{
			success: true,
			message: "Update complete! Please restart graphite-tui to use the new version.",
		}
	}
}

// performUninstall removes the tool
func performUninstall() tea.Cmd {
	return func() tea.Msg {
		// Find the binary location
		binaryPath, err := exec.LookPath("graphite-tui")
		if err != nil {
			// Try common locations
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				home, _ := os.UserHomeDir()
				gopath = filepath.Join(home, "go")
			}
			binaryPath = filepath.Join(gopath, "bin", "graphite-tui")
		}

		// Remove the binary
		if err := os.Remove(binaryPath); err != nil {
			return uninstallCompleteMsg{
				success: false,
				err:     fmt.Errorf("failed to remove %s: %v", binaryPath, err),
			}
		}

		return uninstallCompleteMsg{success: true}
	}
}

// --- Version Comparison ---

// parseVersion extracts major, minor, patch from a version string like "v1.2.3" or "1.2.3"
func parseVersion(v string) (major, minor, patch int, ok bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")

	if len(parts) < 1 {
		return 0, 0, 0, false
	}

	var err error
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}

	if len(parts) >= 2 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			minor = 0
		}
	}

	if len(parts) >= 3 {
		// Handle versions like "1.2.3-beta" by taking only the numeric part
		patchStr := parts[2]
		if idx := strings.IndexAny(patchStr, "-+"); idx != -1 {
			patchStr = patchStr[:idx]
		}
		patch, err = strconv.Atoi(patchStr)
		if err != nil {
			patch = 0
		}
	}

	return major, minor, patch, true
}

// isNewerVersion checks if latest is newer than current using proper semver comparison
func isNewerVersion(current, latest string) bool {
	curMajor, curMinor, curPatch, ok1 := parseVersion(current)
	latMajor, latMinor, latPatch, ok2 := parseVersion(latest)

	if !ok1 || !ok2 {
		return false
	}

	if latMajor > curMajor {
		return true
	}
	if latMajor < curMajor {
		return false
	}

	// Major versions equal, check minor
	if latMinor > curMinor {
		return true
	}
	if latMinor < curMinor {
		return false
	}

	// Minor versions equal, check patch
	return latPatch > curPatch
}

// GetCurrentVersion returns the current version string
func GetCurrentVersion() string {
	return currentVersion
}
