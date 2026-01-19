package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// --- Theme & Aesthetics (Vercel Black + Hot Pink) ---

const (
	// Palette - Minimalist dark with hot pink accent
	ColorBg        = "#000000" // Pure Black (OLED/Vercel)
	ColorFg        = "#EDEDED" // Off-white text
	ColorSub       = "#666666" // Dark Gray for subtitles
	ColorBorder    = "#333333" // Subtle borders
	ColorAccent    = "#FF0080" // Hot Pink (Primary accent)
	ColorSuccess   = "#50E3C2" // Teal/Green (Success/Added)
	ColorWarning   = "#F5A623" // Orange (Modified/Warning)
	ColorError     = "#FF3366" // Bright red-pink for errors
	ColorHighlight = "#1A1A1A" // Dark highlight for active items
)

// --- Styles ---

var (
	// Layout
	DocStyle = lipgloss.NewStyle().
			Margin(1, 2).
			Foreground(lipgloss.Color(ColorFg))

	// Typography
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorFg)).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSub))

	// Box styles (borderless for transparent terminal support)
	BoxStyle = lipgloss.NewStyle().
			PaddingLeft(1)

	BoxTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSub)).
			Bold(true)

	// Bordered box styles for panels
	BorderedBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorAccent)).
				PaddingLeft(1).
				PaddingRight(1)

	// Navigation / Menu
	MenuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSub))

	MenuSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorAccent)).
				Bold(true)

	// Branch / Status Badges
	StatusAheadStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorBg)).
				Background(lipgloss.Color(ColorSuccess)).
				Bold(true).
				Padding(0, 1)

	StatusBehindStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorBg)).
				Background(lipgloss.Color(ColorError)).
				Bold(true).
				Padding(0, 1)

	// Files
	FileAddedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess))
	FileModifiedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))
	FileDeletedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError))
	FileUntrackedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSub))

	// Wizard / Input
	WizardQuestionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorFg)).
				Bold(true).
				PaddingBottom(1)

	InputBoxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorFg)).
			Width(50)

	// Output Window
	OutputWindowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorFg)).
				PaddingLeft(1)

	// Cards (Tips)
	CardStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorFg)).
			PaddingLeft(1)

	// Highlight box (transparent-friendly)
	HighlightBoxStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorFg)).
				PaddingLeft(1)

	// Footer bar
	FooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSub)).
			PaddingTop(1)

	FooterKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent)).
			Bold(true)

	FooterLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorSub))
)
