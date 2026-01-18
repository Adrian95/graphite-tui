package ui

import (
	"time"

	"github.com/Adrian95/graphite-tui/v2/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Ticker for periodic refresh ---

// TickMsg is sent periodically to trigger git refreshes
type TickMsg time.Time

// VercelTickMsg is sent periodically to trigger Vercel refreshes
type VercelTickMsg time.Time

// TickEvery creates a command that sends a tick message after the specified duration
func TickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// VercelTickEvery creates a command that sends a Vercel tick message after the specified duration
func VercelTickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return VercelTickMsg(t)
	})
}

// RefreshNow triggers an immediate git status refresh
func RefreshNow() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return git.CheckLocalStatus() },
		func() tea.Msg { return git.CheckStackStatus() },
	)
}
