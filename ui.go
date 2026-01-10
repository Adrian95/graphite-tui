package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// --- Theme & Aesthetics (Vercel Black + Hot Pink) ---

const (
	// Palette - Minimalist dark with hot pink accent
	colorBg        = "#000000" // Pure Black (OLED/Vercel)
	colorFg        = "#EDEDED" // Off-white text
	colorSub       = "#666666" // Dark Gray for subtitles
	colorBorder    = "#333333" // Subtle borders
	colorAccent    = "#FF0080" // Hot Pink (Primary accent)
	colorSuccess   = "#50E3C2" // Teal/Green (Success/Added)
	colorWarning   = "#F5A623" // Orange (Modified/Warning)
	colorError     = "#FF3366" // Bright red-pink for errors
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
			Width(20).
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

	// Footer bar
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSub)).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color(colorBorder)).
			PaddingTop(0).
			MarginTop(1)

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Bold(true)

	footerLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorSub))
)

// --- View Rendering ---

// View is the main render loop
func (m model) View() string {
	// 1. Calculate Layout Dimensions
	w := m.width
	if w == 0 {
		w = 100
	}

	sidebarWidth := 20
	mainWidth := w - sidebarWidth - 6
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

	// 4. Footer (keyboard shortcuts)
	footer := renderFooter(w)

	// 5. Combine
	content := lipgloss.JoinHorizontal(lipgloss.Top,
		sidebar,
		mainStage,
	)

	return docStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content, footer))
}

// --- Footer Component ---

func renderFooter(width int) string {
	shortcuts := []struct {
		key   string
		label string
	}{
		{"s", "Start"},
		{"p", "Share"},
		{"f", "Fix"},
		{"y", "Sync"},
		{"d", "Done"},
		{"g", "GPS"},
		{"?", "Help"},
		{"q", "Quit"},
	}

	var parts []string
	for _, s := range shortcuts {
		parts = append(parts, footerKeyStyle.Render(s.key)+" "+footerLabelStyle.Render(s.label))
	}

	content := strings.Join(parts, "  ")
	return footerStyle.Width(width - 4).Render(content)
}

// --- Sidebar Component ---

func renderSidebar(m model) string {
	// Header: App name + version
	header := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorAccent)).Render("Graphite TUI"),
		subtitleStyle.Render(GetCurrentVersion()),
	)

	// Settings Toggle
	hooksState := "ON"
	hooksColor := colorSuccess
	if m.skipHooks {
		hooksState = "OFF"
		hooksColor = colorWarning
	}
	hooksLine := lipgloss.JoinHorizontal(lipgloss.Left,
		subtitleStyle.Render("Hooks "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(hooksColor)).Bold(true).Render(hooksState),
	)

	// Flash message (shows temporarily)
	flashContent := ""
	if m.flashMessage != "" {
		flashContent = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Italic(true).
			Render(m.flashMessage)
	}

	// Update available indicator
	updateIndicator := ""
	if m.updateAvailable != "" {
		updateIndicator = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSuccess)).
			Render("↑ " + m.updateAvailable)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		hooksLine,
		updateIndicator,
		flashContent,
	)
}

// --- Main Stage Components ---

func renderDashboardMain(m model, width int) string {
	// 1. Branch Header (prominent at top)
	branch := m.branch
	if branch == "" {
		branch = "loading..."
	}

	branchDisplay := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorAccent)).
		Bold(true).
		Render(" " + branch)

	// Status badges
	var badges []string
	if m.ahead > 0 {
		badges = append(badges, statusAheadStyle.Render(fmt.Sprintf(" ↑%d ", m.ahead)))
	}
	if m.behind > 0 {
		badges = append(badges, statusBehindStyle.Render(fmt.Sprintf(" ↓%d ", m.behind)))
	}

	branchLine := branchDisplay
	if len(badges) > 0 {
		branchLine = lipgloss.JoinHorizontal(lipgloss.Left,
			branchDisplay,
			"  ",
			lipgloss.JoinHorizontal(lipgloss.Left, badges...),
		)
	}

	// 2. File List Header with expand hint
	var filesHeader string
	if len(m.changedFiles) > 0 {
		expandHint := "[Tab] expand"
		if m.filesExpanded {
			expandHint = "[Tab] collapse"
		}
		filesHeader = lipgloss.JoinHorizontal(lipgloss.Left,
			titleStyle.Render(fmt.Sprintf("Changes (%d)", len(m.changedFiles))),
			"  ",
			subtitleStyle.Render(expandHint),
		)
	} else {
		filesHeader = titleStyle.Render("No Changes")
	}

	// 3. File List (expandable with Tab)
	var fileList string
	if len(m.changedFiles) > 0 {
		var files []string
		maxFiles := 8
		if m.filesExpanded {
			maxFiles = 50
		}

		for i, f := range m.changedFiles {
			if i >= maxFiles {
				remaining := len(m.changedFiles) - maxFiles
				files = append(files, subtitleStyle.Render(fmt.Sprintf("  ...and %d more", remaining)))
				break
			}

			path := f.Path
			maxLen := width - 8
			if maxLen < 30 {
				maxLen = 30
			}
			if len(path) > maxLen {
				path = "..." + path[len(path)-maxLen+3:]
			}

			var styledLine string
			switch f.Status {
			case "M", "MM":
				styledLine = fileModifiedStyle.Render("M " + path)
			case "A", "AM":
				styledLine = fileAddedStyle.Render("+ " + path)
			case "D":
				styledLine = fileDeletedStyle.Render("- " + path)
			case "R":
				styledLine = fileModifiedStyle.Render("R " + path)
			case "??":
				styledLine = fileUntrackedStyle.Render("? " + path)
			default:
				styledLine = fileUntrackedStyle.Render("● " + path)
			}
			files = append(files, "  "+styledLine)
		}
		fileList = lipgloss.JoinVertical(lipgloss.Left, files...)
	} else {
		fileList = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSuccess)).
			Render("  ✓ Working tree clean")
	}

	// 4. Contextual Tip (based on actual state, not menu)
	tip := getContextualTip(m)
	tipCard := cardStyle.Width(width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			tipPrefixStyle.Render("NEXT"),
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorFg)).Render(tip),
		),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		branchLine,
		"\n",
		filesHeader,
		fileList,
		"\n",
		tipCard,
	)
}

// getContextualTip returns workflow advice based on current git/graphite state
func getContextualTip(m model) string {
	// Not initialized
	if !m.gtInitialized {
		return "Press [i] to initialize Graphite in this repo"
	}

	// On main branch
	if m.onMain {
		if len(m.changedFiles) > 0 {
			return "Press [s] to save changes to a new branch"
		}
		return "Press [s] to start new work"
	}

	// On a feature branch
	if len(m.changedFiles) > 0 {
		return "Press [f] to add to current commit, or [s] for a new stacked branch"
	}

	// No changes, check sync status
	if m.ahead > 0 {
		return "Press [p] to submit for review → get a preview build"
	}

	if m.behind > 0 {
		return "Press [y] to sync with latest changes from your team"
	}

	// All synced up
	return "All synced! Press [d] when your PR is approved to merge"
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
		title = "Rescue Mode (Ghost Fix)"
		desc = "Working on a merged branch? Name a new branch to save your work:"
	} else if m.cursor >= 0 && m.cursor < len(m.items) {
		title = m.items[m.cursor].title
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render(title),
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
			menuSelectedStyle.Render("❯ [y] Share & Create PR  (recommended)"),
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
		"  Tab Expand    Show/hide full file list",
		"  ?  Help       This screen",
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
