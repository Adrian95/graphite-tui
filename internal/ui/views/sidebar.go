package views

import (
	"strings"

	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// SidebarData contains data needed to render the sidebar
type SidebarData struct {
	CurrentVersion  string
	SkipHooks       bool
	UpdateAvailable string
	FlashMessage    string
}

// RenderSidebar renders the left sidebar
func RenderSidebar(ctx RenderContext, data SidebarData) string {
	width := ctx.GetSidebarWidth()

	// App title
	appTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.ColorAccent)).
		Bold(true).
		Render("GRAPHITE TUI")

	version := ui.SubtitleStyle.Render(data.CurrentVersion)

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

	// Quick actions (moved from footer)
	actionsTitle := ui.SubtitleStyle.Bold(true).Render("ACTIONS")
	actions := renderQuickActions()

	return lipgloss.NewStyle().Width(width).PaddingRight(2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			appTitle,
			version,
			"",
			hooksLine,
			updateLine,
			flashLine,
			"",
			actionsTitle,
			"",
			actions,
		),
	)
}

// renderQuickActions renders the quick action shortcuts
func renderQuickActions() string {
	shortcuts := []struct {
		key   string
		label string
	}{
		{"s", "Ship"},
		{"y", "Sync"},
		{"d", "Done"},
		{"?", "Help"},
	}

	var parts []string
	for _, s := range shortcuts {
		parts = append(parts, ui.FooterKeyStyle.Render(s.key)+" "+ui.FooterLabelStyle.Render(s.label))
	}

	return strings.Join(parts, "\n")
}
