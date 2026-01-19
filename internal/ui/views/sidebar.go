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
	width := ctx.SidebarWidth()

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

	return lipgloss.NewStyle().Width(width).PaddingRight(2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			appTitle,
			version,
			"",
			hooksLine,
			updateLine,
			flashLine,
		),
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
