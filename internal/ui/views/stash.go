package views

import (
	"fmt"

	"github.com/Adrian95/graphite-tui/internal/git"
	"github.com/Adrian95/graphite-tui/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// StashViewData contains data for rendering the stash list
type StashViewData struct {
	Items  []git.StashItem
	Cursor int
}

// RenderStash renders the stash manager
func RenderStash(data StashViewData) string {
	var content string

	title := ui.TitleStyle.Render("Stash Manager")
	subtitle := ui.SubtitleStyle.Render("[Enter] Pop  [a] Apply  [d] Drop  [c] Create  [Esc] Back")

	if len(data.Items) == 0 {
		content = ui.BoxStyle.Render("No stashed changes found.")
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

			id := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSub)).Render(item.ID)
			date := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSub)).Render(item.Date)
			branch := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Render(item.Branch)
			msg := item.Message

			if isCursor {
				msg = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render(msg)
				cursorStr = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Render(cursorStr)
			} else {
				msg = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorFg)).Render(msg)
			}

			// Format: > stash@{0}  2 mins ago  [main]  message
			line := fmt.Sprintf("%s%s  %s  [%s]  %s", cursorStr, id, date, branch, msg)
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
