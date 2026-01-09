package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// --- Command Messages ---

type cmdFinishedMsg struct {
	output  string
	err     error
	stderr  string
	command string
}

type statusMsg struct {
	suggestion    string
	branch        string
	ahead         int
	behind        int
	stack         []string
	changedFiles  []ChangedFile
	hasStack      bool
	onMain        bool
	gtInitialized bool
}

type stackLoadedMsg []stackItem

// --- Environment Setup ---

// getNonInteractiveEnv returns environment variables that prevent vim/editor popups
func getNonInteractiveEnv() []string {
	return append(os.Environ(),
		"GIT_EDITOR=true",
		"EDITOR=true",
		"VISUAL=true",
		"GT_NON_INTERACTIVE=1",
		"GIT_TERMINAL_PROMPT=0",
	)
}

// addHooksFlag adds --no-verify flag to gt commands if skipHooks is true
func addHooksFlag(commandStr string, skipHooks bool) string {
	if skipHooks && strings.Contains(commandStr, "gt ") {
		return commandStr + " --no-verify"
	}
	return commandStr
}

// buildShellCommand creates a safe shell command with optional argument
func buildShellCommand(commandStr, inputArg string) *exec.Cmd {
	if inputArg != "" {
		script := commandStr + " \"$1\""
		return exec.Command("sh", "-c", script, "--", inputArg)
	}
	return exec.Command("sh", "-c", commandStr)
}

// --- Command Execution ---

// executeCommand runs a shell command with proper error handling and no-vim guarantees.
// It captures both stdout and stderr for better error reporting.
func executeCommand(commandStr string, inputArg string, skipHooks bool) tea.Cmd {
	return func() tea.Msg {
		fullCmd := addHooksFlag(commandStr, skipHooks)
		cmd := buildShellCommand(fullCmd, inputArg)
		cmd.Env = getNonInteractiveEnv()

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		return cmdFinishedMsg{
			output:  stdout.String(),
			stderr:  stderr.String(),
			err:     err,
			command: commandStr,
		}
	}
}

// executeInteractive runs a command that needs terminal access (for gt commands that show output)
func executeInteractive(commandStr string, inputArg string, skipHooks bool) tea.Cmd {
	fullCmd := addHooksFlag(commandStr, skipHooks)
	cmd := buildShellCommand(fullCmd, inputArg)
	cmd.Env = getNonInteractiveEnv()

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return cmdFinishedMsg{
			err:     err,
			command: commandStr,
		}
	})
}

// executeGhostFix runs the ghost branch rescue sequence
func executeGhostFix(branchName string, skipHooks bool) tea.Cmd {
	// Sanitize branch name - remove any shell-unsafe characters
	safeName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/' {
			return r
		}
		return '-'
	}, branchName)

	noVerify := ""
	if skipHooks {
		noVerify = " --no-verify"
	}

	// Use parameterized command for safety
	script := "gt create -a --no-interactive" + noVerify + " -m \"$1\" && gt rebase main --no-interactive && gt sync --no-interactive"
	cmd := exec.Command("sh", "-c", script, "--", safeName)
	cmd.Env = getNonInteractiveEnv()

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return cmdFinishedMsg{err: err, command: "Ghost Fix"}
	})
}

// executeCommit creates a new branch and commit using direct exec.Command (no shell injection)
// This fixes the "empty commit message" bug that occurred with shell parameter expansion
func executeCommit(msg string, skipHooks bool) tea.Cmd {
	args := []string{"create", "-a", "--no-interactive", "-m", msg}
	if skipHooks {
		args = append(args, "--no-verify")
	}
	cmd := exec.Command("gt", args...)
	cmd.Env = getNonInteractiveEnv()

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return cmdFinishedMsg{err: err, command: "gt create -m \"" + msg + "\""}
	})
}

// executeCheckout switches to a different branch using direct gt command (no shell injection)
func executeCheckout(branch string) tea.Cmd {
	return func() tea.Msg {
		// Use exec.Command directly to prevent shell injection
		cmd := exec.Command("gt", "checkout", branch)
		cmd.Env = getNonInteractiveEnv()

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		return cmdFinishedMsg{
			output:  stdout.String(),
			stderr:  stderr.String(),
			err:     err,
			command: "gt checkout " + branch,
		}
	}
}

// --- Git Status Commands ---

// checkGitStatus gets comprehensive git/graphite status
func checkGitStatus() tea.Msg {
	status := statusMsg{}

	// Check if Graphite is initialized (look for .graphite_repo_config in .git)
	if _, err := os.Stat(".git/.graphite_repo_config"); err == nil {
		status.gtInitialized = true
	}

	// Get current branch
	if out, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		status.branch = strings.TrimSpace(string(out))
	}

	// Check if on main/master
	if status.branch == "main" || status.branch == "master" {
		status.onMain = true
	}

	// Get ahead/behind counts
	if out, err := exec.Command("git", "status", "-sb").Output(); err == nil {
		statusLine := string(out)
		if strings.Contains(statusLine, "ahead") {
			// Parse "ahead N" - extract number after "ahead "
			for i := strings.Index(statusLine, "ahead "); i != -1 && i+6 < len(statusLine); {
				var count int
				for j := i + 6; j < len(statusLine) && statusLine[j] >= '0' && statusLine[j] <= '9'; j++ {
					count = count*10 + int(statusLine[j]-'0')
				}
				status.ahead = count
				break
			}
		}
		if strings.Contains(statusLine, "behind") {
			for i := strings.Index(statusLine, "behind "); i != -1 && i+7 < len(statusLine); {
				var count int
				for j := i + 7; j < len(statusLine) && statusLine[j] >= '0' && statusLine[j] <= '9'; j++ {
					count = count*10 + int(statusLine[j]-'0')
				}
				status.behind = count
				break
			}
		}
	}

	// Get changed files
	if out, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			// Need at least 4 chars: 2 for status, 1 space, 1+ for path
			if len(line) < 4 {
				continue
			}
			status.changedFiles = append(status.changedFiles, ChangedFile{
				Status: strings.TrimSpace(line[0:2]),
				Path:   strings.TrimSpace(line[3:]),
			})
		}
	}

	// Check if we have a graphite stack
	if out, err := exec.Command("gt", "log", "short").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 1 { // More than just current branch
			status.hasStack = true
			// Build simple stack representation
			for _, line := range lines {
				name := strings.TrimSpace(line)
				name = strings.TrimPrefix(name, "* ")
				name = strings.TrimLeft(name, " ")
				if name != "" {
					status.stack = append(status.stack, name)
				}
			}
		}
	}

	// Generate suggestion (beginner-friendly explanations)
	if !status.gtInitialized {
		status.suggestion = "suggestion: Graphite not initialized. Press [i] to set up Graphite in this repo."
	} else if len(status.changedFiles) > 0 {
		status.suggestion = "suggestion: You have unsaved changes! Press [s] to save as NEW work, or [f] to add to your LAST save."
	} else if status.ahead > 0 {
		status.suggestion = "suggestion: Your work is ready to share! Press [p] to push to GitHub and open a PR."
	} else if status.behind > 0 {
		status.suggestion = "suggestion: Your team made updates. Press [y] to sync and get the latest changes."
	} else if status.onMain {
		status.suggestion = "suggestion: You're on main. Press [s] to start working on something new!"
	} else {
		status.suggestion = "suggestion: All caught up! Press [s] to add more work, or [d] when your PR is approved."
	}

	return status
}

// loadStack loads the graphite stack for the GPS view
func loadStack() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gt", "log", "short")
		out, err := cmd.Output()
		if err != nil {
			return stackLoadedMsg{}
		}

		var items []stackItem
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			current := strings.Contains(line, "*")
			cleanName := strings.TrimSpace(line)
			cleanName = strings.TrimPrefix(cleanName, "* ")

			indent := len(line) - len(strings.TrimLeft(line, " "))

			items = append(items, stackItem{
				name:    cleanName,
				current: current,
				level:   indent / 2,
			})
		}
		return stackLoadedMsg(items)
	}
}

// --- Graphite Command Definitions ---

type menuItem struct {
	title   string
	desc    string
	guide   string
	command string
	key     string // keyboard shortcut
}

func getMenuItems() []menuItem {
	return []menuItem{
		{
			title:   "Init",
			desc:    "Initialize Graphite",
			guide:   "Sets up Graphite in this repo. Required before using other commands. Only needed once per repo.",
			command: "gt init --no-interactive",
			key:     "i",
		},
		{
			title:   "Start",
			desc:    "Create branch & commit",
			guide:   "Creates a new branch and guides you through writing a commit message. Perfect for starting new work.",
			command: "gt create -a --no-interactive -m",
			key:     "s",
		},
		{
			title:   "Preview",
			desc:    "Push & Open PR",
			guide:   "Uploads your changes to GitHub and opens a Pull Request for others to review.",
			command: "gt submit --no-interactive",
			key:     "p",
		},
		{
			title:   "Fix",
			desc:    "Amend changes",
			guide:   "Made a small mistake? This updates your last commit without creating a new one.",
			command: "gt modify -a --no-interactive --no-edit",
			key:     "f",
		},
		{
			title:   "Sync",
			desc:    "Update local",
			guide:   "Pulls the latest changes from GitHub. Run this often to stay up to date!",
			command: "gt sync --no-interactive",
			key:     "y",
		},
		{
			title:   "Done",
			desc:    "Merge & Cleanup",
			guide:   "Merges your approved PR and cleans up the local branches. Use when your PR is approved!",
			command: "gt merge --no-interactive && gt sync --no-interactive",
			key:     "d",
		},
		{
			title:   "Fold",
			desc:    "Squash stack",
			guide:   "Combines multiple small commits into one. Useful for cleaning up messy history.",
			command: "gt fold --no-interactive",
			key:     "",
		},
		{
			title:   "Ghost Fix",
			desc:    "Rescue ghost branch",
			guide:   "Fixes the 'working on a merged branch' problem. Rescues your work onto a fresh branch.",
			command: "",
			key:     "",
		},
		{
			title:   "Stack Map",
			desc:    "Visual GPS",
			guide:   "See your branches visually and jump between them. Like a GPS for your code!",
			command: "gt log short",
			key:     "g",
		},
	}
}

// Commit types for the wizard
type commitType struct {
	label string
	desc  string
}

func getCommitTypes() []commitType {
	return []commitType{
		{"feat", "A new feature"},
		{"fix", "A bug fix"},
		{"docs", "Documentation only"},
		{"style", "Formatting, no code change"},
		{"refactor", "Code change, no new feature or fix"},
		{"perf", "Performance improvement"},
		{"test", "Adding or fixing tests"},
		{"chore", "Build process or tooling"},
	}
}
