package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// --- Theme & Colors ---

const (
	colorBlack     = "#000000"
	colorWhite     = "#EDEDED"
	colorGray      = "#666666"
	colorLightGray = "#999999"
	colorDarkGray  = "#333333"
	colorBlue      = "#0070F3"
	colorPurple    = "#7928CA"
	colorRed       = "#FF4500"
	colorGreen     = "#00C64F"
	colorYellow    = "#F5A623"
	colorCyan      = "#00D4FF"
)

// --- Styles ---

var (
	// Layout
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	// Header
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWhite)).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray))

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			Italic(true)

	// Dashboard panels
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorDarkGray)).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			Bold(true)

	// Branch info
	branchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCyan)).
			Bold(true)

	aheadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorYellow))

	// File status
	fileModifiedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorYellow))

	fileAddedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGreen))

	fileDeletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorRed))

	fileUntrackedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorGray))

	// Shortcuts bar
	shortcutKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorBlue)).
				Bold(true)

	shortcutLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorGray))

	// Menu items
	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorWhite)).
				Bold(true).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color(colorBlue)).
				PaddingLeft(1)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			Italic(true).
			MarginLeft(2)

	// Guide/tip box
	guideStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorYellow)).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorDarkGray))

	copilotStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorDarkGray))

	// Input
	inputTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			Bold(true).
			MarginBottom(1)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorGray)).
			Padding(0, 1).
			Width(60)

	// Status indicators
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorRed))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorYellow))

	// Toggles
	toggleOnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	toggleOffStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGray))
)

// --- View Helpers ---

func renderHeader(version string, lastRefresh time.Time, updateAvailable string) string {
	title := headerStyle.Render("Graphite")
	subtitle := subtitleStyle.Render(" / Speedrun")
	ver := versionStyle.Render(" " + version)

	header := lipgloss.JoinHorizontal(lipgloss.Left, title, subtitle, ver)

	// Add refresh indicator
	if !lastRefresh.IsZero() {
		ago := time.Since(lastRefresh).Round(time.Second)
		refresh := subtitleStyle.Render(fmt.Sprintf("  ↻ %s ago", ago))
		header = lipgloss.JoinHorizontal(lipgloss.Left, header, refresh)
	}

	// Add update notification
	if updateAvailable != "" {
		update := warningStyle.Render(fmt.Sprintf("  ⬆ %s available", updateAvailable))
		header = lipgloss.JoinHorizontal(lipgloss.Left, header, update)
	}

	return header
}

func renderBranchInfo(branch string, stack []string, ahead int, behind int) string {
	var parts []string

	// Branch name
	branchLine := "📍 " + branchStyle.Render(branch)
	if ahead > 0 {
		branchLine += aheadStyle.Render(fmt.Sprintf(" (%d ahead)", ahead))
	}
	if behind > 0 {
		branchLine += warningStyle.Render(fmt.Sprintf(" (%d behind)", behind))
	}
	parts = append(parts, branchLine)

	// Stack visualization
	if len(stack) > 0 {
		stackStr := "📚 " + subtitleStyle.Render(strings.Join(stack, " → "))
		parts = append(parts, stackStr)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return panelStyle.Width(60).Render(content)
}

func renderChangedFiles(files []ChangedFile) string {
	if len(files) == 0 {
		return panelStyle.Width(60).Render(
			subtitleStyle.Render("No changes detected"),
		)
	}

	title := panelTitleStyle.Render(fmt.Sprintf("Changed Files (%d)", len(files)))

	var fileLines []string
	maxFiles := 8 // Limit display
	for i, f := range files {
		if i >= maxFiles {
			remaining := len(files) - maxFiles
			fileLines = append(fileLines, subtitleStyle.Render(fmt.Sprintf("  ... and %d more", remaining)))
			break
		}

		var statusStyle lipgloss.Style
		switch f.Status {
		case "M":
			statusStyle = fileModifiedStyle
		case "A":
			statusStyle = fileAddedStyle
		case "D":
			statusStyle = fileDeletedStyle
		default:
			statusStyle = fileUntrackedStyle
		}

		line := fmt.Sprintf("  %s  %s", statusStyle.Render(f.Status), f.Path)
		fileLines = append(fileLines, line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title}, fileLines...)...,
	)
	return panelStyle.Width(60).Render(content)
}

func renderShortcutsBar(skipHooks bool) string {
	return renderShortcutsBarWithInit(skipHooks, true)
}

func renderShortcutsBarWithInit(skipHooks bool, gtInitialized bool) string {
	shortcuts := []struct {
		key   string
		label string
	}{
		{"i", "Init"},
		{"s", "Start"},
		{"p", "Preview"},
		{"f", "Fix"},
		{"y", "Sync"},
		{"d", "Done"},
		{"g", "GPS"},
		{"x", "Rescue"},
		{"m", "Menu"},
		{"?", "Help"},
		{"q", "Quit"},
	}

	// If already initialized, skip showing Init
	if gtInitialized {
		shortcuts = shortcuts[1:]
	}

	var parts []string
	for _, sc := range shortcuts {
		key := shortcutKeyStyle.Render("[" + sc.key + "]")
		label := shortcutLabelStyle.Render(sc.label)
		parts = append(parts, key+" "+label)
	}

	bar := strings.Join(parts, "  ")

	// Add hooks status
	hooksStatus := toggleOffStyle.Render("hooks:on")
	if skipHooks {
		hooksStatus = toggleOnStyle.Render("hooks:skip")
	}
	bar += "  " + shortcutKeyStyle.Render("[h]") + " " + hooksStatus

	return bar
}

func renderCopilot(suggestion string, guide string) string {
	if suggestion != "" && strings.HasPrefix(suggestion, "suggestion:") {
		text := strings.TrimPrefix(suggestion, "suggestion: ")
		title := lipgloss.NewStyle().Bold(true).Render("🤖 Co-Pilot: ")
		return copilotStyle.Width(60).Render(title + text)
	}

	if guide != "" {
		title := lipgloss.NewStyle().Bold(true).Render("💡 Tip: ")
		return guideStyle.Width(60).Render(title + guide)
	}

	return ""
}

func renderMenuItem(title, desc string, selected bool) string {
	if selected {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			selectedItemStyle.Render(title),
			descStyle.Render(desc),
		)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left,
		itemStyle.Render(title),
		descStyle.Render(desc),
	)
}

func renderWizardTypes(types []commitType, selectedIdx int) string {
	var lines []string
	for i, t := range types {
		prefix := "  "
		style := itemStyle
		if i == selectedIdx {
			prefix = "> "
			style = selectedItemStyle
		}
		lines = append(lines, style.Render(fmt.Sprintf("%s%s: %s", prefix, t.label, t.desc)))
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		inputTitleStyle.Render("Step 1: Select Commit Type"),
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func renderStackMap(items []stackItem, cursor int) string {
	var lines []string
	if len(items) == 0 {
		lines = append(lines, itemStyle.Render("No stack found or empty."))
	} else {
		for i, s := range items {
			prefix := "  "
			indent := strings.Repeat("  ", s.level)
			marker := " "
			style := itemStyle

			if s.current {
				marker = "*"
			}
			if i == cursor {
				prefix = "> "
				style = selectedItemStyle
			}

			label := fmt.Sprintf("%s%s%s %s", prefix, indent, marker, s.name)
			lines = append(lines, style.Render(label))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		inputTitleStyle.Render("Stack Map (GPS)"),
		lipgloss.JoinVertical(lipgloss.Left, lines...),
		"\n"+subtitleStyle.Render("Enter to checkout • r to refresh • Esc back"),
	)
}

func renderHelp() string {
	sections := []struct {
		title string
		items []string
	}{
		{
			"Quick Actions",
			[]string{
				"[s] Start   - Create new branch & commit (wizard)",
				"[p] Preview - Push changes & open PR",
				"[f] Fix     - Amend current commit",
				"[y] Sync    - Pull latest changes",
				"[d] Done    - Merge stack & cleanup",
				"[x] Rescue  - Save work from a stuck/merged branch",
			},
		},
		{
			"Navigation",
			[]string{
				"[g] GPS     - Visual stack map",
				"[m] Menu    - Full menu view",
				"[?] Help    - This screen",
				"[q] Quit    - Exit application",
			},
		},
		{
			"Options",
			[]string{
				"[h] Toggle  - Skip git hooks (for speed)",
				"[u] Update  - Check for updates",
			},
		},
		{
			"Concepts",
			[]string{
				"Stack   - A chain of branches building on each other",
				"PR      - Pull Request, asks others to review your code",
				"Commit  - A saved snapshot of your changes",
				"Amend   - Update the last commit instead of making a new one",
			},
		},
	}

	var content []string
	content = append(content, inputTitleStyle.Render("Help & Keyboard Shortcuts"))
	content = append(content, "")

	for _, section := range sections {
		content = append(content, panelTitleStyle.Render(section.title))
		for _, item := range section.items {
			content = append(content, "  "+item)
		}
		content = append(content, "")
	}

	content = append(content, subtitleStyle.Render("Press Esc or ? to close"))

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func renderError(err error, output string, command string) string {
	title := errorStyle.Render("● Command Failed")

	var content []string
	content = append(content, title)
	content = append(content, "")

	if command != "" {
		content = append(content, subtitleStyle.Render("Command: ")+command)
	}

	if err != nil {
		content = append(content, "")
		content = append(content, errorStyle.Render("Error: ")+err.Error())
	}

	if output != "" {
		content = append(content, "")
		content = append(content, subtitleStyle.Render("Output:"))
		// Limit output lines
		lines := strings.Split(output, "\n")
		maxLines := 10
		for i, line := range lines {
			if i >= maxLines {
				content = append(content, subtitleStyle.Render(fmt.Sprintf("... (%d more lines)", len(lines)-maxLines)))
				break
			}
			content = append(content, "  "+line)
		}
	}

	content = append(content, "")
	content = append(content, subtitleStyle.Render("Press Esc to continue"))

	return panelStyle.Width(70).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}

func renderSuccess(output string, command string) string {
	title := successStyle.Render("● Success")

	var content []string
	content = append(content, title)

	if output != "" && strings.TrimSpace(output) != "" {
		content = append(content, "")
		lines := strings.Split(strings.TrimSpace(output), "\n")
		maxLines := 15
		for i, line := range lines {
			if i >= maxLines {
				content = append(content, subtitleStyle.Render(fmt.Sprintf("... (%d more lines)", len(lines)-maxLines)))
				break
			}
			content = append(content, line)
		}
	} else {
		content = append(content, "")
		content = append(content, "Command completed successfully.")
	}

	content = append(content, "")
	content = append(content, subtitleStyle.Render("Press Esc to continue"))

	return panelStyle.Width(70).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}

func renderUpdateView(currentVersion, latestVersion string, checking bool) string {
	var content []string
	content = append(content, inputTitleStyle.Render("Update Manager"))
	content = append(content, "")

	content = append(content, "Current version: "+versionStyle.Render(currentVersion))

	if checking {
		content = append(content, "")
		content = append(content, subtitleStyle.Render("Checking for updates..."))
	} else if latestVersion != "" && latestVersion != currentVersion {
		content = append(content, "Latest version:  "+successStyle.Render(latestVersion))
		content = append(content, "")
		content = append(content, warningStyle.Render("Update available!"))
		content = append(content, "")
		content = append(content, "[u] Update now")
		content = append(content, "[x] Uninstall")
		content = append(content, "[Esc] Back")
	} else {
		content = append(content, "")
		content = append(content, successStyle.Render("You're up to date!"))
		content = append(content, "")
		content = append(content, "[x] Uninstall")
		content = append(content, "[Esc] Back")
	}

	return panelStyle.Width(50).Render(lipgloss.JoinVertical(lipgloss.Left, content...))
}
