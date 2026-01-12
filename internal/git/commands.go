package git

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// --- Data Types ---

// ChangedFile represents a file with changes in git status
type ChangedFile struct {
	Status string // M, A, D, ??, etc.
	Path   string
}

// StackItem represents a single item in the git stack
type StackItem struct {
	Name    string
	Current bool
	Level   int
}

// MenuItem represents a menu option
type MenuItem struct {
	Title   string
	Desc    string
	Guide   string
	Command string
	Key     string
}

// CommitType represents a conventional commit type
type CommitType struct {
	Label   string
	Desc    string
	Example string
}

// --- Message Types ---

// CmdFinishedMsg is sent when a command completes
type CmdFinishedMsg struct {
	Output  string
	Stderr  string
	Err     error
	Command string
}

// StatusMsg contains the current git/graphite status
type StatusMsg struct {
	Branch        string
	Stack         []string
	Ahead         int
	Behind        int
	ChangedFiles  []ChangedFile
	HasStack      bool
	OnMain        bool
	GtInitialized bool
	Suggestion    string
}

// StackLoadedMsg is sent when the stack is loaded
type StackLoadedMsg []StackItem

// --- Singleton Executor ---

var executor = NewCommandExecutor()

// --- Menu and Commit Type Definitions ---

// GetMenuItems returns the list of menu items
func GetMenuItems() []MenuItem {
	return []MenuItem{
		{Title: "Init", Desc: "Initialize Graphite", Guide: "Sets up Graphite in this repo.", Command: "gt init --no-interactive", Key: "i"},
		{Title: "Start", Desc: "Create branch & commit", Guide: "Creates a new branch with commit.", Command: "gt create -a --no-interactive -m", Key: "s"},
		{Title: "Share", Desc: "Submit for review", Guide: "Submits your changes for review.", Command: "gt submit --no-interactive", Key: "p"},
		{Title: "Fix", Desc: "Amend changes", Guide: "Updates your last commit.", Command: "", Key: "f"},
		{Title: "Sync", Desc: "Update local", Guide: "Pulls latest changes from GitHub.", Command: "gt sync", Key: "y"},
		{Title: "Done", Desc: "Merge & Cleanup", Guide: "Merges approved PR and cleans up.", Command: "gt merge --no-interactive && gt sync --no-interactive", Key: "d"},
		{Title: "Fold", Desc: "Squash stack", Guide: "Combines multiple commits into one.", Command: "gt fold --no-interactive", Key: ""},
		{Title: "Ghost Fix", Desc: "Rescue ghost branch", Guide: "Rescues work from merged branch.", Command: "", Key: ""},
		{Title: "Stack Map", Desc: "Visual GPS", Guide: "See branches visually.", Command: "gt log short", Key: "g"},
	}
}

// GetCommitTypes returns the list of conventional commit types
func GetCommitTypes() []CommitType {
	return []CommitType{
		{"feat", "New feature", "add dark mode toggle"},
		{"fix", "Bug fix", "fix login redirect loop"},
		{"docs", "Documentation", "update API docs"},
		{"style", "Formatting", "fix indentation"},
		{"refactor", "Code restructure", "extract auth logic"},
		{"perf", "Performance", "cache API responses"},
		{"test", "Tests", "add login tests"},
		{"chore", "Tooling", "update dependencies"},
	}
}

// --- Helper Functions ---

// parseGitCount extracts a number after a keyword like "ahead " or "behind "
func parseGitCount(statusLine, keyword string) int {
	idx := strings.Index(statusLine, keyword)
	if idx == -1 {
		return 0
	}
	start := idx + len(keyword)
	if start >= len(statusLine) {
		return 0
	}
	var count int
	for i := start; i < len(statusLine) && statusLine[i] >= '0' && statusLine[i] <= '9'; i++ {
		count = count*10 + int(statusLine[i]-'0')
	}
	return count
}

// getLastCommitMessage retrieves the last commit message
func getLastCommitMessage() string {
	cmd := exec.Command("git", "log", "-1", "--format=%B")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// getDefaultCommitMessage returns a timestamped WIP message
func getDefaultCommitMessage() string {
	return "wip: " + time.Now().Format("Jan 2 15:04")
}

// --- Git Status Commands ---

// CheckGitStatus gets comprehensive git/graphite status
func CheckGitStatus() tea.Msg {
	status := StatusMsg{}

	// Check if Graphite is initialized
	if _, err := os.Stat(".git/.graphite_repo_config"); err == nil {
		status.GtInitialized = true
	}

	// Get current branch
	if out, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		status.Branch = strings.TrimSpace(string(out))
	}

	// Check if on main/master
	status.OnMain = status.Branch == "main" || status.Branch == "master"

	// Get ahead/behind counts
	if out, err := exec.Command("git", "status", "-sb").Output(); err == nil {
		statusLine := string(out)
		status.Ahead = parseGitCount(statusLine, "ahead ")
		status.Behind = parseGitCount(statusLine, "behind ")
	}

	// Get changed files
	if out, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if len(line) >= 4 {
				status.ChangedFiles = append(status.ChangedFiles, ChangedFile{
					Status: strings.TrimSpace(line[0:2]),
					Path:   strings.TrimSpace(line[3:]),
				})
			}
		}
	}

	// Check graphite stack
	if out, err := exec.Command("gt", "log", "short").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 1 {
			status.HasStack = true
			for _, line := range lines {
				name := strings.TrimPrefix(strings.TrimSpace(line), "* ")
				name = strings.TrimLeft(name, " ")
				if name != "" {
					status.Stack = append(status.Stack, name)
				}
			}
		}
	}

	// Generate suggestion
	status.Suggestion = generateSuggestion(status)

	return status
}

func generateSuggestion(s StatusMsg) string {
	switch {
	case !s.GtInitialized:
		return "suggestion: Graphite not initialized. Press [i] to set up Graphite in this repo."
	case len(s.ChangedFiles) > 0:
		return "suggestion: You have unsaved changes! Press [s] to save as NEW work, or [f] to add to your LAST save."
	case s.Ahead > 0:
		return "suggestion: Your work is ready to share! Press [p] to submit for review."
	case s.Behind > 0:
		return "suggestion: Your team made updates. Press [y] to sync and get the latest changes."
	case s.OnMain:
		return "suggestion: You're on main. Press [s] to start working on something new!"
	default:
		return "suggestion: All caught up! Press [s] to add more work, or [d] when approved."
	}
}

// LoadStack loads the graphite stack for the GPS view
func LoadStack() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("gt", "log", "short").Output()
		if err != nil {
			return StackLoadedMsg{}
		}

		var items []StackItem
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			items = append(items, StackItem{
				Name:    strings.TrimPrefix(strings.TrimSpace(line), "* "),
				Current: strings.Contains(line, "*"),
				Level:   (len(line) - len(strings.TrimLeft(line, " "))) / 2,
			})
		}
		return StackLoadedMsg(items)
	}
}

// --- Command Execution Functions ---

// ExecuteInteractive runs a command that needs terminal access
func ExecuteInteractive(commandStr string, inputArg string, skipHooks bool) tea.Cmd {
	cfg := ExecutionConfig{
		Command:   commandStr,
		SkipHooks: skipHooks,
		UseShell:  true,
	}
	if inputArg != "" {
		cfg.Args = []string{inputArg}
	}
	return executor.ExecuteInteractive(cfg)
}

// ExecuteCommit creates a new branch and commit
func ExecuteCommit(msg string, skipHooks bool) tea.Cmd {
	args := []string{"create", "-a", "--no-interactive", "-m", msg}
	if skipHooks {
		args = append(args, "--no-verify")
	}

	cmd := exec.Command("gt", args...)
	cmd.Env = buildNonInteractiveEnv()

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return CmdFinishedMsg{Err: err, Command: "gt create"}
	})
}

// ExecuteSubmit pushes changes and creates/updates PR
func ExecuteSubmit(skipHooks bool) tea.Cmd {
	return executor.ExecuteAsync(ExecutionConfig{
		Command:   "gt submit --no-interactive",
		SkipHooks: skipHooks,
	})
}

// ExecuteSync syncs with remote
func ExecuteSync(skipHooks bool) tea.Cmd {
	cfg := ExecutionConfig{
		Command:      "gt sync",
		SkipHooks:    skipHooks,
		AllowUserEnv: true, // Allow prompts
	}
	return executor.ExecuteInteractive(cfg)
}

// ExecuteTurboShip creates a WIP commit and immediately submits
func ExecuteTurboShip(skipHooks bool) tea.Cmd {
	return ExecuteTurboShipWithMsg(getDefaultCommitMessage(), skipHooks)
}

// ExecuteTurboShipWithMsg creates a commit with custom message and immediately submits
func ExecuteTurboShipWithMsg(commitMsg string, skipHooks bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Step 1: Create commit
		result := executor.Execute(ctx, ExecutionConfig{
			Command:   "gt create -a --no-interactive -m",
			Args:      []string{commitMsg},
			SkipHooks: skipHooks,
			UseShell:  true,
		})
		if result.Err != nil {
			result.Command = "turbo ship (commit)"
			return result
		}

		// Step 2: Submit
		result = executor.Execute(ctx, ExecutionConfig{
			Command:   "gt submit --no-interactive",
			SkipHooks: skipHooks,
		})
		if result.Output == "" && result.Err == nil {
			result.Output = "Shipped! PR submitted for preview."
		}
		result.Command = "turbo ship"
		return result
	}
}

// ExecuteIterate amends the last commit and pushes
func ExecuteIterate(skipHooks bool) tea.Cmd {
	commitMsg := getLastCommitMessage()
	if commitMsg == "" {
		commitMsg = getDefaultCommitMessage()
	}
	return ExecuteIterateWithMsg(commitMsg, skipHooks)
}

// ExecuteIterateWithMsg amends the last commit with custom message and pushes
func ExecuteIterateWithMsg(commitMsg string, skipHooks bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Step 1: Amend commit
		result := executor.Execute(ctx, ExecutionConfig{
			Command:   "gt modify -a --no-interactive -m",
			Args:      []string{commitMsg},
			SkipHooks: skipHooks,
			UseShell:  true,
		})
		if result.Err != nil {
			result.Command = "iterate (amend)"
			return result
		}

		// Step 2: Push
		result = executor.Execute(ctx, ExecutionConfig{
			Command:   "gt submit --no-interactive",
			SkipHooks: skipHooks,
		})
		if result.Output == "" && result.Err == nil {
			result.Output = "Iterated! Changes pushed to PR."
		}
		result.Command = "iterate"
		return result
	}
}

// ExecuteFix amends the last commit without pushing
func ExecuteFix(skipHooks bool) tea.Cmd {
	return func() tea.Msg {
		commitMsg := getLastCommitMessage()
		if commitMsg == "" {
			commitMsg = getDefaultCommitMessage()
		}

		result := executor.Execute(context.Background(), ExecutionConfig{
			Command:   "gt modify -a --no-interactive -m",
			Args:      []string{commitMsg},
			SkipHooks: skipHooks,
			UseShell:  true,
		})
		if result.Output == "" && result.Err == nil {
			result.Output = "Fix complete! Commit amended."
		}
		result.Command = "gt modify"
		return result
	}
}

// ExecuteGhostFix runs the ghost branch rescue sequence
func ExecuteGhostFix(branchName string, skipHooks bool) tea.Cmd {
	// Sanitize branch name
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

	script := "gt create -a --no-interactive" + noVerify + " -m \"$1\" && gt rebase main --no-interactive && gt sync --no-interactive"
	cmd := exec.Command("sh", "-c", script, "--", safeName)
	cmd.Env = buildNonInteractiveEnv()

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return CmdFinishedMsg{Err: err, Command: "Ghost Fix"}
	})
}

// ExecuteHardReset discards all uncommitted changes
func ExecuteHardReset() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Reset tracked files
		result := executor.Execute(ctx, ExecutionConfig{
			Command: "git reset --hard HEAD",
		})
		if result.Err != nil {
			result.Command = "hard reset"
			return result
		}

		// Clean untracked files
		result = executor.Execute(ctx, ExecutionConfig{
			Command: "git clean -fd",
		})
		result.Output = "Reset complete! All changes discarded."
		result.Command = "hard reset"
		return result
	}
}

// ExecuteUndo reverses the last Graphite operation
func ExecuteUndo() tea.Cmd {
	return func() tea.Msg {
		result := executor.Execute(context.Background(), ExecutionConfig{
			Command: "gt undo --no-interactive",
		})
		if result.Output == "" && result.Err == nil {
			result.Output = "Undo complete!"
		}
		result.Command = "gt undo"
		return result
	}
}

// ExecuteCheckout switches to a different branch
func ExecuteCheckout(branch string) tea.Cmd {
	return func() tea.Msg {
		result := executor.Execute(context.Background(), ExecutionConfig{
			Command: "gt checkout " + branch,
		})
		result.Command = "gt checkout " + branch
		return result
	}
}

// --- Legacy Compatibility ---

// GetNonInteractiveEnv returns environment for non-interactive commands
// Deprecated: Use CommandExecutor instead
func GetNonInteractiveEnv() []string {
	return buildNonInteractiveEnv()
}

// AddHooksFlag adds --no-verify flag if skipHooks is true
// Deprecated: Use CommandExecutor with SkipHooks config
func AddHooksFlag(commandStr string, skipHooks bool) string {
	if skipHooks && strings.Contains(commandStr, "gt ") {
		return commandStr + " --no-verify"
	}
	return commandStr
}

// BuildShellCommand creates a shell command
// Deprecated: Use CommandExecutor with UseShell config
func BuildShellCommand(commandStr, inputArg string) *exec.Cmd {
	if inputArg != "" {
		script := commandStr + " \"$1\""
		return exec.Command("sh", "-c", script, "--", inputArg)
	}
	return exec.Command("sh", "-c", commandStr)
}
