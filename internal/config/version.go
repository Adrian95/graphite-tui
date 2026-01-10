package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	CurrentVersion = "v1.9.0"
	RepoOwner      = "Adrian95"
	RepoName       = "graphite-tui"
	GithubAPI      = "https://api.github.com/repos/%s/%s/releases/latest"
)

// IsDevBuild checks if this binary was built locally (go run/build) vs installed (go install)
func IsDevBuild() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return true // Can't read build info, assume dev
	}
	// When installed via `go install ...@latest`, the version will be set
	// When built locally, version is "(devel)"
	return info.Main.Version == "(devel)" || info.Main.Version == ""
}

// --- Version Messages ---

type VersionCheckMsg struct {
	LatestVersion string
	Err           error
}

type UpdateCompleteMsg struct {
	Success bool
	Err     error
	Message string
}

type UninstallCompleteMsg struct {
	Success bool
	Err     error
}

// --- Version Checking ---

// CheckForUpdates queries GitHub API for the latest release
func CheckForUpdates() tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf(GithubAPI, RepoOwner, RepoName)

		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return VersionCheckMsg{Err: err}
		}

		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "graphite-tui")

		resp, err := client.Do(req)
		if err != nil {
			return VersionCheckMsg{Err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return VersionCheckMsg{Err: fmt.Errorf("GitHub API returned %d", resp.StatusCode)}
		}

		var release struct {
			TagName string `json:"tag_name"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return VersionCheckMsg{Err: err}
		}

		return VersionCheckMsg{LatestVersion: release.TagName}
	}
}

// PerformUpdate updates the tool to the latest version
// It handles identifying the correct installation path and replacing the running binary
func PerformUpdate() tea.Cmd {
	return func() tea.Msg {
		// 1. Check if Go is installed
		goPath, err := exec.LookPath("go")
		if err != nil {
			return UpdateCompleteMsg{
				Success: false,
				Err:     err,
				Message: `Go is not installed!

To install graphite-tui, you need Go first:

  1. Install Go: https://go.dev/dl/
  2. Restart your terminal
  3. Run: go install github.com/Adrian95/graphite-tui@latest
  4. Make sure ~/go/bin is in your PATH`,
			}
		}

		// 2. Determine where 'go install' puts binaries (GOBIN or GOPATH/bin)
		var installDir string
		if gobin := os.Getenv("GOBIN"); gobin != "" {
			installDir = gobin
		} else if gopath := os.Getenv("GOPATH"); gopath != "" {
			installDir = filepath.Join(gopath, "bin")
		} else {
			// Default GOPATH is $HOME/go
			home, _ := os.UserHomeDir()
			installDir = filepath.Join(home, "go", "bin")
		}

		targetBinary := filepath.Join(installDir, "graphite-tui")

		// 3. Run go install to update the global binary (with GOPROXY=direct to bypass cache)
		installPath := fmt.Sprintf("github.com/%s/%s@latest", RepoOwner, RepoName)
		cmd := exec.Command(goPath, "install", installPath)
		cmd.Env = append(os.Environ(), "GOPROXY=direct")
		output, err := cmd.CombinedOutput()
		if err != nil {
			errMsg := string(output)
			if errMsg == "" {
				errMsg = err.Error()
			}
			return UpdateCompleteMsg{
				Success: false,
				Err:     err,
				Message: fmt.Sprintf("Update failed: %s\n\nTry manually:\n  go install %s", errMsg, installPath),
			}
		}

		// 4. Identify the currently running executable
		currentExe, err := os.Executable()
		if err != nil {
			// Can't identify self, assume success if go install worked
			return UpdateCompleteMsg{
				Success: true,
				Message: "Update installed globally! If you are running a local copy, please restart from the global path.",
			}
		}

		// Resolve symlinks to compare real paths
		realCurrent, err := filepath.EvalSymlinks(currentExe)
		if err == nil {
			currentExe = realCurrent
		}

		realTarget, err := filepath.EvalSymlinks(targetBinary)
		if err == nil {
			targetBinary = realTarget
		}

		// 5. Always copy the new binary to the current location
		// This handles both the case where we are running a different binary AND
		// the case where we are running the global binary but the update didn't overwrite it correctly in-place (e.g. windows/file lock issues, though we are on mac)
		// But mostly, this ensures that IF we are running a local copy, it gets updated.

		// Read the new binary
		input, err := os.ReadFile(targetBinary)
		if err != nil {
			return UpdateCompleteMsg{
				Success: true,
				Message: fmt.Sprintf("Updated globally to %s\n(Could not read new binary to replace current execution)", targetBinary),
			}
		}

		// Replace current binary
		oldPath := currentExe + ".old"
		_ = os.Rename(currentExe, oldPath) // Ignore error, best effort

		// Write new file
		err = os.WriteFile(currentExe, input, 0755)
		if err != nil {
			// Restore if write failed
			_ = os.Rename(oldPath, currentExe)
			return UpdateCompleteMsg{
				Success: true,
				Message: fmt.Sprintf("Updated globally to %s\n(Note: Could not replace currently running file at %s)", targetBinary, currentExe),
			}
		}

		// Cleanup old file
		_ = os.Remove(oldPath)

		return UpdateCompleteMsg{
			Success: true,
			Message: fmt.Sprintf("Update complete! Replaced binary at %s\nPlease restart the application.", currentExe),
		}
	}
}

// PerformUninstall removes the tool
func PerformUninstall() tea.Cmd {
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
			return UninstallCompleteMsg{
				Success: false,
				Err:     fmt.Errorf("failed to remove %s: %v", binaryPath, err),
			}
		}

		return UninstallCompleteMsg{Success: true}
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

// IsNewerVersion checks if latest is newer than current using proper semver comparison
func IsNewerVersion(current, latest string) bool {
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
// Shows "(dev)" suffix when running a locally built binary
func GetCurrentVersion() string {
	if IsDevBuild() {
		return CurrentVersion + " (dev)"
	}
	return CurrentVersion
}
