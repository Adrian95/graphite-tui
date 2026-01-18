package views

import (
	"strings"

	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// SidebarData contains data needed to render the sidebar
type SidebarData struct {
	CurrentVersion  string
	SpeedCursor     int
	SkipHooks       bool
	UpdateAvailable string
	FlashMessage    string
}

// RenderSidebar renders the left sidebar
func RenderSidebar(ctx RenderContext, data SidebarData) string {
	width := ctx.SidebarWidth()

	// App title
	appTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorAccent)).
		Bold(true).
		Render("GRAPHITE TUI")

	version := ui.SubtitleStyle.Render(data.CurrentVersion)

	// Speed box
	speedBox := renderSpeedBox(width, data.SpeedCursor)

	// Hooks state
	hooksState := "ON"
	hooksColor := ui.ColorSuccess
	if data.SkipHooks {
		hooksState = "OFF"
		hooksColor = ui.ColorWarning
	}
	hooksLine := ui.SubtitleStyle.Render("[h] Hooks ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color(hooksColor)).Render(hooksState)

	// Update indicator
	updateLine := ""
	if data.UpdateAvailable != "" {
		updateLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color(ui.ColorSuccess)).
			Render("↑ "+data.UpdateAvailable+" [u]")
	}

	// Flash message
	flashLine := ""
	if data.FlashMessage != "" {
		flashLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color(ui.ColorAccent)).
			Italic(true).
			Render(data.FlashMessage)
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

func renderSpeedBox(width int, cursor int) string {
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
		if i == cursor {
			line = ui.MenuSelectedStyle.Render("❯ "+item.key) + " " + ui.MenuSelectedStyle.Render(item.label)
		} else {
			line = ui.MenuItemStyle.Render("  "+item.key) + " " + ui.MenuItemStyle.Render(item.label)
		}
		speedActions = append(speedActions, line)
	}

	speedContent := lipgloss.JoinVertical(lipgloss.Left, speedActions...)
	return ui.BoxStyle.Width(width - 2).Render(
		ui.BoxTitleStyle.Render("SPEED") + "\n" + speedContent,
	)
}

// RenderFooter renders the bottom footer with shortcuts
func RenderFooter(ctx RenderContext) string {
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
		parts = append(parts, ui.FooterKeyStyle.Render(s.key)+" "+ui.FooterLabelStyle.Render(s.label))
	}

	line := strings.Repeat("─", ctx.Width-4)
	content := strings.Join(parts, "   ")
	return ui.FooterStyle.Render(ui.SubtitleStyle.Render(line) + "\n" + content)
}
