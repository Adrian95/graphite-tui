package git

import (
	"context"
	"fmt"
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
	Staged bool
}

// StackItem represents a single item in the git stack
type StackItem struct {
	Name    string
	Current bool
	Level   int
}

// ReflogItem represents a git reflog entry
type ReflogItem struct {
	Hash    string
	Message string
	Date    string
	RefName string // HEAD@{n}
}

// StashItem represents a git stash entry
type StashItem struct {
	ID      string // stash@{n}
	Branch  string
	Message string
	Date    string
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

// LocalStatusMsg contains local git status (fast)
type LocalStatusMsg struct {
	Branch        string
	Ahead         int
	Behind        int
	ChangedFiles  []ChangedFile
	OnMain        bool
	GtInitialized bool
	Suggestion    string
	StagedCount   int
	LinesAdded    int
	LinesRemoved  int
	NewFiles      int
	ModFiles      int
	DelFiles      int
}

// StackStatusMsg contains graphite stack status (slow)
type StackStatusMsg struct {
	Stack    []string
	HasStack bool
}

// StatusMsg contains the current git/graphite status
// Deprecated: Use LocalStatusMsg and StackStatusMsg
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

// ReflogLoadedMsg is sent when the reflog is loaded
type ReflogLoadedMsg []ReflogItem

// StashLoadedMsg is sent when the stash is loaded
type StashLoadedMsg []StashItem

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

// GetRecentScopes scans git log for used scopes
func GetRecentScopes() []string {
	// grep for patterns like "feat(scope):" or "fix(scope):"
	cmd := exec.Command("git", "log", "-n", "100", "--format=%s")
	out, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	scopesMap := make(map[string]bool)
	var scopes []string

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		// regex-like parsing: look for '(' and '):'
		start := strings.Index(line, "(")
		end := strings.Index(line, "):")
		if start != -1 && end != -1 && end > start {
			scope := line[start+1 : end]
			if !scopesMap[scope] && scope != "" {
				scopesMap[scope] = true
				scopes = append(scopes, scope)
			}
		}
	}

	// Limit to top 5
	if len(scopes) > 5 {
		return scopes[:5]
	}
	return scopes
}

// --- Git Status Commands ---

// CheckLocalStatus gets fast local git status
func CheckLocalStatus() tea.Msg {
	status := LocalStatusMsg{}

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
				code := line[0:2]
				path := strings.TrimSpace(line[3:])

				// Determine staged status
				// Index 0 is index status, Index 1 is worktree status
				// M_ = Staged Modified
				// A_ = Staged Added
				// D_ = Staged Deleted
				// R_ = Staged Renamed
				// ?? = Untracked (Unstaged)
				// _M = Unstaged Modified
				staged := code[0] != ' ' && code[0] != '?'

				status.ChangedFiles = append(status.ChangedFiles, ChangedFile{
					Status: strings.TrimSpace(code),
					Path:   path,
					Staged: staged,
				})

				if staged {
					status.StagedCount++
				}

				// Categorize for metrics
				if strings.Contains(code, "A") || strings.Contains(code, "?") {
					status.NewFiles++
				} else if strings.Contains(code, "D") {
					status.DelFiles++
				} else {
					status.ModFiles++
				}
			}
		}
	}

	// Get Diff Stats (LOC)
	// git diff --shortstat HEAD
	// Output: " 2 files changed, 15 insertions(+), 3 deletions(-)"
	if out, err := exec.Command("git", "diff", "--shortstat", "HEAD").Output(); err == nil {
		parts := strings.Fields(string(out))
		for i, p := range parts {
			if strings.Contains(p, "insertion") && i > 0 {
				fmt.Sscanf(parts[i-1], "%d", &status.LinesAdded)
			} else if strings.Contains(p, "deletion") && i > 0 {
				fmt.Sscanf(parts[i-1], "%d", &status.LinesRemoved)
			}
		}
	}

	status.Suggestion = generateSuggestion(status)

	return status
}

// CheckStackStatus gets slow graphite stack status
func CheckStackStatus() tea.Msg {
	status := StackStatusMsg{}

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
	return status
}

// CheckGitStatus gets comprehensive git/graphite status

func CheckGitStatus() (LocalStatusMsg, StackStatusMsg) {
	local := CheckLocalStatus().(LocalStatusMsg)
	stack := CheckStackStatus().(StackStatusMsg)
	return local, stack
}

// CheckMinimalStatus gets lightweight status without expensive file operations
func CheckMinimalStatus() LocalStatusMsg {
	status := LocalStatusMsg{}

	// Check if Graphite is initialized
	if _, err := os.Stat(".git/.graphite_repo_config"); err == nil {
		status.GtInitialized = true
	}

	// Get current branch only
	if out, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		status.Branch = strings.TrimSpace(string(out))
	}

	// Skip expensive operations (file listing, ahead/behind counts)
	// These will be loaded lazily when needed

	return status
}

func generateSuggestion(s LocalStatusMsg) string {
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

// LoadReflog loads the git reflog
func LoadReflog() tea.Cmd {
	return func() tea.Msg {
		// format: abbreviated_commit|relative_date|subject|refname
		out, err := exec.Command("git", "reflog", "--date=relative", "--format=%h|%cr|%gs|%gd", "-n", "50").Output()
		if err != nil {
			return ReflogLoadedMsg{}
		}

		var items []ReflogItem
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				items = append(items, ReflogItem{
					Hash:    parts[0],
					Date:    parts[1],
					Message: parts[2],
					RefName: parts[3],
				})
			}
		}
		return ReflogLoadedMsg(items)
	}
}

// LoadStash loads the git stash
func LoadStash() tea.Cmd {
	return func() tea.Msg {
		// format: stash@{n}|branch|relative_date|message
		// git stash list --format="%gd|%b|%cr|%s"
		out, err := exec.Command("git", "stash", "list", "--format=%gd|%b|%cr|%s").Output()
		if err != nil {
			return StashLoadedMsg{}
		}

		var items []StashItem
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 4 {
				items = append(items, StashItem{
					ID:      parts[0],
					Branch:  parts[1],
					Date:    parts[2],
					Message: parts[3],
				})
			}
		}
		return StashLoadedMsg(items)
	}
}

// ExecuteStashCommand performs a stash action
func ExecuteStashCommand(action string, id string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		msg := ""

		switch action {
		case "pop":
			cmd = exec.Command("git", "stash", "pop", id)
			msg = "Stash popped!"
		case "apply":
			cmd = exec.Command("git", "stash", "apply", id)
			msg = "Stash applied!"
		case "drop":
			cmd = exec.Command("git", "stash", "drop", id)
			msg = "Stash dropped!"
		case "create":
			// id is message here
			cmd = exec.Command("git", "stash", "push", "-m", id)
			msg = "Stashed changes!"
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			return CmdFinishedMsg{Err: err, Command: "Stash " + action, Output: string(output)}
		}
		return CmdFinishedMsg{Command: "Stash " + action, Output: msg + "\n" + string(output)}
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

// ExecuteStage toggles staging for a file
func ExecuteStage(path string, stage bool) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		if stage {
			cmd = exec.Command("git", "add", path)
		} else {
			cmd = exec.Command("git", "reset", "HEAD", path)
		}

		err := cmd.Run()
		if err != nil {
			return CmdFinishedMsg{Err: err, Command: "Stage/Unstage"}
		}
		return CmdFinishedMsg{Command: "Stage/Unstage", Output: "Updated index"}
	}
}

// ExecuteStageAll stages or unstages all files
func ExecuteStageAll(stage bool) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		if stage {
			cmd = exec.Command("git", "add", ".")
		} else {
			cmd = exec.Command("git", "reset", "HEAD")
		}

		err := cmd.Run()
		if err != nil {
			return CmdFinishedMsg{Err: err, Command: "Stage All"}
		}
		return CmdFinishedMsg{Command: "Stage All", Output: "Updated index"}
	}
}

// ExecuteCommit creates a new branch and commit
func ExecuteCommit(msg string, skipHooks bool) tea.Cmd {
	// Check if we have staged files
	// If yes, we commit only staged (no -a)
	// If no, we commit all (-a) - Legacy behavior

	// We can't easily check logic here without running a command,
	// so we'll rely on the caller or check quickly.
	// Actually, let's just check git diff --cached --quiet
	// Exit code 1 means differences (staged changes exist)
	// Exit code 0 means no differences (nothing staged)

	hasStaged := false
	if err := exec.Command("git", "diff", "--cached", "--quiet").Run(); err != nil {
		hasStaged = true
	}

	args := []string{"create", "--no-interactive", "-m", msg}
	if !hasStaged {
		args = append(args, "-a")
	}

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

		// Check for staged files
		hasStaged := false
		if err := exec.Command("git", "diff", "--cached", "--quiet").Run(); err != nil {
			hasStaged = true
		}

		// Construct create command
		// If staged changes exist, don't use -a. Otherwise use -a (commit all).
		cmdStr := "gt create --no-interactive -m"
		if !hasStaged {
			cmdStr = "gt create -a --no-interactive -m"
		}

		// Step 1: Create commit
		// Use UseShell: false to ensure arguments are passed correctly as exec args, avoiding shell quoting issues
		result := executor.Execute(ctx, ExecutionConfig{
			Command:   cmdStr,
			Args:      []string{commitMsg},
			SkipHooks: skipHooks,
			UseShell:  false,
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

		// Check for staged files
		hasStaged := false
		if err := exec.Command("git", "diff", "--cached", "--quiet").Run(); err != nil {
			hasStaged = true
		}

		// Construct modify command
		cmdStr := "gt modify --no-interactive -m"
		if !hasStaged {
			cmdStr = "gt modify -a --no-interactive -m"
		}

		// Step 1: Amend commit
		result := executor.Execute(ctx, ExecutionConfig{
			Command:   cmdStr,
			Args:      []string{commitMsg},
			SkipHooks: skipHooks,
			UseShell:  false,
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
