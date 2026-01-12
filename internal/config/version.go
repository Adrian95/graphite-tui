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

// --- Constants ---

const (
	// FallbackVersion is used only for dev builds where build info is unavailable.
	// For release builds, version comes from the git tag via go install.
	FallbackVersion = "v0.0.0-dev"

	RepoOwner = "Adrian95"
	RepoName  = "graphite-tui"
	GithubAPI = "https://api.github.com/repos/%s/%s/releases/latest"
)

// --- Build Information ---

// BuildInfo contains version and build metadata embedded at compile time
type BuildInfo struct {
	Version   string // Semantic version (e.g., "v1.9.3")
	Commit    string // Git commit hash (short)
	BuildTime string // When the binary was built
	GoVersion string // Go version used to build
	IsDev     bool   // True if this is a local dev build
}

// GetBuildInfo returns comprehensive build information
func GetBuildInfo() BuildInfo {
	info := BuildInfo{
		Version: FallbackVersion,
		IsDev:   true,
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}

	info.GoVersion = bi.GoVersion

	// Check if installed via go install (has real version)
	if bi.Main.Version != "(devel)" && bi.Main.Version != "" {
		info.Version = bi.Main.Version
		info.IsDev = false
	}

	// Extract VCS info from build settings
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			if len(setting.Value) >= 7 {
				info.Commit = setting.Value[:7]
			} else {
				info.Commit = setting.Value
			}
		case "vcs.time":
			info.BuildTime = setting.Value
		}
	}

	return info
}

// GetCurrentVersion returns the current version string
// For installed binaries, returns the version embedded by `go install`
// For dev builds, shows the fallback version with "(dev)" suffix
func GetCurrentVersion() string {
	info := GetBuildInfo()
	if info.IsDev {
		return info.Version + " (dev)"
	}
	return info.Version
}

// GetVersionForComparison returns a clean version suitable for semver comparison
// Strips any suffixes like "(dev)" or build metadata
func GetVersionForComparison() string {
	info := GetBuildInfo()
	return normalizeVersion(info.Version)
}

// normalizeVersion cleans a version string for comparison
func normalizeVersion(v string) string {
	// Remove common suffixes
	v = strings.TrimSuffix(v, " (dev)")
	v = strings.TrimSuffix(v, "-dev")

	// Ensure v prefix
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}

	return v
}

// IsDevBuild checks if this binary was built locally (go run/build) vs installed (go install)
func IsDevBuild() bool {
	return GetBuildInfo().IsDev
}

// --- Messages ---

type VersionCheckMsg struct {
	LatestVersion string
	CurrentInfo   BuildInfo
	Err           error
}

type UpdateCompleteMsg struct {
	Success        bool
	Err            error
	Message        string
	PreviousVer    string
	InstalledVer   string
	VerifyFailed   bool   // True if version verification failed
	RollbackPath   string // Path to backup if rollback is possible
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

		return VersionCheckMsg{
			LatestVersion: release.TagName,
			CurrentInfo:   GetBuildInfo(),
		}
	}
}

// --- Update Execution ---

// PerformUpdate updates the tool to the latest version with verification
func PerformUpdate() tea.Cmd {
	return PerformUpdateToVersion("")
}

// PerformUpdateToVersion updates to a specific version (empty string = latest)
func PerformUpdateToVersion(targetVersion string) tea.Cmd {
	return func() tea.Msg {
		previousVersion := GetCurrentVersion()

		// 1. Check if Go is installed
		goPath, err := exec.LookPath("go")
		if err != nil {
			return UpdateCompleteMsg{
				Success:     false,
				Err:         err,
				PreviousVer: previousVersion,
				Message: `Go is not installed!

To install graphite-tui, you need Go first:

  1. Install Go: https://go.dev/dl/
  2. Restart your terminal
  3. Run: go install github.com/Adrian95/graphite-tui@latest
  4. Make sure ~/go/bin is in your PATH`,
			}
		}

		// 2. Determine install directory
		installDir := getGoBinDir()
		targetBinary := filepath.Join(installDir, "graphite-tui")

		// 3. Determine version specifier
		versionSpec := "@latest"
		if targetVersion != "" {
			// Normalize version format
			if !strings.HasPrefix(targetVersion, "v") {
				targetVersion = "v" + targetVersion
			}
			versionSpec = "@" + targetVersion
		}

		// 4. Run go install with cache bypass
		installPath := fmt.Sprintf("github.com/%s/%s%s", RepoOwner, RepoName, versionSpec)
		cmd := exec.Command(goPath, "install", installPath)
		// GOPROXY=direct bypasses the module proxy cache
		// GOFLAGS=-mod=mod ensures we get fresh dependencies
		cmd.Env = append(os.Environ(),
			"GOPROXY=direct",
			"GOFLAGS=-mod=mod",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			errMsg := strings.TrimSpace(string(output))
			if errMsg == "" {
				errMsg = err.Error()
			}
			return UpdateCompleteMsg{
				Success:     false,
				Err:         err,
				PreviousVer: previousVersion,
				Message:     fmt.Sprintf("go install failed:\n%s\n\nTry manually:\n  GOPROXY=direct go install %s", errMsg, installPath),
			}
		}

		// 5. Verify the new binary exists
		if _, err := os.Stat(targetBinary); os.IsNotExist(err) {
			return UpdateCompleteMsg{
				Success:     false,
				Err:         fmt.Errorf("binary not found after install"),
				PreviousVer: previousVersion,
				Message:     fmt.Sprintf("go install succeeded but binary not found at %s\n\nCheck your GOBIN/GOPATH configuration.", targetBinary),
			}
		}

		// 6. Get version of newly installed binary
		installedVersion, err := getInstalledBinaryVersion(targetBinary)
		if err != nil {
			// Can't verify, but install seemed to work
			installedVersion = "unknown"
		}

		// 7. Verify version actually changed (if we know the target)
		verifyFailed := false
		if targetVersion != "" && installedVersion != "unknown" {
			expectedNorm := normalizeVersion(targetVersion)
			installedNorm := normalizeVersion(installedVersion)
			if expectedNorm != installedNorm {
				verifyFailed = true
			}
		}

		// 8. Handle currently running executable
		currentExe, err := os.Executable()
		if err != nil {
			msg := fmt.Sprintf("Updated to %s in %s\n\nRestart the application to use the new version.", installedVersion, installDir)
			if verifyFailed {
				msg += fmt.Sprintf("\n\nWarning: Expected %s but got %s", targetVersion, installedVersion)
			}
			return UpdateCompleteMsg{
				Success:      true,
				PreviousVer:  previousVersion,
				InstalledVer: installedVersion,
				VerifyFailed: verifyFailed,
				Message:      msg,
			}
		}

		// Resolve symlinks
		realCurrent, err := filepath.EvalSymlinks(currentExe)
		if err == nil {
			currentExe = realCurrent
		}

		realTarget, err := filepath.EvalSymlinks(targetBinary)
		if err == nil {
			targetBinary = realTarget
		}

		// 9. Copy new binary to current location (handles running from different path)
		rollbackPath := ""
		if currentExe != targetBinary {
			// Read new binary
			newBinaryData, err := os.ReadFile(targetBinary)
			if err != nil {
				return UpdateCompleteMsg{
					Success:      true,
					PreviousVer:  previousVersion,
					InstalledVer: installedVersion,
					VerifyFailed: verifyFailed,
					Message:      fmt.Sprintf("Updated globally to %s\n\nNote: Could not update currently running binary at %s\nRestart from %s to use new version.", installedVersion, currentExe, installDir),
				}
			}

			// Backup current binary for potential rollback
			rollbackPath = currentExe + ".backup"
			if err := copyFile(currentExe, rollbackPath); err != nil {
				rollbackPath = "" // Backup failed, can't rollback
			}

			// Replace current binary
			oldPath := currentExe + ".old"
			_ = os.Rename(currentExe, oldPath)

			if err := os.WriteFile(currentExe, newBinaryData, 0755); err != nil {
				// Restore from .old
				_ = os.Rename(oldPath, currentExe)
				return UpdateCompleteMsg{
					Success:      true,
					PreviousVer:  previousVersion,
					InstalledVer: installedVersion,
					VerifyFailed: verifyFailed,
					Message:      fmt.Sprintf("Updated globally to %s\n\nNote: Could not replace currently running binary (permission denied).\nRestart from %s to use new version.", installedVersion, installDir),
				}
			}

			// Cleanup
			_ = os.Remove(oldPath)
		}

		// 10. Build success message
		msg := fmt.Sprintf("Update complete!\n\n  %s → %s\n\nRestart the application to use the new version.", previousVersion, installedVersion)
		if verifyFailed {
			msg += fmt.Sprintf("\n\nWarning: Expected %s but installed %s\nThis may indicate a release/tagging issue.", targetVersion, installedVersion)
		}
		if rollbackPath != "" {
			msg += fmt.Sprintf("\n\nBackup saved to: %s", rollbackPath)
		}

		return UpdateCompleteMsg{
			Success:      true,
			PreviousVer:  previousVersion,
			InstalledVer: installedVersion,
			VerifyFailed: verifyFailed,
			RollbackPath: rollbackPath,
			Message:      msg,
		}
	}
}

// getGoBinDir returns the directory where go install places binaries
func getGoBinDir() string {
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return gobin
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "bin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}

// getInstalledBinaryVersion runs the installed binary to get its version
func getInstalledBinaryVersion(binaryPath string) (string, error) {
	// Use go version -m to inspect the binary's embedded module info
	cmd := exec.Command("go", "version", "-m", binaryPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output for mod line: "mod github.com/Adrian95/graphite-tui v1.9.3"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "mod") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				return fields[2], nil
			}
		}
	}

	return "", fmt.Errorf("version not found in binary metadata")
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

// --- Uninstall ---

// PerformUninstall removes the tool
func PerformUninstall() tea.Cmd {
	return func() tea.Msg {
		// Find the binary location
		binaryPath, err := exec.LookPath("graphite-tui")
		if err != nil {
			binaryPath = filepath.Join(getGoBinDir(), "graphite-tui")
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
	// Remove any suffix like "(dev)" or build metadata
	if idx := strings.Index(v, " "); idx != -1 {
		v = v[:idx]
	}
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

// --- Diagnostics ---

// DiagnoseVersionIssue returns diagnostic info when version doesn't match expected
func DiagnoseVersionIssue(expected, actual string) string {
	var issues []string

	if expected == actual {
		return "Versions match - no issue detected."
	}

	// Check if it's a normalization issue
	if normalizeVersion(expected) == normalizeVersion(actual) {
		issues = append(issues, "Versions match after normalization (formatting difference only)")
	}

	// Check module proxy cache
	issues = append(issues, "Possible causes:")
	issues = append(issues, "  - Go module proxy cache (try: GOPROXY=direct go install ...)")
	issues = append(issues, "  - Git tag not pushed (verify tag exists on GitHub)")
	issues = append(issues, "  - Release created without matching tag")
	issues = append(issues, "  - go.mod version doesn't match release tag")

	return strings.Join(issues, "\n")
}
