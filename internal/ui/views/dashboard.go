package views

import (
	"fmt"
	"strings"

	"github.com/Adrian95/graphite-tui/v2/internal/git"
	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// DashboardViewData contains all data needed to render the dashboard
type DashboardViewData struct {
	// Branch info
	Branch string
	Ahead  int
	Behind int

	// Files
	ChangedFiles  []git.ChangedFile
	FilesExpanded bool
	FileList      list.Model

	// Metrics
	LinesAdded   int
	LinesRemoved int
	NewFiles     int
	ModFiles     int
	DelFiles     int
	StagedCount  int

	// State
	GtInitialized bool
	OnMain        bool
	SpeedCursor   int

	// Options
	SkipHooks       bool
	UpdateAvailable string
	FlashMessage    string
	FileBoxFocused  bool
	VercelFocused   bool
	StackFocused    bool

	// Vercel
	VercelSummary VercelViewData

	// Stack
	StackData StackViewData

	// Version
	CurrentVersion string
}

// RenderDashboard renders the complete dashboard view
func RenderDashboard(ctx RenderContext, data DashboardViewData) string {
	sidebar := RenderSidebar(ctx, SidebarData{
		CurrentVersion:  data.CurrentVersion,
		SkipHooks:       data.SkipHooks,
		UpdateAvailable: data.UpdateAvailable,
		FlashMessage:    data.FlashMessage,
	})

	main := RenderDashboardMain(ctx, data)
	footer := "" // Actions moved to sidebar

	return RenderLayout(sidebar, main, footer)
}

// RenderDashboardMain renders the main content area
func RenderDashboardMain(ctx RenderContext, data DashboardViewData) string {
	width := ctx.MainWidth()

	// Minimal top status line
	branchBox := renderBranchBox(width, data.Branch, data.Ahead, data.Behind)
	metrics := renderMetrics(data)

	mainPanels := RenderDashboardPanels(ctx, data)

	tip := getContextualTip(data)
	tipBox := ui.CardStyle.Width(width - 2).Render(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render("→ ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorFg)).Render(tip),
	)

	parts := []string{branchBox}
	if metrics != "" {
		parts = append(parts, "", metrics)
	}
	parts = append(parts, "", mainPanels, "", tipBox)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func renderVercelSummaryBox(width int, data VercelViewData, focused bool) string {
	title := "Vercel"
	if focused {
		title = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Render(title)
	} else {
		title = ui.BoxTitleStyle.Render(title)
	}

	content := ui.SubtitleStyle.Render("Not configured")
	if data.Enabled {
		content = RenderVercelSummary(data.Summary)
		if data.HasError {
			content = ui.SubtitleStyle.Render("Error: " + data.ErrorString)
		}
	}

	return ui.BoxStyle.Width(width - 2).Render(title + "\n" + content)
}

func renderMetrics(data DashboardViewData) string {
	if len(data.ChangedFiles) == 0 {
		return ""
	}

	added := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Render(fmt.Sprintf("+%d", data.LinesAdded))
	removed := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorError)).Render(fmt.Sprintf("-%d", data.LinesRemoved))

	loc := fmt.Sprintf("%s / %s lines", added, removed)

	files := fmt.Sprintf("%d New • %d Mod • %d Del", data.NewFiles, data.ModFiles, data.DelFiles)
	filesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSub)).Render(files)

	staged := ""
	if data.StagedCount > 0 {
		staged = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Bold(true).Render(fmt.Sprintf(" • %d Staged", data.StagedCount))
	} else {
		staged = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorWarning)).Render(" • All Unstaged")
	}

	return ui.BoxStyle.Render(loc + "  " + filesStyle + staged)
}

func renderBranchBox(width int, branch string, ahead, behind int) string {
	if branch == "" {
		branch = "loading..."
	}

	branchIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Render("")
	branchName := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorFg)).Bold(true).Render(branch)

	var badges []string
	if ahead > 0 {
		badges = append(badges, ui.StatusAheadStyle.Render(fmt.Sprintf("↑%d", ahead)))
	}
	if behind > 0 {
		badges = append(badges, ui.StatusBehindStyle.Render(fmt.Sprintf("↓%d", behind)))
	}

	branchLine := branchIcon + " " + branchName
	if len(badges) > 0 {
		branchLine += "  " + strings.Join(badges, " ")
	}

	return ui.BoxStyle.Width(width - 2).Render(branchLine)
}

func renderFilesBox(width int, files []git.ChangedFile, expanded bool, fileList list.Model, focused bool) string {
	var filesContent string

	if len(files) > 0 {
		// Safety check for width
		safeWidth := width - 4
		if safeWidth < 1 {
			safeWidth = 20 // Fallback minimum
		}

		fileList.SetWidth(safeWidth)
		fileList.SetHeight(12)

		filesContent = fileList.View()
	} else {
		// Use manual title for empty state to match look
		title := ui.BoxTitleStyle.Render("Changes")
		if focused {
			title = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render("Changes")
		}
		content := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Render("✓ Clean")
		filesContent = title + "\n" + content
	}

	return ui.BoxStyle.Width(width - 2).Render(filesContent)
}

func getContextualTip(data DashboardViewData) string {
	if data.FileBoxFocused {
		return "[↑↓] Navigate  [Space] Stage/Unstage  [a] Stage All  [u] Unstage All"
	}
	if data.VercelFocused {
		if !data.VercelSummary.Enabled {
			return "Set VERCEL_TOKEN + VERCEL_PROJECT_ID (+ VERCEL_ORG_ID if org) in .env.local/.env (token: vercel.com/account/tokens, project: Vercel → Project → Settings)"
		}
		return "[↑↓] Navigate  [⏎] Open Preview  [p] Open Prod  [y] Copy Preview"
	}

	if data.StackFocused {
		return "[↑↓] Navigate  [⏎] Checkout"
	}

	if !data.GtInitialized {
		return "Press [i] to initialize Graphite"
	}

	hasChanges := len(data.ChangedFiles) > 0

	switch data.SpeedCursor {
	case 0: // Ship
		if hasChanges {
			if data.OnMain {
				return "Creates branch → commits → submits PR"
			}
			return "Amends commit → pushes → updates PR"
		} else if data.Ahead > 0 {
			return "Submits PR for review"
		}
		return "Nothing to ship"

	case 1: // Iterate
		if hasChanges {
			return "Adds changes to last commit → pushes"
		}
		return "Nothing to amend"

	case 2: // Reset
		if hasChanges {
			return "Discards ALL uncommitted changes"
		}
		return "Working tree is clean"

	case 3: // Undo
		return "Reverses last Graphite operation"
	}

	return "[↑↓] navigate  [⏎] execute"
}
