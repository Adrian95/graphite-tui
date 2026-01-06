package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// --- File Change Types ---

// ChangedFile represents a file with changes
type ChangedFile struct {
	Status string // M, A, D, ??, etc.
	Path   string
}

// --- Ticker for periodic refresh ---

type tickMsg time.Time

// tickEvery creates a command that sends a tick message after the specified duration
func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// RefreshNow triggers an immediate git status refresh
func RefreshNow() tea.Cmd {
	return func() tea.Msg {
		return checkGitStatus()
	}
}
