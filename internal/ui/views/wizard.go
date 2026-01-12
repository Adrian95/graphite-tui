package views

import (
	"fmt"

	"github.com/Adrian95/graphite-tui/internal/git"
	"github.com/Adrian95/graphite-tui/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// WizardViewData contains wizard state data
type WizardViewData struct {
	Stage         int // 1=Type, 2=Scope, 3=Summary, 4=Preview
	CommitTypes   []git.CommitType
	TypeIdx       int
	Scope         string
	Summary       string
	ErrorMsg      string
	CommitMessage string
	InputValue    string
	FileCount     int
}

// RenderWizardType renders the commit type selection (Step 1)
func RenderWizardType(data WizardViewData) string {
	var types []string
	for i, t := range data.CommitTypes {
		prefix := "  "
		style := ui.MenuItemStyle
		if i == data.TypeIdx {
			prefix = "❯ "
			style = ui.MenuSelectedStyle
		}
		exampleText := ui.SubtitleStyle.Render(fmt.Sprintf(" \"%s\"", t.Example))
		types = append(types, style.Render(fmt.Sprintf("%s%-10s %-16s", prefix, t.Label, t.Desc))+exampleText)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		ui.SubtitleStyle.Render("Step 1 of 4"),
		ui.WizardQuestionStyle.Render("What kind of change is this?"),
		"",
		lipgloss.JoinVertical(lipgloss.Left, types...),
		"",
		ui.SubtitleStyle.Render("[↑↓] Select  [Enter] Confirm  [Esc] Cancel"),
	)
}

// RenderWizardScope renders the scope input (Step 2)
func RenderWizardScope(data WizardViewData, inputView string) string {
	return renderWizardInput(
		"Step 2 of 4",
		"What part of the codebase? (Optional)",
		"Component name: auth, navbar, api, etc.",
		inputView,
		"",
		"[Enter] Next  [Backspace] Back  [Esc] Cancel",
	)
}

// RenderWizardSummary renders the summary input (Step 3)
func RenderWizardSummary(data WizardViewData, inputView string) string {
	charCount := ui.SubtitleStyle.Render(fmt.Sprintf("[%d/72 characters]", len(data.InputValue)))

	content := renderWizardInput(
		"Step 3 of 4",
		"Describe the change",
		"Use present tense: 'add button' not 'added button'",
		inputView,
		data.ErrorMsg,
		"[Enter] Next  [Backspace] Back  [Esc] Cancel",
	)

	// Insert char count after input
	lines := []string{content, charCount}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// RenderWizardPreview renders the commit preview (Step 4)
func RenderWizardPreview(data WizardViewData) string {
	fileText := fmt.Sprintf("%d file", data.FileCount)
	if data.FileCount != 1 {
		fileText += "s"
	}

	commitBox := ui.HighlightBoxStyle.Render(data.CommitMessage)

	tipBox := ui.CardStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent)).Bold(true).Render("This will:"),
			"  • Create a new branch",
			fmt.Sprintf("  • Commit your %s", fileText),
			"  • You can share for review with [p]",
		),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		ui.SubtitleStyle.Render("Step 4 of 4"),
		ui.WizardQuestionStyle.Render("Ready to commit?"),
		"",
		commitBox,
		"",
		tipBox,
		"",
		ui.SubtitleStyle.Render("[Enter] Commit  [Esc] Edit  [Backspace] Back"),
	)
}

func renderWizardInput(step, question, placeholder, inputView, errorMsg, hint string) string {
	inputDisplay := ui.InputBoxStyle.Render(inputView)

	if errorMsg != "" {
		inputDisplay = lipgloss.JoinVertical(lipgloss.Left,
			inputDisplay,
			lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorError)).Render("⚠ "+errorMsg),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		ui.SubtitleStyle.Render(step),
		ui.WizardQuestionStyle.Render(question),
		ui.SubtitleStyle.Render(placeholder),
		"",
		inputDisplay,
		"",
		ui.SubtitleStyle.Render(hint),
	)
}
