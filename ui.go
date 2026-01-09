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
	case viewInput:
		mainStage = renderInputStage(m)
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
		subtitleStyle.Render(currentVersion),
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
	if m.skipHooks {
		hooksState = "OFF"
	}
	settings := lipgloss.JoinVertical(lipgloss.Left,
		subtitleStyle.Render("────"),
		subtitleStyle.Render(fmt.Sprintf("[h] Hooks: %s", hooksState)),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		branchInfo,
		"\n",
		lipgloss.JoinVertical(lipgloss.Left, menuItems...),
		"\n\n",
		settings,
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
	// 1. Context Header
	contextTitle := "Project Overview"
	if len(m.changedFiles) > 0 {
		contextTitle = "Unsaved Changes"
	}
	header := titleStyle.Render(contextTitle)

	// 2. File List or Empty State
	var content string
	if len(m.changedFiles) > 0 {
		var files []string
		for _, f := range m.changedFiles {
			var icon, style string
			// Nerd Font icons would go here, using ASCII/Unicode for now
			switch f.Status {
			case "M":
				icon = "● "
				style = fileModifiedStyle.Render(icon + f.Path)
			case "A":
				icon = "+ "
				style = fileAddedStyle.Render(icon + f.Path)
			case "D":
				icon = "✖ "
				style = fileDeletedStyle.Render(icon + f.Path)
			default:
				icon = "? "
				style = fileUntrackedStyle.Render(icon + f.Path)
			}
			files = append(files, style)
		}
		// Limit to 10 files to avoid overflow
		if len(files) > 12 {
			remaining := len(files) - 12
			files = files[:12]
			files = append(files, subtitleStyle.Render(fmt.Sprintf("...and %d more", remaining)))
		}
		content = lipgloss.JoinVertical(lipgloss.Left, files...)
	} else {
		// "All Clear" State
		content = lipgloss.JoinVertical(lipgloss.Center,
			"\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSub)).Render("No changed files"),
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Render("Everything is clean ✨"),
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
		step = "Step 1 of 3"
		question = "What kind of change is this?"
		// Render types as a clean list
		var types []string
		for i, t := range m.commitTypes {
			prefix := "  "
			style := menuItemStyle
			if i == m.wizardTypeIdx {
				prefix = "❯ "
				style = menuSelectedStyle
			}
			types = append(types, style.Render(fmt.Sprintf("%s%-10s %s", prefix, t.label, t.desc)))
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			subtitleStyle.Render(step),
			wizardQuestionStyle.Render(question),
			"\n",
			lipgloss.JoinVertical(lipgloss.Left, types...),
		)

	case viewWizardScope:
		step = "Step 2 of 3"
		question = "What is the scope? (Optional)"
		placeholder = "e.g. auth, api, ui"

	case viewWizardSummary:
		step = "Step 3 of 3"
		question = "Describe the change"
		placeholder = "e.g. add new login button"
	}

	// Input view
	inputDisplay := inputBoxStyle.Render(m.textInput.View())
	if m.wizardError != "" {
		inputDisplay = lipgloss.JoinVertical(lipgloss.Left,
			inputDisplay,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)).Render("⚠ "+m.wizardError),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		subtitleStyle.Render(step),
		wizardQuestionStyle.Render(question),
		subtitleStyle.Render(placeholder),
		"\n",
		inputDisplay,
		"\n\n",
		subtitleStyle.Render("Enter to confirm • Esc to back"),
	)
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
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Bold(true).Render("✔ Changes Saved"),
			"\n",
			lipgloss.NewStyle().Background(lipgloss.Color(colorHighlight)).Padding(0, 1).Render(m.lastCommitMsg),
			"\n",
			titleStyle.Render("What's next?"),
			menuSelectedStyle.Render("❯ [p] Share & Create PR"),
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

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
		"\n",
		subtitleStyle.Render("Enter to checkout • r to refresh"),
	)
}

func renderHelpStage() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Help & Shortcuts"),
		"\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Render("General"),
		"  i  Initialize Graphite",
		"  s  Start (Commit)",
		"  p  Share (Submit)",
		"  f  Fix (Amend)",
		"  y  Sync",
		"  d  Done (Merge)",
		"  g  Stack Map",
		"\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Render("Navigation"),
		"  ↑/↓  Move selection",
		"  Enter Select",
		"  Esc  Back/Cancel",
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
			subtitleStyle.Render(currentVersion),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}
