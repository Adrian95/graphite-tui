package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/Adrian95/graphite-tui/v2/internal/vercel"
	"github.com/charmbracelet/lipgloss"
)

// RenderDashboardPanels renders the main panel layout
func RenderDashboardPanels(ctx RenderContext, data DashboardViewData) string {
	mode := ctx.GetLayoutMode()

	switch mode {
	case LayoutGrid2x2:
		return renderGridLayout(ctx, data)
	case LayoutStack1x4:
		return renderStackLayout(ctx, data)
	case LayoutSingle:
		return renderSingleLayout(ctx, data)
	}

	// Fallback to grid
	return renderGridLayout(ctx, data)
}

func renderGridLayout(ctx RenderContext, data DashboardViewData) string {
	panelWidth := ctx.MainSplitWidth()
	gap := "  "

	// Panels with focus states
	speedFocused := !data.FileBoxFocused && !data.VercelFocused && !data.StackFocused
	speedPanel := renderSpeedPanel(panelWidth, data, speedFocused)
	filesPanel := renderFilesPanel(panelWidth, data, data.FileBoxFocused)
	vercelPanel := renderVercelPanel(panelWidth, data, data.VercelFocused)
	stackPanel := RenderStackPanel(panelWidth, data.StackData, data.StackFocused)

	// Layout rows
	leftColumn := lipgloss.JoinVertical(lipgloss.Top,
		speedPanel,
		"",
		filesPanel,
	)
	rightColumn := lipgloss.JoinVertical(lipgloss.Top,
		vercelPanel,
		"",
		stackPanel,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, gap, rightColumn)
}

func renderStackLayout(ctx RenderContext, data DashboardViewData) string {
	fullWidth := ctx.MainWidth() - 2 // Account for borders

	// Panels with focus states
	speedFocused := !data.FileBoxFocused && !data.VercelFocused && !data.StackFocused
	speedPanel := renderSpeedPanel(fullWidth, data, speedFocused)
	filesPanel := renderFilesPanel(fullWidth, data, data.FileBoxFocused)
	vercelPanel := renderVercelPanel(fullWidth, data, data.VercelFocused)
	stackPanel := RenderStackPanel(fullWidth, data.StackData, data.StackFocused)

	// Stack vertically
	return lipgloss.JoinVertical(lipgloss.Left,
		speedPanel,
		"",
		filesPanel,
		"",
		vercelPanel,
		"",
		stackPanel,
	)
}

func renderSingleLayout(ctx RenderContext, data DashboardViewData) string {
	fullWidth := ctx.MainWidth() - 2 // Account for borders

	// Show only the currently focused panel
	if data.FileBoxFocused {
		return renderFilesPanel(fullWidth, data, true)
	} else if data.VercelFocused {
		return renderVercelPanel(fullWidth, data, true)
	} else if data.StackFocused {
		return RenderStackPanel(fullWidth, data.StackData, true)
	} else {
		// Default to speed panel
		return renderSpeedPanel(fullWidth, data, true)
	}
}

func renderSpeedPanel(width int, data DashboardViewData, isFocused bool) string {
	focused := !data.FileBoxFocused && !data.VercelFocused && !data.StackFocused
	title := renderPanelTitle("Speed", focused)

	actions := []string{}
	items := []struct {
		label string
	}{
		{"Ship"},
		{"Iterate"},
		{"Reset"},
		{"Undo"},
	}
	for i, item := range items {
		marker := ui.MenuItemStyle.Render("□")
		labelStyle := ui.MenuItemStyle
		if i == data.SpeedCursor {
			marker = ui.MenuSelectedStyle.Render("■")
			labelStyle = ui.MenuSelectedStyle
		}
		actions = append(actions, marker+" "+labelStyle.Render(item.label))
	}

	content := strings.Join(actions, "\n")

	// Use focused or unfocused styling
	style := ui.BorderedBoxStyleUnfocused
	if isFocused {
		style = ui.BorderedBoxStyleFocused
	}
	return style.Width(width - 2).Render(title + "\n" + content)
}

func renderFilesPanel(width int, data DashboardViewData, isFocused bool) string {
	title := renderPanelTitle("Changes", data.FileBoxFocused)

	var content string
	if !data.StatusLoaded {
		// Show loading state before first status fetch completes
		content = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSub)).Render("Loading...")
	} else if len(data.ChangedFiles) == 0 {
		content = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Render("✓ Clean")
	} else {
		fileList := data.FileList
		safeWidth := width - 4
		if safeWidth < 1 {
			safeWidth = 20
		}
		fileList.SetWidth(safeWidth)
		fileList.SetHeight(10)
		content = fileList.View()
	}

	// Use focused or unfocused styling
	style := ui.BorderedBoxStyleUnfocused
	if isFocused {
		style = ui.BorderedBoxStyleFocused
	}
	return style.Width(width - 2).Render(title + "\n" + content)
}

func renderVercelPanel(width int, data DashboardViewData, isFocused bool) string {
	title := renderPanelTitle("Vercel", data.VercelFocused)

	if !data.VercelSummary.Enabled {
		content := ui.SubtitleStyle.Render("Not configured")
		style := ui.BorderedBoxStyleUnfocused
		if isFocused {
			style = ui.BorderedBoxStyleFocused
		}
		return style.Width(width - 2).Render(title + "\n" + content)
	}

	table := RenderVercelTable(width, data.VercelSummary, data.VercelFocused)
	style := ui.BorderedBoxStyleUnfocused
	if isFocused {
		style = ui.BorderedBoxStyleFocused
	}
	return style.Width(width - 2).Render(title + "\n" + table)
}

func renderPanelTitle(label string, focused bool) string {
	if focused {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render(label)
	}
	return ui.BoxTitleStyle.Render(label)
}

// RenderVercelTable renders the Vercel deployments table
func RenderVercelTable(width int, data VercelViewData, focused bool) string {
	if len(data.Statuses) == 0 {
		return ui.SubtitleStyle.Render("No deployments found")
	}

	headers := ui.SubtitleStyle.Render("Branch               Target   Status     Age")
	divider := ui.SubtitleStyle.Render(strings.Repeat("─", 48))

	var rows []string
	for i, status := range data.Statuses {
		cursor := "  "
		if i == data.Cursor && focused {
			cursor = "❯ "
		}
		rowTarget := "preview"
		rowState := formatState(status.Preview)
		age := formatAge(status.Preview)
		if status.Production != nil {
			rowTarget = "prod"
			rowState = formatState(status.Production)
			age = formatAge(status.Production)
		}
		row := fmt.Sprintf("%s%-18s %-7s %-9s %s", cursor, truncate(status.Branch, 18), rowTarget, rowState, age)
		if i == data.Cursor && focused {
			row = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render(row)
		}
		rows = append(rows, row)
	}

	return strings.Join(append([]string{headers, divider}, rows...), "\n")
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
