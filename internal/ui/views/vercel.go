package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/Adrian95/graphite-tui/v2/internal/vercel"
	"github.com/charmbracelet/lipgloss"
)

// VercelViewData holds Vercel deployment info
type VercelViewData struct {
	Enabled     bool
	Statuses    []vercel.DeploymentStatus
	Summary     vercel.Summary
	Cursor      int
	HasError    bool
	ErrorString string
}

// RenderVercel renders the Vercel panel list
func RenderVercel(data VercelViewData, width int) string {
	title := ui.BoxTitleStyle.Render("Vercel")

	if !data.Enabled {
		return ui.BoxStyle.Width(width - 2).Render(title + "\n" + ui.SubtitleStyle.Render("Not configured"))
	}
	if data.HasError {
		return ui.BoxStyle.Width(width - 2).Render(title + "\n" + ui.SubtitleStyle.Render("Error: "+data.ErrorString))
	}

	var lines []string
	if len(data.Statuses) == 0 {
		lines = append(lines, ui.SubtitleStyle.Render("No deployments found"))
	} else {
		lines = append(lines, ui.SubtitleStyle.Render("Branch                 Target    Status     Age"))
		lines = append(lines, ui.SubtitleStyle.Render(strings.Repeat("─", 54)))
		for i, status := range data.Statuses {
			cursor := "  "
			if i == data.Cursor {
				cursor = "> "
			}

			previewState := formatState(status.Preview)
			prodState := formatState(status.Production)

			// Prefer production row if available; otherwise preview
			rowTarget := "preview"
			rowState := previewState
			age := formatAge(status.Preview)
			if status.Production != nil {
				rowTarget = "prod"
				rowState = prodState
				age = formatAge(status.Production)
			}

			line := fmt.Sprintf("%s%-20s %-8s %-10s %s", cursor, truncate(status.Branch, 20), rowTarget, rowState, age)

			if i == data.Cursor {
				line = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render(line)
			}
			lines = append(lines, line)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)

	return ui.BoxStyle.Width(width - 2).Render(content)
}

func RenderVercelSummary(summary vercel.Summary) string {
	ready := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Render(fmt.Sprintf("Preview %d ready", summary.PreviewReady))
	building := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorWarning)).Render(fmt.Sprintf("%d building", summary.PreviewBuilding))
	errorCount := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorError)).Render(fmt.Sprintf("%d failed", summary.PreviewError))
	prod := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Render(fmt.Sprintf("Prod %d ready", summary.ProdReady))

	return lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSub)).Render(fmt.Sprintf("%s • %s • %s • %s", ready, building, errorCount, prod))
}

func formatState(dep *vercel.Deployment) string {
	if dep == nil {
		return "-"
	}

	state := dep.State
	switch vercelState(state) {
	case "ready":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Render("ready")
	case "error":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorError)).Render("error")
	default:
		// building
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorWarning)).Render("building")
	}
}

func vercelState(state string) string {
	switch state {
	case "READY", "ready":
		return "ready"
	case "ERROR", "error", "canceled":
		return "error"
	default:
		return "building"
	}
}

func formatTime(ts int64) string {
	if ts == 0 {
		return ""
	}
	return time.UnixMilli(ts).Format("15:04")
}

func formatAge(dep *vercel.Deployment) string {
	if dep == nil {
		return "-"
	}
	created := dep.CreatedAt
	if created == 0 {
		created = dep.Created
	}
	if created == 0 {
		return "-"
	}
	age := time.Since(time.UnixMilli(created))
	if age.Hours() >= 24 {
		return fmt.Sprintf("%dd", int(age.Hours()/24))
	}
	if age.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(age.Hours()))
	}
	if age.Minutes() >= 1 {
		return fmt.Sprintf("%dm", int(age.Minutes()))
	}
	return "just now"
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}
