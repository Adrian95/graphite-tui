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
			Margin(1, 2).
			Background(lipgloss.Color(colorBg)).
			Foreground(lipgloss.Color(colorFg))

	// Typography
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorFg)).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSub))

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(0, 1)

	boxTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSub)).
			Bold(true)

	// Navigation / Menu
	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSub))

	menuSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorAccent)).
				Bold(true)

	// Branch / Status Badges
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

	// Cards (Tips)
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(0, 1)

	// Footer bar
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSub)).
			PaddingTop(1)

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Bold(true)

	footerLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorSub))
)

// --- View Rendering ---

func (m model) View() string {
	w := m.width
	if w == 0 {
		w = 100
	}

	// Layout dimensions
	sidebarWidth := 24
	mainWidth := w - sidebarWidth - 4
	if mainWidth < 50 {
		mainWidth = 50
	}

	// Build components based on state
	var mainStage string
	switch m.state {
	case viewDashboard, viewMenu:
		sidebar := renderSidebar(m, sidebarWidth)
		mainStage = renderDashboardMain(m, mainWidth)
		footer := renderFooter(w)

		content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainStage)
		return docStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content, footer))

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
		sidebar := renderSidebar(m, sidebarWidth)
		mainStage = renderDashboardMain(m, mainWidth)
		footer := renderFooter(w)

		content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainStage)
		return docStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content, footer))
	}

	return docStyle.Render(mainStage)
}

// --- Footer Component ---

func renderFooter(width int) string {
	shortcuts := []struct {
		key   string
		label string
	}{
		{"s", "New"},
		{"p", "PR"},
		{"y", "Sync"},
		{"d", "Merge"},
		{"g", "Stack"},
		{"?", "Help"},
	}

	var parts []string
	for _, s := range shortcuts {
		parts = append(parts, footerKeyStyle.Render(s.key)+" "+footerLabelStyle.Render(s.label))
	}

	line := strings.Repeat("─", width-4)
	content := strings.Join(parts, "   ")
	return footerStyle.Render(subtitleStyle.Render(line) + "\n" + content)
}

// --- Sidebar Component ---

func renderSidebar(m model, width int) string {
	// App Title
	appTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorAccent)).
		Bold(true).
		Render("GRAPHITE TUI")

	version := subtitleStyle.Render(GetCurrentVersion())

	// Speed Box - quick actions for AI workflow with cursor navigation
	speedItems := []struct {
		key   string
		label string
	}{
		{"⏎", "Ship"},
		{"f", "Iterate"},
		{"R", "Reset"},
		{"z", "Undo"},
	}

	var speedActions []string
	for i, item := range speedItems {
		var line string
		if i == m.speedCursor {
			// Selected item
			line = menuSelectedStyle.Render("❯ "+item.key) + " " + menuSelectedStyle.Render(item.label)
		} else {
			line = menuItemStyle.Render("  "+item.key) + " " + menuItemStyle.Render(item.label)
		}
		speedActions = append(speedActions, line)
	}

	speedContent := lipgloss.JoinVertical(lipgloss.Left, speedActions...)
	speedBox := boxStyle.Width(width - 2).Render(
		boxTitleStyle.Render("SPEED") + "\n" + speedContent,
	)

	// Status section
	hooksState := "ON"
	hooksColor := colorSuccess
	if m.skipHooks {
		hooksState = "OFF"
		hooksColor = colorWarning
	}
	hooksLine := subtitleStyle.Render("[h] Hooks ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color(hooksColor)).Render(hooksState)

	// Update indicator
	updateLine := ""
	if m.updateAvailable != "" {
		updateLine = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("↑ "+m.updateAvailable+" [u]")
	}

	// Flash message
	flashLine := ""
	if m.flashMessage != "" {
		flashLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Italic(true).
			Render(m.flashMessage)
	}

	return lipgloss.NewStyle().Width(width).PaddingRight(2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			appTitle,
			version,
			"",
			speedBox,
			"",
			hooksLine,
			updateLine,
			flashLine,
		),
	)
}

// --- Main Stage Components ---

func renderDashboardMain(m model, width int) string {
	// Branch Header Box
	branch := m.branch
	if branch == "" {
		branch = "loading..."
	}

	branchIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Render("")
	branchName := lipgloss.NewStyle().Foreground(lipgloss.Color(colorFg)).Bold(true).Render(branch)

	var badges []string
	if m.ahead > 0 {
		badges = append(badges, statusAheadStyle.Render(fmt.Sprintf("↑%d", m.ahead)))
	}
	if m.behind > 0 {
		badges = append(badges, statusBehindStyle.Render(fmt.Sprintf("↓%d", m.behind)))
	}

	branchLine := branchIcon + " " + branchName
	if len(badges) > 0 {
		branchLine += "  " + strings.Join(badges, " ")
	}

	branchBox := boxStyle.Width(width - 2).Render(branchLine)

	// Files Box
	var filesContent string
	filesTitle := boxTitleStyle.Render("CHANGES")

	if len(m.changedFiles) > 0 {
		expandHint := subtitleStyle.Render(fmt.Sprintf(" (%d) [Tab]", len(m.changedFiles)))
		filesTitle = boxTitleStyle.Render("CHANGES") + expandHint

		var files []string
		maxFiles := 6
		if m.filesExpanded {
			maxFiles = 30
		}

		for i, f := range m.changedFiles {
			if i >= maxFiles {
				remaining := len(m.changedFiles) - maxFiles
				files = append(files, subtitleStyle.Render(fmt.Sprintf("...%d more", remaining)))
				break
			}

			path := f.Path
			maxLen := width - 10
			if maxLen < 20 {
				maxLen = 20
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
				styledLine = fileUntrackedStyle.Render("• " + path)
			}
			files = append(files, styledLine)
		}
		filesContent = lipgloss.JoinVertical(lipgloss.Left, files...)
	} else {
		filesContent = lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("✓ Clean")
	}

	filesBox := boxStyle.Width(width - 2).Render(filesTitle + "\n" + filesContent)

	// Tip Box
	tip := getContextualTip(m)
	tipBox := cardStyle.Width(width - 2).Render(
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("→ ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorFg)).Render(tip),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		branchBox,
		"",
		filesBox,
		"",
		tipBox,
	)
}

// getContextualTip returns context-aware tip based on selected speed action and git state
func getContextualTip(m model) string {
	if !m.gtInitialized {
		return "Press [i] to initialize Graphite"
	}

	hasChanges := len(m.changedFiles) > 0

	switch m.speedCursor {
	case 0: // Ship
		if hasChanges {
			if m.onMain {
				return "Creates branch → commits → submits PR"
			}
			return "Amends commit → pushes → updates PR"
		} else if m.ahead > 0 {
			return "Submits PR for review"
		}
		return "Nothing to ship"

	case 1: // Iterate
		if hasChanges {
			return "Adds changes to last commit → pushes"
		}
		return "Nothing to amend"

	case 2: // Reset
		if hasChanges {
			return "Discards ALL uncommitted changes"
		}
		return "Working tree is clean"

	case 3: // Undo
		return "Reverses last Graphite operation"
	}

	return "[↑↓] navigate  [⏎] execute"
}

func renderWizardStage(m model) string {
	var step, question, placeholder string

	switch m.state {
	case viewWizardType:
		step = "Step 1 of 4"
		question = "What kind of change is this?"
		var types []string
		for i, t := range m.commitTypes {
			prefix := "  "
			style := menuItemStyle
			if i == m.wizardTypeIdx {
				prefix = "❯ "
				style = menuSelectedStyle
			}
			exampleText := subtitleStyle.Render(fmt.Sprintf(" \"%s\"", t.example))
			types = append(types, style.Render(fmt.Sprintf("%s%-10s %-16s", prefix, t.label, t.desc))+exampleText)
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			subtitleStyle.Render(step),
			wizardQuestionStyle.Render(question),
			"",
			lipgloss.JoinVertical(lipgloss.Left, types...),
			"",
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

	inputDisplay := inputBoxStyle.Render(m.textInput.View())

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

	hint := "[Enter] Next  [Esc] Cancel"
	if m.state == viewWizardScope || m.state == viewWizardSummary {
		hint = "[Enter] Next  [Backspace] Back  [Esc] Cancel"
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		subtitleStyle.Render(step),
		wizardQuestionStyle.Render(question),
		subtitleStyle.Render(placeholder),
		"",
		inputDisplay,
		charCount,
		"",
		subtitleStyle.Render(hint),
	)
}

func renderWizardPreviewStage(m model) string {
	fileCount := len(m.changedFiles)
	fileText := fmt.Sprintf("%d file", fileCount)
	if fileCount != 1 {
		fileText += "s"
	}

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
		"",
		commitBox,
		"",
		tipBox,
		"",
		subtitleStyle.Render("[Enter] Commit  [Esc] Edit  [Backspace] Back"),
	)
}

func renderConfirmStage(m model) string {
	var title, description, warning string
	var bullets []string

	switch m.confirmAction {
	case "merge":
		title = "Merge & Cleanup"
		description = "Merge approved PR and delete local branch."
		warning = "Make sure your PR is approved!"
		bullets = []string{"• Merge PR on GitHub", "• Delete local branch", "• Sync with remote"}
	case "ghost":
		title = "Rescue Mode"
		description = fmt.Sprintf("Creating new branch: %s", m.confirmBranch)
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
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render(title),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorFg)).Render(description),
	}

	if len(bullets) > 0 {
		content = append(content, "")
		content = append(content, cardStyle.Render(strings.Join(bullets, "\n")))
	}

	if warning != "" {
		content = append(content, "")
		content = append(content, lipgloss.NewStyle().Foreground(lipgloss.Color(colorWarning)).Render("⚠ "+warning))
	}

	content = append(content, "")
	content = append(content, menuSelectedStyle.Render("❯ [y] Confirm"))
	content = append(content, menuItemStyle.Render("  [n] Cancel"))

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func renderInputStage(m model) string {
	title := "Input"
	desc := "Enter value below"

	if m.isGhostFix {
		title = "Rescue Mode"
		desc = "Name a new branch to save your work:"
	} else if m.cursor >= 0 && m.cursor < len(m.items) {
		title = m.items[m.cursor].title
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render(title),
		subtitleStyle.Render(desc),
		"",
		inputBoxStyle.Render(m.textInput.View()),
		"",
		subtitleStyle.Render("Enter to confirm • Esc to cancel"),
	)
}

func renderOutputStage(m model) string {
	if m.state == viewPostCommit {
		tipBox := cardStyle.Render(
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("Tip: ") +
				"Submit now to create a PR and trigger preview builds.",
		)

		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Bold(true).Render("✔ Committed"),
			"",
			lipgloss.NewStyle().Background(lipgloss.Color(colorHighlight)).Padding(0, 1).Render(m.lastCommitMsg),
			"",
			tipBox,
			"",
			menuSelectedStyle.Render("❯ [y] Submit PR"),
			menuItemStyle.Render("  [n] Continue"),
			menuItemStyle.Render("  [f] Amend"),
		)
	}

	var header string
	if m.state == viewRunning {
		header = lipgloss.JoinHorizontal(lipgloss.Left, m.spinner.View(), " Running...")
	} else {
		if m.err != nil {
			header = lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)).Bold(true).Render("✖ Failed")
		} else {
			header = lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Bold(true).Render("✔ Done")
		}
	}

	content := m.output
	if content == "" && m.err != nil {
		content = m.err.Error()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		subtitleStyle.Render(m.command),
		"",
		outputWindowStyle.Render(content),
		"",
		subtitleStyle.Render("[Esc] Back"),
	)
}

func renderStackStage(m model) string {
	header := titleStyle.Render("Stack Map")

	if len(m.stackItems) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			subtitleStyle.Render("No stack found."),
		)
	}

	var lines []string
	for i, s := range m.stackItems {
		indent := strings.Repeat("  ", s.level)
		marker := "○"
		if s.current {
			marker = "●"
		}

		label := fmt.Sprintf("%s%s %s", indent, marker, s.name)

		style := menuItemStyle
		if i == m.stackCursor {
			style = menuSelectedStyle
			label = fmt.Sprintf("%s%s %s", indent, "⦿", s.name)
		}

		lines = append(lines, style.Render(label))
	}

	legend := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("● you"),
		subtitleStyle.Render("  ○ branch  "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Render("⦿ selected"),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		lipgloss.JoinVertical(lipgloss.Left, lines...),
		"",
		legend,
		"",
		subtitleStyle.Render("[↑↓] Navigate  [Enter] Checkout  [Esc] Back"),
	)
}

func renderHelpStage() string {
	speedSection := lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("SPEED (↑↓ to select, ⏎ to run)")
	workflowSection := lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("WORKFLOW")
	navSection := lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render("NAVIGATION")

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Help"),
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
		"  Tab Expand    Show/hide full file list",
		"  h   Hooks     Toggle pre-commit hooks",
		"  u   Update    Check for updates",
		"",
		subtitleStyle.Render("[Esc] Back"),
	)
}

func renderUpdateStage(m model) string {
	content := []string{titleStyle.Render("Update")}

	if m.checkingUpdate {
		content = append(content, m.spinner.View()+" Checking...")
	} else if m.latestVersion != "" && isNewerVersion(currentVersion, m.latestVersion) {
		content = append(content,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("New: "+m.latestVersion),
			"",
			menuSelectedStyle.Render("❯ [u] Update"),
			menuItemStyle.Render("  [x] Uninstall"),
		)
	} else {
		content = append(content,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("Up to date!"),
			subtitleStyle.Render(GetCurrentVersion()),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}
