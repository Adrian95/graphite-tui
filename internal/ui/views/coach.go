package views

import (
	"fmt"
	"strings"

	"github.com/Adrian95/graphite-tui/v2/internal/coach"
	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// RenderExplainer renders an educational explainer screen after user actions
func RenderExplainer(data coach.ExplainerData) string {
	var sections []string

	// Title with emoji
	sections = append(sections,
		ui.TitleStyle.Render(data.Title),
		"",
	)

	// What Happened (in box)
	if data.WhatHappened != "" {
		sections = append(sections,
			ui.SubtitleStyle.Render("What Just Happened"),
			ui.CardStyle.Render(data.WhatHappened),
			"",
		)
	}

	// Why It Matters
	if data.WhyItMatters != "" {
		sections = append(sections,
			ui.SubtitleStyle.Render("Why This Matters"),
			ui.CardStyle.Render(data.WhyItMatters),
			"",
		)
	}

	// Repo State Visualization
	if data.RepoState != "" {
		sections = append(sections,
			ui.SubtitleStyle.Render("Your Stack Now"),
			data.RepoState,
			"",
		)
	}

	// Next Steps (bulleted list)
	if len(data.NextSteps) > 0 {
		sections = append(sections,
			ui.SubtitleStyle.Render("What's Next?"),
		)

		accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent))

		for i, step := range data.NextSteps {
			bullet := accentStyle.Render(fmt.Sprintf("%d.", i+1))
			sections = append(sections, fmt.Sprintf("  %s %s", bullet, step))
		}
		sections = append(sections, "")
	}

	// Footer
	var controlParts []string
	controlParts = append(controlParts, "[Enter] Continue")

	if data.CanSkip {
		controlParts = append(controlParts, "[d] Don't show this again")
	}

	controlParts = append(controlParts, "[c] Disable coach mode")

	controls := strings.Join(controlParts, "    ")
	sections = append(sections, ui.SubtitleStyle.Render(controls))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// RenderInlineTip renders smaller tips for the dashboard (future enhancement)
func RenderInlineTip(tip string) string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ui.ColorAccent)).
		Padding(0, 1).
		Render("💡 Coach: " + tip)
}
