package ui

import (
	"time"

	"github.com/Adrian95/graphite-tui/v2/internal/git"
	"github.com/charmbracelet/bubbletea"
)

// CachedStatus stores git status to reduce command frequency
type CachedStatus struct {
	Branch      string
	Ahead       int
	Behind      int
	FileCount   int
	LastUpdated time.Time
	IsValid     bool
}

var statusCache CachedStatus

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
		func() tea.Msg { return checkCachedLocalStatus() },
		func() tea.Msg { return checkCachedStackStatus() },
	)
}

// checkCachedLocalStatus returns cached status if recent, otherwise refreshes
func checkCachedLocalStatus() tea.Msg {
	// Cache for 5 seconds to reduce git commands
	if statusCache.IsValid && time.Since(statusCache.LastUpdated) < 5*time.Second {
		return git.LocalStatusMsg{
			Branch:        statusCache.Branch,
			Ahead:         statusCache.Ahead,
			Behind:        statusCache.Behind,
			GtInitialized: true, // Assume if we have cache
			OnMain:        statusCache.Branch == "main" || statusCache.Branch == "master",
		}
	}

	// Get fresh status
	msg := git.CheckLocalStatus()
	if localMsg, ok := msg.(git.LocalStatusMsg); ok {
		statusCache = CachedStatus{
			Branch:      localMsg.Branch,
			Ahead:       localMsg.Ahead,
			Behind:      localMsg.Behind,
			FileCount:   len(localMsg.ChangedFiles),
			LastUpdated: time.Now(),
			IsValid:     true,
		}
	}
	return msg
}

// checkCachedStackStatus returns cached stack status if recent
func checkCachedStackStatus() tea.Msg {
	// Cache stack status for 10 seconds (less critical)
	if statusCache.IsValid && time.Since(statusCache.LastUpdated) < 10*time.Second {
		return git.StackStatusMsg{
			HasStack: true, // Simplified - assume if we have cache
		}
	}

	return git.CheckStackStatus()
}

// SmartRefresh performs intelligent refreshing based on active panel
func SmartRefresh(focusIndex int) tea.Cmd {
	switch focusIndex {
	case 0: // Speed panel - minimal refresh
		return tea.Batch(
			func() tea.Msg { return checkMinimalStatus() },
		)
	case 1: // Files panel - full refresh needed
		return tea.Batch(
			func() tea.Msg { return checkCachedLocalStatus() },
		)
	case 2: // Vercel panel - moderate refresh
		return tea.Batch(
			func() tea.Msg { return checkCachedLocalStatus() },
		)
	case 3: // Stack panel - full refresh
		return tea.Batch(
			func() tea.Msg { return checkCachedLocalStatus() },
			func() tea.Msg { return checkCachedStackStatus() },
		)
	}
	return RefreshNow() // Fallback
}

// checkMinimalStatus returns lightweight status for speed panel
func checkMinimalStatus() tea.Msg {
	if statusCache.IsValid && time.Since(statusCache.LastUpdated) < 2*time.Second {
		return git.LocalStatusMsg{
			Branch:        statusCache.Branch,
			GtInitialized: true,
			OnMain:        statusCache.Branch == "main" || statusCache.Branch == "master",
		}
	}
	return git.CheckMinimalStatus()
}

// GetRefreshInterval returns appropriate refresh interval based on focus
func GetRefreshInterval(focusIndex int) time.Duration {
	switch focusIndex {
	case 0: // Speed panel - less frequent when not active
		return 8 * time.Second
	case 1: // Files panel - more frequent for file changes
		return 3 * time.Second
	case 2: // Vercel panel - moderate frequency
		return 5 * time.Second
	case 3: // Stack panel - frequent for navigation
		return 4 * time.Second
	}
	return 3 * time.Second
}
