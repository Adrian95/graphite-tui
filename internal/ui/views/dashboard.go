package views

import (
	"fmt"
	"strings"

	"github.com/Adrian95/graphite-tui/internal/git"
	"github.com/Adrian95/graphite-tui/internal/ui"
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

	// State
	GtInitialized bool
	OnMain        bool
	SpeedCursor   int

	// Options
	SkipHooks       bool
	UpdateAvailable string
	FlashMessage    string

	// Version
	CurrentVersion string
}

// RenderDashboard renders the complete dashboard view
func RenderDashboard(ctx RenderContext, data DashboardViewData) string {
	sidebar := RenderSidebar(ctx, SidebarData{
		CurrentVersion:  data.CurrentVersion,
		SpeedCursor:     data.SpeedCursor,
		SkipHooks:       data.SkipHooks,
		UpdateAvailable: data.UpdateAvailable,
		FlashMessage:    data.FlashMessage,
	})

	main := RenderDashboardMain(ctx, data)
	footer := RenderFooter(ctx)

	return RenderLayout(sidebar, main, footer)
}

// RenderDashboardMain renders the main content area
func RenderDashboardMain(ctx RenderContext, data DashboardViewData) string {
	width := ctx.MainWidth()

	// Branch box
	branchBox := renderBranchBox(width, data.Branch, data.Ahead, data.Behind)

	// Files box
	filesBox := renderFilesBox(width, data.ChangedFiles, data.FilesExpanded)

	// Tip box
	tip := getContextualTip(data)
	tipBox := ui.CardStyle.Width(width - 2).Render(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render("→ ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorFg)).Render(tip),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		branchBox,
		"",
		filesBox,
		"",
		tipBox,
	)
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

func renderFilesBox(width int, files []git.ChangedFile, expanded bool) string {
	filesTitle := ui.BoxTitleStyle.Render("CHANGES")
	var filesContent string

	if len(files) > 0 {
		expandHint := ui.SubtitleStyle.Render(fmt.Sprintf(" (%d) [Tab]", len(files)))
		filesTitle = ui.BoxTitleStyle.Render("CHANGES") + expandHint

		var fileLines []string
		maxFiles := 6
		if expanded {
			maxFiles = 30
		}

		for i, f := range files {
			if i >= maxFiles {
				remaining := len(files) - maxFiles
				fileLines = append(fileLines, ui.SubtitleStyle.Render(fmt.Sprintf("...%d more", remaining)))
				break
			}

			path := f.Path
			maxLen := width - 10
			if maxLen < 20 {
				maxLen = 20
			}
			if len(path) > maxLen {
				path = "..." + path[len(path)-maxLen+3:]
			}

			var styledLine string
			switch f.Status {
			case "M", "MM":
				styledLine = ui.FileModifiedStyle.Render("M " + path)
			case "A", "AM":
				styledLine = ui.FileAddedStyle.Render("+ " + path)
			case "D":
				styledLine = ui.FileDeletedStyle.Render("- " + path)
			case "R":
				styledLine = ui.FileModifiedStyle.Render("R " + path)
			case "??":
				styledLine = ui.FileUntrackedStyle.Render("? " + path)
			default:
				styledLine = ui.FileUntrackedStyle.Render("• " + path)
			}
			fileLines = append(fileLines, styledLine)
		}
		filesContent = lipgloss.JoinVertical(lipgloss.Left, fileLines...)
	} else {
		filesContent = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorSuccess)).Render("✓ Clean")
	}

	return ui.BoxStyle.Width(width - 2).Render(filesTitle + "\n" + filesContent)
}

func getContextualTip(data DashboardViewData) string {
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
