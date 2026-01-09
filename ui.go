package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// --- Theme & Aesthetics (Vercel x Ghostty x Starship) ---

const (
	// Palette
	colorBg        = "#000000" // Pure Black (OLED/Vercel)
	colorFg        = "#EDEDED" // Off-white text
	colorSub       = "#666666" // Dark Gray for subtitles
	colorBorder    = "#333333" // Subtle borders
	colorAccent    = "#0070F3" // Vercel Blue (Primary)
	colorSuccess   = "#50E3C2" // Teal/Green (Success/Added)
	colorWarning   = "#F5A623" // Orange (Modified/Warning)
	colorError     = "#FF0080" // Hot Pink (Deleted/Error/Ghost)
	colorHighlight = "#1A1A1A" // Dark highlight for active items
)

// --- Styles ---

var (
	// Layout
	docStyle = lipgloss.NewStyle().
			Margin(1, 2). // Add some breathing room
			Background(lipgloss.Color(colorBg)).
			Foreground(lipgloss.Color(colorFg))

	// Layout Blocks
	sidebarStyle = lipgloss.NewStyle().
			Width(25).
			PaddingRight(2).
			Border(lipgloss.NormalBorder(), false, true, false, false). // Right border only
			BorderForeground(lipgloss.Color(colorBorder))

	mainStageStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Width(60) // Will be dynamic in Resize

	// Typography
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorFg)).
			Bold(true).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSub)).
			Italic(true)

	// Navigation / Menu
	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSub)).
			PaddingLeft(1).
			MarginBottom(0)

	menuSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorAccent)).
				Background(lipgloss.Color(colorHighlight)).
				Bold(true).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color(colorAccent)).
				PaddingLeft(1)

	// Branch / Status Badges
	branchPillStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBg)).
			Background(lipgloss.Color(colorFg)).
			Bold(true).
			Padding(0, 1).
			MarginRight(1)

	statusAheadStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorBg)).
				Background(lipgloss.Color(colorSuccess)).
				Bold(true).
				Padding(0, 1)

	statusBehindStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorBg)).
				Background(lipgloss.Color(colorError)).
				Bold(true).
				Padding(0, 1)

	// Files
	fileAddedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess))
	fileModifiedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorWarning))
	fileDeletedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorError))
	fileUntrackedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorSub))

	// Wizard / Input
	wizardQuestionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorFg)).
				Bold(true).
				PaddingBottom(1)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorAccent)).
			Padding(0, 1).
			Width(50)

	// Output Window
	outputWindowStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(colorBorder)).
				Padding(1)

	// Cards (Copilot/Hints)
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(1).
			MarginTop(1)

	tipPrefixStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Bold(true).
			MarginRight(1)
)

// --- View Rendering ---

// View is the main render loop
func (m model) View() string {
	// 1. Calculate Layout Dimensions
	// Assuming m.width is set. If not, default to 80.
	w := m.width
	if w == 0 {
		w = 100
	}

	sidebarWidth := 25
	mainWidth := w - sidebarWidth - 6 // -6 for margins/borders
	if mainWidth < 40 {
		mainWidth = 40
	}

	sidebarStyle = sidebarStyle.Width(sidebarWidth)
	mainStageStyle = mainStageStyle.Width(mainWidth)

	// 2. Build Sidebar (Always present)
	sidebar := renderSidebar(m)

	// 3. Build Main Stage (Context dependent)
	var mainStage string
	switch m.state {
	case viewDashboard, viewMenu:
		mainStage = renderDashboardMain(m, mainWidth)
	case viewWizardType, viewWizardScope, viewWizardSummary:
		mainStage = renderWizardStage(m)
	case viewWizardPreview:
		mainStage = renderWizardPreviewStage(m)
	case viewInput:
		mainStage = renderInputStage(m)
	case viewConfirm:
		mainStage = renderConfirmStage(m)
	case viewStack:
		mainStage = renderStackStage(m)
	case viewRunning, viewOutput, viewPostCommit:
		mainStage = renderOutputStage(m)
	case viewHelp:
		mainStage = renderHelpStage()
	case viewUpdate:
		mainStage = renderUpdateStage(m)
	default:
		mainStage = renderDashboardMain(m, mainWidth)
	}

	// 4. Combine
	content := lipgloss.JoinHorizontal(lipgloss.Top,
		sidebar,
		mainStage,
	)

	return docStyle.Render(content)
}

// --- Sidebar Component ---

func renderSidebar(m model) string {
	// Header: Repo Info
	repoName := "Graphite TUI" // Could fetch actual repo name if we wanted
	header := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(repoName),
		subtitleStyle.Render(GetCurrentVersion()),
	)

	// Status: Branch Info
	branchInfo := renderSidebarBranchInfo(m.branch, m.ahead, m.behind)

	// Navigation Menu
	var menuItems []string
	menuItems = append(menuItems, subtitleStyle.Render("ACTIONS"))

	for i, item := range m.items {
		// Styling
		isSelected := m.cursor == i
		label := fmt.Sprintf("%s %s", item.key, item.title)

		// Use symbols for extra flair
		symbol := "  "
		if isSelected {
			symbol = "❯ " // Ghostty/Starship style arrow
			menuItems = append(menuItems, menuSelectedStyle.Render(symbol+label))
		} else {
			menuItems = append(menuItems, menuItemStyle.Render(symbol+label))
		}
	}

	// Settings Toggle
	hooksState := "ON"
	hooksStyle := subtitleStyle
	if m.skipHooks {
		hooksState = "OFF"
		hooksStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorWarning))
	}
	settings := lipgloss.JoinVertical(lipgloss.Left,
		subtitleStyle.Render("────"),
		hooksStyle.Render(fmt.Sprintf("[h] Hooks: %s", hooksState)),
	)

	// Flash message (shows temporarily after toggling)
	flashContent := ""
	if m.flashMessage != "" {
		flashContent = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Bold(true).
			Render("⚡ " + m.flashMessage)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		branchInfo,
		"\n",
		lipgloss.JoinVertical(lipgloss.Left, menuItems...),
		"\n\n",
		settings,
		flashContent,
	)
}

func renderSidebarBranchInfo(branch string, ahead, behind int) string {
	// Branch Name
	if branch == "" {
		branch = "loading..."
	}

	// Truncate branch if too long
	if len(branch) > 18 {
		branch = branch[:15] + "..."
	}

	branchRow := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Render(" "), // Git Branch Icon
		branch,
	)

	// Status Pills
	var statusPills []string
	if ahead > 0 {
		statusPills = append(statusPills, statusAheadStyle.Render(fmt.Sprintf("↑%d", ahead)))
	}
	if behind > 0 {
		statusPills = append(statusPills, statusBehindStyle.Render(fmt.Sprintf("↓%d", behind)))
	}

	content := []string{branchRow}
	if len(statusPills) > 0 {
		content = append(content, lipgloss.JoinHorizontal(lipgloss.Left, statusPills...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// --- Main Stage Components ---

func renderDashboardMain(m model, width int) string {
	// 1. Context Header with file count
	var header string
	if len(m.changedFiles) > 0 {
		fileWord := "file"
		if len(m.changedFiles) != 1 {
			fileWord = "files"
		}
		header = titleStyle.Render(fmt.Sprintf("Unsaved Changes (%d %s)", len(m.changedFiles), fileWord))
	} else {
		header = titleStyle.Render("Project Overview")
	}

	// 2. File List or Empty State
	var content string
	if len(m.changedFiles) > 0 {
		var files []string
		for _, f := range m.changedFiles {
			var statusLabel, styledLine string
			path := f.Path

			// Smart path truncation - keep filename, truncate directories
			maxLen := width - 12
			if maxLen < 20 {
				maxLen = 20
			}
			if len(path) > maxLen {
				// Show ...end of path
				path = "..." + path[len(path)-maxLen+3:]
			}

			switch f.Status {
			case "M", "MM":
				statusLabel = "M "
				styledLine = fileModifiedStyle.Render(statusLabel + path)
			case "A", "AM":
				statusLabel = "+ "
				styledLine = fileAddedStyle.Render(statusLabel + path)
			case "D":
				statusLabel = "- "
				styledLine = fileDeletedStyle.Render(statusLabel + path)
			case "R":
				statusLabel = "R "
				styledLine = fileModifiedStyle.Render(statusLabel + path)
			case "??":
				statusLabel = "? "
				styledLine = fileUntrackedStyle.Render(statusLabel + path)
			default:
				statusLabel = "● "
				styledLine = fileUntrackedStyle.Render(statusLabel + path)
			}
			files = append(files, "  "+styledLine)
		}
		// Limit files to avoid overflow
		if len(files) > 10 {
			remaining := len(files) - 10
			files = files[:10]
			files = append(files, subtitleStyle.Render(fmt.Sprintf("     ...and %d more", remaining)))
		}
		content = lipgloss.JoinVertical(lipgloss.Left, files...)
	} else {
		// "All Clear" State
		content = lipgloss.JoinVertical(lipgloss.Center,
			"\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSub)).Render("No changed files"),
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("Everything is clean"),
		)
	}

	// 3. Copilot / Guide Card (Prominent)
	guideContent := ""
	if m.cursor >= 0 && m.cursor < len(m.items) {
		item := m.items[m.cursor]

		// Contextual Tip
		tip := m.suggestion
		if tip != "" {
			tip = strings.TrimPrefix(tip, "suggestion: ")
		} else {
			tip = item.guide
		}

		guideContent = cardStyle.Width(width - 4).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.JoinHorizontal(lipgloss.Left,
					tipPrefixStyle.Render("TIP"),
					titleStyle.Render(item.title),
				),
				lipgloss.NewStyle().Foreground(lipgloss.Color(colorFg)).Render(tip),
			),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		"\n",
		guideContent,
	)
}

func renderWizardStage(m model) string {
	// Clean, centered, "Typeform" style
	var step, question, placeholder string

	switch m.state {
	case viewWizardType:
		step = "Step 1 of 4"
		question = "What kind of change is this?"
		// Render types as a clean list with examples
		var types []string
		for i, t := range m.commitTypes {
			prefix := "  "
			style := menuItemStyle
			if i == m.wizardTypeIdx {
				prefix = "❯ "
				style = menuSelectedStyle
			}
			// Format: "feat     New feature    'add dark mode'"
			exampleText := subtitleStyle.Render(fmt.Sprintf("\"%s\"", t.example))
			types = append(types, style.Render(fmt.Sprintf("%s%-10s %-16s", prefix, t.label, t.desc))+exampleText)
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			subtitleStyle.Render(step),
			wizardQuestionStyle.Render(question),
			"\n",
			lipgloss.JoinVertical(lipgloss.Left, types...),
			"\n",
			subtitleStyle.Render("[↑↓] Select  [Enter] Confirm  [Esc] Cancel"),
		)

	case viewWizardScope:
		step = "Step 2 of 4"
		question = "What part of the codebase? (Optional)"
		placeholder = "Component name: auth, navbar, api, etc."

	case viewWizardSummary:
		step = "Step 3 of 4"
		question = "Describe the change"
		placeholder = "Use present tense: 'add button' not 'added button'"
	}

	// Input view
	inputDisplay := inputBoxStyle.Render(m.textInput.View())

	// Add character count for summary step
	charCount := ""
	if m.state == viewWizardSummary {
		count := len(m.textInput.Value())
		charCount = subtitleStyle.Render(fmt.Sprintf("[%d/72 characters]", count))
	}

	if m.wizardError != "" {
		inputDisplay = lipgloss.JoinVertical(lipgloss.Left,
			inputDisplay,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)).Render("⚠ "+m.wizardError),
		)
	}

	// Build hint based on step
	hint := "[Enter] Next  [Esc] Cancel"
	if m.state == viewWizardScope || m.state == viewWizardSummary {
		hint = "[Enter] Next  [Backspace] Back  [Esc] Cancel"
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		subtitleStyle.Render(step),
		wizardQuestionStyle.Render(question),
		subtitleStyle.Render(placeholder),
		"\n",
		inputDisplay,
		charCount,
		"\n",
		subtitleStyle.Render(hint),
	)
}

func renderWizardPreviewStage(m model) string {
	// Show file count
	fileCount := len(m.changedFiles)
	fileText := fmt.Sprintf("%d file", fileCount)
	if fileCount != 1 {
		fileText += "s"
	}

	// Build the preview
	commitBox := lipgloss.NewStyle().
		Background(lipgloss.Color(colorHighlight)).
		Foreground(lipgloss.Color(colorFg)).
		Padding(0, 1).
		Render(m.lastCommitMsg)

	tipBox := cardStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("This will:"),
			"  • Create a new branch",
			fmt.Sprintf("  • Commit your %s", fileText),
			"  • You can share for review with [p]",
		),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		subtitleStyle.Render("Step 4 of 4"),
		wizardQuestionStyle.Render("Ready to commit?"),
		"\n",
		commitBox,
		"\n",
		tipBox,
		"\n",
		subtitleStyle.Render("[Enter] Commit  [Esc] Edit message  [Backspace] Back"),
	)
}

func renderConfirmStage(m model) string {
	var title, description, warning string
	var bullets []string

	switch m.confirmAction {
	case "merge":
		title = "Merge & Cleanup"
		description = "This will merge your approved PR and delete the local branch."
		warning = "Make sure your PR is approved first!"
		bullets = []string{
			"• Merge your PR on GitHub",
			"• Delete the local branch",
			"• Sync with remote",
		}
	case "ghost":
		title = "Rescue Mode"
		description = fmt.Sprintf("Creating new branch: %s", m.confirmBranch)
		warning = ""
		bullets = []string{
			"• Create branch with your changes",
			"• Rebase onto main",
			"• Sync with remote",
		}
	default:
		title = "Confirm Action"
		description = "Are you sure?"
	}

	content := []string{
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render(title),
		"\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorFg)).Render(description),
		"\n",
	}

	if len(bullets) > 0 {
		content = append(content, cardStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("This will:"),
				strings.Join(bullets, "\n"),
			),
		))
		content = append(content, "\n")
	}

	if warning != "" {
		content = append(content, lipgloss.NewStyle().Foreground(lipgloss.Color(colorWarning)).Render("⚠ "+warning))
		content = append(content, "\n")
	}

	content = append(content, "\n")
	content = append(content, menuSelectedStyle.Render("❯ [y] Confirm"))
	content = append(content, menuItemStyle.Render("  [n] Cancel"))

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func renderInputStage(m model) string {
	title := "Input"
	desc := "Enter value below"

	if m.isGhostFix {
		title = "👻 Rescue Mode (Ghost Fix)"
		desc = "Working on a merged branch? Name a new branch to save your work:"
	} else if m.cursor >= 0 && m.cursor < len(m.items) {
		title = m.items[m.cursor].title
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)).Bold(true).Render(title),
		subtitleStyle.Render(desc),
		"\n",
		inputBoxStyle.Render(m.textInput.View()),
		"\n",
		subtitleStyle.Render("Enter to confirm • Esc to cancel"),
	)
}

func renderOutputStage(m model) string {
	// Post-commit special success state
	if m.state == viewPostCommit {
		// Educational tip about Graphite workflow
		tipBox := cardStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("Graphite Tip"),
				"Submit now to create a PR and trigger",
				"preview builds. Small, frequent PRs get",
				"reviewed faster and merge easier!",
			),
		)

		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Bold(true).Render("✔ Changes Saved"),
			"\n",
			lipgloss.NewStyle().Background(lipgloss.Color(colorHighlight)).Padding(0, 1).Render(m.lastCommitMsg),
			"\n",
			tipBox,
			"\n",
			titleStyle.Render("What's next?"),
			menuSelectedStyle.Render("❯ [p] Share & Create PR  (recommended)"),
			menuItemStyle.Render("  [n] Continue Working"),
			menuItemStyle.Render("  [f] Fix (Amend)"),
		)
	}

	// Normal Output/Running
	var header string
	if m.state == viewRunning {
		header = lipgloss.JoinHorizontal(lipgloss.Left,
			m.spinner.View(),
			" Executing...",
		)
	} else {
		// Finished
		if m.err != nil {
			header = lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)).Bold(true).Render("✖ Command Failed")
		} else {
			header = lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Bold(true).Render("✔ Success")
		}
	}

	content := m.output
	if content == "" && m.err != nil {
		content = m.err.Error()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		subtitleStyle.Render(m.command),
		"\n",
		outputWindowStyle.Render(content),
		"\n",
		subtitleStyle.Render("Press Esc to return"),
	)
}

func renderStackStage(m model) string {
	header := titleStyle.Render("Stack Map (GPS)")

	if len(m.stackItems) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			subtitleStyle.Render("No stack found."),
		)
	}

	var lines []string
	for i, s := range m.stackItems {
		// Visual tree
		indent := strings.Repeat("  ", s.level)
		marker := "○"
		if s.current {
			marker = "●" // Current
		}

		label := fmt.Sprintf("%s%s %s", indent, marker, s.name)

		style := menuItemStyle
		if i == m.stackCursor {
			style = menuSelectedStyle
			label = fmt.Sprintf("%s%s %s", indent, "⦿", s.name)
		}

		lines = append(lines, style.Render(label))
	}

	// Legend
	legend := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("● current"),
		subtitleStyle.Render("  ○ branch  "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Render("⦿ selected"),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		lipgloss.JoinVertical(lipgloss.Left, lines...),
		"\n",
		legend,
		"\n",
		subtitleStyle.Render("[↑↓] Navigate  [Enter] Checkout  [r] Refresh  [Esc] Back"),
	)
}

func renderHelpStage() string {
	workflowSection := lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("WORKFLOW")
	navSection := lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("NAVIGATION")
	settingsSection := lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("SETTINGS")

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Help & Shortcuts"),
		"\n",
		workflowSection,
		"  s  Start      Create branch + commit → ready to stack!",
		"  p  Share      Submit PR → triggers preview build",
		"  f  Fix        Amend last commit (forgot something?)",
		"  y  Sync       Pull latest from your team",
		"  d  Done       Merge approved PR + cleanup",
		"\n",
		navSection,
		"  g  GPS        Visual stack map - see all branches",
		"  x  Rescue     Ghost fix - save work from merged branch",
		"  m  Menu       Full command menu",
		"  ?  Help       This screen",
		"  ↑↓ Navigate   Move selection up/down",
		"  ⏎  Select     Confirm selection",
		"  ⎋  Back       Cancel / go back",
		"\n",
		settingsSection,
		"  i  Init       Set up Graphite (first time only)",
		"  h  Hooks      Toggle pre-commit hooks on/off",
		"  u  Update     Check for updates",
		"  r  Refresh    Refresh git status",
		"\n",
		subtitleStyle.Render("Press Esc to return"),
	)
}

func renderUpdateStage(m model) string {
	content := []string{titleStyle.Render("Update Manager")}

	if m.checkingUpdate {
		content = append(content, m.spinner.View()+" Checking for updates...")
	} else if m.latestVersion != "" && isNewerVersion(currentVersion, m.latestVersion) {
		content = append(content,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("New version available: "+m.latestVersion),
			"\n",
			menuSelectedStyle.Render("❯ [u] Update Now"),
			menuItemStyle.Render("  [x] Uninstall"),
		)
	} else {
		content = append(content,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("You are up to date!"),
			subtitleStyle.Render(GetCurrentVersion()),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}
