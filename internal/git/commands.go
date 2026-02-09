package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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


// MergedAncestorIssue represents a merged ancestor branch
type MergedAncestorIssue struct {
	Branch   string
	PRNumber int
	PRURL    string
}

// MergedAncestorCheckMsg is sent after checking for merged ancestors
type MergedAncestorCheckMsg struct {
	Issues        []MergedAncestorIssue
	CurrentBranch string
	TrunkBranch   string
	Err           error
}

// --- Singleton Executor ---

var executor = NewCommandExecutor()


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


// CheckMergedAncestors inspects ancestors for merged PRs missing from trunk
func CheckMergedAncestors() tea.Cmd {
	return func() tea.Msg {
		trunk, err := getTrunkBranch()
		if err != nil {
			return MergedAncestorCheckMsg{Err: err}
		}

		currentBranch, err := getCurrentBranch()
		if err != nil {
			return MergedAncestorCheckMsg{Err: err, TrunkBranch: trunk}
		}

		ancestors, err := getAncestorBranches(currentBranch, trunk)
		if err != nil {
			return MergedAncestorCheckMsg{Err: err, CurrentBranch: currentBranch, TrunkBranch: trunk}
		}

		issues := []MergedAncestorIssue{}
		for _, branch := range ancestors {
			prInfo, ok := readPRInfo(branch)
			if !ok || !prInfo.Merged {
				continue
			}
			if prInfo.Number == 0 {
				continue
			}
			contains, err := trunkContainsBranch(trunk, branch)
			if err != nil {
				return MergedAncestorCheckMsg{Err: err, CurrentBranch: currentBranch, TrunkBranch: trunk}
			}
			if !contains {
				issues = append(issues, MergedAncestorIssue{
					Branch:   branch,
					PRNumber: prInfo.Number,
					PRURL:    prInfo.URL,
				})
			}
		}

		return MergedAncestorCheckMsg{
			Issues:        issues,
			CurrentBranch: currentBranch,
			TrunkBranch:   trunk,
		}
	}
}

// ExecuteResolveMergedAncestors resolves merged ancestor branches and rebases current branch
func ExecuteResolveMergedAncestors(trunk, current string, mergedBranches []string, skipHooks bool) tea.Cmd {
	return func() tea.Msg {
		if trunk == "" {
			return CmdFinishedMsg{Err: fmt.Errorf("trunk branch not found"), Command: "resolve merged ancestor"}
		}
		ctx := context.Background()
		configs := []ExecutionConfig{}

		currentIsMerged := false
		for _, branch := range mergedBranches {
			if branch == current {
				currentIsMerged = true
				break
			}
		}

		if currentIsMerged {
			configs = append(configs, GTCommand("checkout --trunk", false))
			current = trunk
		}

		configs = append(configs, ExecutionConfig{
			Command:      "gt sync",
			AllowUserEnv: true,
		})

		for _, branch := range mergedBranches {
			configs = append(configs, ExecutionConfig{
				Command: "gt delete --force " + branch,
			})
		}

		if current != trunk {
			configs = append(configs, ExecutionConfig{
				Command: "gt checkout " + current,
			})
			configs = append(configs, ExecutionConfig{
				Command: "gt rebase " + trunk + " --no-interactive",
			})
		}

		result := executor.ExecuteChain(ctx, configs)
		if result.Err == nil && result.Output == "" {
			result.Output = "Merged ancestor resolved."
		}
		result.Command = "resolve merged ancestor"
		return result
	}
}

// ExecuteTreatAsNew unlinks the PR so it can be resubmitted
func ExecuteTreatAsNew(skipHooks bool) tea.Cmd {
	return executor.ExecuteAsync(ExecutionConfig{
		Command:   "gt unlink --no-interactive",
		SkipHooks: skipHooks,
	})
}

// --- Merged Ancestor Helpers ---

type prInfoEntry struct {
	PRNumber int    `json:"prNumber"`
	State    string `json:"state"`
	URL      string `json:"url"`
	HeadRef  string `json:"headRefName"`
	MergedAt string `json:"mergedAt"`
}

type prInfoFile struct {
	PRInfos []prInfoEntry `json:"prInfos"`
}

type prInfoMatch struct {
	Number int
	URL    string
	Merged bool
}

func getTrunkBranch() (string, error) {
	out, err := exec.Command("gt", "trunk", "--no-interactive").Output()
	if err == nil {
		trunk := strings.TrimSpace(string(out))
		if trunk != "" {
			return trunk, nil
		}
	}

	trunk, err := readTrunkFromConfig()
	if err == nil && trunk != "" {
		return trunk, nil
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("trunk branch not found")
}

func readTrunkFromConfig() (string, error) {
	path := filepath.Join(".git", ".graphite_repo_config")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var payload struct {
		Trunk string `json:"trunk"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Trunk), nil
}

func getCurrentBranch() (string, error) {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getAncestorBranches(current, trunk string) ([]string, error) {
	out, err := exec.Command("gt", "log", "short", "--stack", "--classic", "--no-interactive").Output()
	if err != nil {
		return nil, err
	}

	branches := []string{}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		name := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		name = strings.TrimSpace(strings.TrimLeft(name, "○●◉↱$"))
		name = strings.TrimSpace(strings.TrimLeft(name, "•"))
		if name == "" {
			continue
		}
		name = strings.TrimSpace(strings.TrimPrefix(name, "$"))
		if idx := strings.Index(name, "("); idx != -1 {
			name = strings.TrimSpace(name[:idx])
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if name == trunk {
			continue
		}
		branches = append(branches, name)
		if name == current {
			break
		}
	}

	return branches, nil
}

func readPRInfo(branch string) (prInfoMatch, bool) {
	path := filepath.Join(".git", ".graphite_pr_info")
	data, err := os.ReadFile(path)
	if err != nil {
		return prInfoMatch{}, false
	}

	var payload prInfoFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return prInfoMatch{}, false
	}

	for _, info := range payload.PRInfos {
		if info.HeadRef == branch {
			return prInfoMatch{
				Number: info.PRNumber,
				URL:    info.URL,
				Merged: strings.EqualFold(info.State, "MERGED") || info.MergedAt != "",
			}, true
		}
	}
	return prInfoMatch{}, false
}

func trunkContainsBranch(trunk, branch string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", branch, trunk)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
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
		commitMsg = strings.TrimSpace(commitMsg)
		if commitMsg == "" {
			commitMsg = getDefaultCommitMessage()
		}
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
