package views

import (
	"fmt"

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

