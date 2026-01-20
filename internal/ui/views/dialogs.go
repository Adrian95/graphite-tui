package views

import (
	"fmt"
	"strings"

	"github.com/Adrian95/graphite-tui/v2/internal/git"
	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// --- Confirm Dialog ---

// ConfirmViewData contains confirmation dialog data
type ConfirmViewData struct {
	Action     string // "merge", "ghost", "reset"
	BranchName string // For ghost fix
}

// RenderConfirm renders the confirmation dialog
func RenderConfirm(data ConfirmViewData) string {
	var title, description, warning string
	var bullets []string

	switch data.Action {
	case "merge":
		title = "Merge & Cleanup"
		description = "Merge approved PR and delete local branch."
		warning = "Make sure your PR is approved!"
		bullets = []string{"• Merge PR on GitHub", "• Delete local branch", "• Sync with remote"}
	case "ghost":
		title = "Rescue Mode"
		description = fmt.Sprintf("Creating new branch: %s", data.BranchName)
		bullets = []string{"• Create branch with changes", "• Rebase onto main", "• Sync with remote"}
	case "reset":
		title = "Hard Reset"
		description = "Discard ALL uncommitted changes."
		warning = "This cannot be undone!"
		bullets = []string{"• Discard all modified files", "• Remove untracked files", "• Reset to last commit"}
	default:
		title = "Confirm Action"
		description = "Are you sure?"
	}

	content := []string{
		lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render(title),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorFg)).Render(description),
	}

	if len(bullets) > 0 {
		content = append(content, "")
		content = append(content, ui.CardStyle.Render(strings.Join(bullets, "\n")))
	}

	if warning != "" {
		content = append(content, "")
		content = append(content, lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorWarning)).Render("⚠ "+warning))
	}

	content = append(content, "")
	content = append(content, ui.MenuSelectedStyle.Render("❯ [y] Confirm"))
	content = append(content, ui.MenuItemStyle.Render("  [n] Cancel"))

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// --- Merged Ancestor Dialog ---

// MergedAncestorViewData contains merged ancestor dialog data
// Option: 0 = Resolve, 1 = Abort, 2 = Treat as New
type MergedAncestorViewData struct {
	Issues        []git.MergedAncestorIssue
	Selected      int
	CurrentBranch string
	TrunkBranch   string
}

// RenderMergedAncestor renders the merged ancestor resolution dialog
func RenderMergedAncestor(data MergedAncestorViewData) string {
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorWarning)).Bold(true)
	title := warningStyle.Render("Merged Parent Detected")

	issueLine := ""
	if len(data.Issues) > 0 {
		issue := data.Issues[0]
		prLabel := "PR"
		if issue.PRNumber > 0 {
			prLabel = fmt.Sprintf("PR #%d", issue.PRNumber)
		}
		issueLine = fmt.Sprintf("%s merged for %s", prLabel, issue.Branch)
		if issue.PRURL != "" {
			issueLine += fmt.Sprintf(" (%s)", issue.PRURL)
		}
	}

	additional := ""
	if len(data.Issues) > 1 {
		additional = fmt.Sprintf("+%d more merged parent(s)", len(data.Issues)-1)
	}

	contextLine := fmt.Sprintf("Local %s does not include merged changes.", data.TrunkBranch)

	optionStyle := ui.MenuItemStyle
	selectedStyle := ui.MenuSelectedStyle
	options := []struct {
		Label       string
		Desc        string
		Recommended bool
	}{
		{
			Label:       "Sync trunk, delete merged branch, restack",
			Desc:        "Auto-clean local graph and continue.",
			Recommended: true,
		},
		{
			Label: "Abort",
			Desc:  "Exit to shell to handle manually.",
		},
		{
			Label: "Treat as new",
			Desc:  "Unlink PR and create a new one.",
		},
	}

	var optionLines []string
	for idx, option := range options {
		label := option.Label
		if option.Recommended {
			label += " (Recommended)"
		}
		prefix := "  "
		style := optionStyle
		if data.Selected == idx {
			prefix = "▸ "
			style = selectedStyle
		}
		line := style.Render(prefix + label)
		desc := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSub)).Render("    " + option.Desc)
		optionLines = append(optionLines, lipgloss.JoinVertical(lipgloss.Left, line, desc))
	}

	content := []string{
		title,
		"",
		ui.SubtitleStyle.Render(issueLine),
	}
	if additional != "" {
		content = append(content, ui.SubtitleStyle.Render(additional))
	}
	content = append(content,
		ui.SubtitleStyle.Render(contextLine),
		"",
		ui.CardStyle.Render(strings.Join(optionLines, "\n\n")),
		"",
		ui.SubtitleStyle.Render("[↑↓] Select  [Enter] Confirm  [Esc] Cancel"),
	)

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// --- Input Dialog ---

// InputViewData contains input dialog data
type InputViewData struct {
	Title       string
	Description string
	InputView   string
}

// RenderInput renders the input dialog
func RenderInput(data InputViewData) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render(data.Title),
		ui.SubtitleStyle.Render(data.Description),
		"",
		ui.InputBoxStyle.Render(data.InputView),
		"",
		ui.SubtitleStyle.Render("Enter to confirm • Esc to cancel"),
	)
}

// --- Output/Running View ---

// OutputViewData contains command output data
type OutputViewData struct {
	IsRunning     bool
	Output        string
	Error         error
	Command       string
	SpinnerView   string
	CommitMessage string // For post-commit view
	IsPostCommit  bool
}

// RenderOutput renders the command output view
func RenderOutput(data OutputViewData) string {
	if data.IsPostCommit {
		return renderPostCommit(data)
	}

	var header string
	if data.IsRunning {
		header = lipgloss.JoinHorizontal(lipgloss.Left, data.SpinnerView, " Running...")
	} else {
		if data.Error != nil {
			header = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorError)).Bold(true).Render("✖ Failed")
		} else {
			header = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Bold(true).Render("✔ Done")
		}
	}

	content := data.Output
	if content == "" && data.Error != nil {
		content = data.Error.Error()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		ui.SubtitleStyle.Render(data.Command),
		"",
		ui.OutputWindowStyle.Render(content),
		"",
		ui.SubtitleStyle.Render("[Esc] Back"),
	)
}

func renderPostCommit(data OutputViewData) string {
	tipBox := ui.CardStyle.Render(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render("Tip: ") +
			"Submit now to create a PR and trigger preview builds.",
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Bold(true).Render("✔ Committed"),
		"",
		ui.HighlightBoxStyle.Render(data.CommitMessage),
		"",
		tipBox,
		"",
		ui.MenuSelectedStyle.Render("❯ [y] Submit PR"),
		ui.MenuItemStyle.Render("  [n] Continue"),
		ui.MenuItemStyle.Render("  [f] Amend"),
	)
}

// --- Stack View ---

// StackViewData contains stack map data
type StackViewData struct {
	Items  []git.StackItem
	Cursor int
}

// RenderStack renders the stack map view
func RenderStack(data StackViewData) string {
	header := ui.TitleStyle.Render("Stack Map")

	if len(data.Items) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			ui.SubtitleStyle.Render("No stack found."),
		)
	}

	var lines []string
	for i, s := range data.Items {
		indent := strings.Repeat("  ", s.Level)
		marker := "○"
		if s.Current {
			marker = "●"
		}

		label := fmt.Sprintf("%s%s %s", indent, marker, s.Name)

		style := ui.MenuItemStyle
		if i == data.Cursor {
			style = ui.MenuSelectedStyle
			label = fmt.Sprintf("%s%s %s", indent, "⦿", s.Name)
		}

		lines = append(lines, style.Render(label))
	}

	legend := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Render("● you"),
		ui.SubtitleStyle.Render("  ○ branch  "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Render("⦿ selected"),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		lipgloss.JoinVertical(lipgloss.Left, lines...),
		"",
		legend,
		"",
		ui.SubtitleStyle.Render("[↑↓] Navigate  [Enter] Checkout  [Esc] Back"),
	)
}

// --- Help View ---

// RenderHelp renders the help view
func RenderHelp() string {
	speedSection := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render("SPEED (↑↓ to select, ⏎ to run)")
	workflowSection := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render("WORKFLOW")
	navSection := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render("NAVIGATION")

	return lipgloss.JoinVertical(lipgloss.Left,
		ui.TitleStyle.Render("Help"),
		"",
		speedSection,
		"  Ship     Smart: creates OR amends based on context",
		"  Iterate  Amend last commit + push to update PR",
		"  Reset    Discard all uncommitted changes",
		"  Undo     Reverse last Graphite operation",
		"",
		workflowSection,
		"  s   Start     Create new branch + commit (wizard)",
		"  p   Share     Submit PR → preview build",
		"  y   Sync      Pull latest from team",
		"  d   Done      Merge approved PR",
		"",
		navSection,
		"  g   Stack     Visual stack map",
		"  x   Rescue    Ghost fix for merged branches",
		"  Tab/←/→ Focus Cycle panels",
		"  h   Hooks     Toggle pre-commit hooks",
		"  u   Update    Check for updates",
		"",
		ui.SubtitleStyle.Render("[Esc] Back"),
	)
}

// --- Quick Commit View (lazygit-inspired) ---

// QuickCommitViewData contains quick commit dialog data
type QuickCommitViewData struct {
	FileCount  int
	InputValue string
	ErrorMsg   string
	IsAmend    bool // true for iterate, false for new commit
}

// RenderQuickCommit renders a minimal commit message input
func RenderQuickCommit(data QuickCommitViewData, inputView string) string {
	// Title based on mode
	title := "Ship"
	if data.IsAmend {
		title = "Iterate"
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorAccent)).
		Bold(true)

	// File count badge
	fileText := fmt.Sprintf("%d file", data.FileCount)
	if data.FileCount != 1 {
		fileText += "s"
	}
	fileBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorSuccess)).
		Render(fileText)

	// Header
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		titleStyle.Render(title),
		"  ",
		fileBadge,
	)

	// Commit types hint (compact, lazygit style)
	typeHints := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorSub)).
		Render("feat: fix: docs: refactor: chore:")

	// Input display
	inputDisplay := ui.InputBoxStyle.Render(inputView)

	// Error message if any
	if data.ErrorMsg != "" {
		inputDisplay = lipgloss.JoinVertical(lipgloss.Left,
			inputDisplay,
			lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorError)).Render("  "+data.ErrorMsg),
		)
	}

	// Character count
	charStyle := ui.SubtitleStyle
	charCount := len(data.InputValue)
	if charCount > 72 {
		charStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorWarning))
	}
	charIndicator := charStyle.Render(fmt.Sprintf("%d/72", charCount))

	// Footer
	footer := ui.SubtitleStyle.Render("[Enter] Ship  [Esc] Cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		typeHints,
		"",
		inputDisplay,
		charIndicator,
		"",
		footer,
	)
}

// --- Commit Choice View (Amend vs Stack) ---

// CommitChoiceViewData contains commit choice dialog data
type CommitChoiceViewData struct {
	Branch    string // Current branch name
	FileCount int    // Number of changed files
	Selected  int    // 0 = Amend, 1 = Stack
}

// RenderCommitChoice renders the amend vs stack selection dialog
func RenderCommitChoice(data CommitChoiceViewData) string {
	// Title with branch name
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorAccent)).
		Bold(true)

	// File count badge
	fileText := fmt.Sprintf("%d file", data.FileCount)
	if data.FileCount != 1 {
		fileText += "s"
	}
	fileBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorSuccess)).
		Render(fileText)

	header := lipgloss.JoinHorizontal(lipgloss.Left,
		titleStyle.Render("Ship: "+data.Branch),
		"  ",
		fileBadge,
	)

	// Question
	question := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorFg)).
		Render("How do you want to ship these changes?")

	// Option styling
	selectedStyle := ui.MenuSelectedStyle
	normalStyle := ui.MenuItemStyle
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSub))

	// Amend option
	amendTitle := "  Amend current PR"
	if data.Selected == 0 {
		amendTitle = "▸ Amend current PR"
	}
	amendDesc := []string{
		"    Updates your existing commit and PR.",
		"    Reviewers see one unified change.",
		"    Best for: small fixes, typos, review feedback",
	}

	// Stack option
	stackTitle := "  Stack new commit"
	if data.Selected == 1 {
		stackTitle = "▸ Stack new commit"
	}
	stackDesc := []string{
		"    Creates a NEW branch & PR on top of this one.",
		"    Original PR stays unchanged.",
		"    Best for: separate fix, want distinct review",
	}

	// Build options
	var amendOption, stackOption string
	if data.Selected == 0 {
		amendOption = selectedStyle.Render(amendTitle)
	} else {
		amendOption = normalStyle.Render(amendTitle)
	}
	for _, line := range amendDesc {
		amendOption = lipgloss.JoinVertical(lipgloss.Left, amendOption, descStyle.Render(line))
	}

	if data.Selected == 1 {
		stackOption = selectedStyle.Render(stackTitle)
	} else {
		stackOption = normalStyle.Render(stackTitle)
	}
	for _, line := range stackDesc {
		stackOption = lipgloss.JoinVertical(lipgloss.Left, stackOption, descStyle.Render(line))
	}

	// Footer
	footer := ui.SubtitleStyle.Render("[↑↓] Select  [Enter] Confirm  [Esc] Cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		question,
		"",
		amendOption,
		"",
		stackOption,
		"",
		footer,
	)
}

// --- Update View ---

// UpdateViewData contains update view data
type UpdateViewData struct {
	CurrentVersion string
	LatestVersion  string
	IsChecking     bool
	IsNewer        bool
	SpinnerView    string
}

// RenderUpdate renders the update view
func RenderUpdate(data UpdateViewData) string {
	content := []string{ui.TitleStyle.Render("Update")}

	if data.IsChecking {
		content = append(content, data.SpinnerView+" Checking...")
	} else if data.IsNewer {
		content = append(content,
			lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Render("New: "+data.LatestVersion),
			"",
			ui.MenuSelectedStyle.Render("❯ [u] Update"),
			ui.MenuItemStyle.Render("  [x] Uninstall"),
		)
	} else {
		content = append(content,
			lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Render("Up to date!"),
			ui.SubtitleStyle.Render(data.CurrentVersion),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}
