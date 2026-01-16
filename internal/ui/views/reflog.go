package views

import (
	"fmt"

	"github.com/Adrian95/graphite-tui/internal/git"
	"github.com/Adrian95/graphite-tui/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// ReflogViewData contains data for rendering the reflog
type ReflogViewData struct {
	Items  []git.ReflogItem
	Cursor int
}

// RenderReflog renders the reflog list
func RenderReflog(data ReflogViewData) string {
	var content string

	title := ui.TitleStyle.Render("Reflog (Undo History)")
	subtitle := ui.SubtitleStyle.Render("[Enter] Reset to state  [Esc] Back")

	if len(data.Items) == 0 {
		content = ui.BoxStyle.Render("No reflog history found.")
	} else {
		var lines []string
		start, end := getVisibleRange(data.Cursor, len(data.Items), 15)

		for i := start; i < end; i++ {
			item := data.Items[i]
			isCursor := i == data.Cursor

			cursorStr := "  "
			if isCursor {
				cursorStr = "> "
			}

			hash := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSub)).Render(item.Hash)
			date := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSub)).Render(item.Date)
			msg := item.Message
			if isCursor {
				msg = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render(msg)
				cursorStr = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Render(cursorStr)
			} else {
				msg = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorFg)).Render(msg)
			}

			// Format: > a1b2c  2 mins ago  commit: message
			line := fmt.Sprintf("%s%s  %s  %s", cursorStr, hash, date, msg)
			lines = append(lines, line)
		}
		content = lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	return ui.BoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			subtitle,
			"",
			content,
		),
	)
}

// Simple pagination helper
func getVisibleRange(cursor, total, maxVisible int) (int, int) {
	if total <= maxVisible {
		return 0, total
	}
	start := cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	return start, end
}
