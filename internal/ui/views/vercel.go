package views

import (
	"fmt"
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

	content := RenderVercelTable(width, data, true)
	return ui.BoxStyle.Width(width - 2).Render(title + "\n" + content)

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
