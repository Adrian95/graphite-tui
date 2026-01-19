package views

import (
	"fmt"
	"strings"

	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// RenderStackPanel renders the stack as a dashboard panel
func RenderStackPanel(width int, data StackViewData, isPanelFocused bool) string {
	title := ui.BoxTitleStyle.Render("Stack")
	if isPanelFocused {
		title = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render("Stack")
	}

	if len(data.Items) == 0 {
		content := ui.SubtitleStyle.Render("No stack")
		style := ui.BorderedBoxStyleUnfocused
		if isPanelFocused {
			style = ui.BorderedBoxStyleFocused
		}
		return style.Width(width - 2).Render(title + "\n" + content)
	}

	var lines []string
	max := 8
	if len(data.Items) < max {
		max = len(data.Items)
	}

	for i := 0; i < max; i++ {
		item := data.Items[i]
		indent := strings.Repeat(" ", item.Level*2)
		cursor := "  "
		if i == data.Cursor && isPanelFocused {
			cursor = ui.MenuSelectedStyle.Render("❯ ")
		}
		name := item.Name
		if item.Current {
			name = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render(name)
		}
		lines = append(lines, fmt.Sprintf("%s%s%s", cursor, indent, name))
	}

	style := ui.BorderedBoxStyleUnfocused
	if isPanelFocused {
		style = ui.BorderedBoxStyleFocused
	}
	return style.Width(width - 2).Render(title + "\n" + strings.Join(lines, "\n"))
}
