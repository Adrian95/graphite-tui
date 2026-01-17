package state

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// --- State IDs ---

type StateID int

const (
	Dashboard StateID = iota
	Menu
	Input
	Running
	Output
	WizardType
	WizardScope
	WizardSummary
	WizardPreview
	Stack
	Help
	Update
	Confirm
	Startup
	PostCommit
	QuickCommit
	CommitChoice
	Reflog
	Stash
	Vercel
)

// String returns the state name for debugging
func (s StateID) String() string {
	names := []string{
		"Dashboard", "Menu", "Input", "Running", "Output",
		"WizardType", "WizardScope", "WizardSummary", "WizardPreview",
		"Stack", "Help", "Update", "Confirm", "Startup", "PostCommit",
		"QuickCommit", "CommitChoice", "Reflog", "Stash", "Vercel",
	}
	if int(s) < len(names) {
		return names[s]
	}
	return fmt.Sprintf("Unknown(%d)", s)
}

// --- State Context Interface ---

// StateContext defines the contract for all view states
type StateContext interface {
	// ID returns the unique identifier for this state
	ID() StateID

	// CanTransitionTo validates if transition to target state is allowed
	CanTransitionTo(target StateID) bool

	// OnEnter is called when entering this state, returns initialization command
	OnEnter() tea.Cmd

	// OnExit is called when leaving this state, for cleanup
	OnExit()
}

// --- State Manager ---

// TransitionError represents an invalid state transition attempt
type TransitionError struct {
	From StateID
	To   StateID
}

func (e TransitionError) Error() string {
	return fmt.Sprintf("invalid state transition: %s -> %s", e.From, e.To)
}

// Manager handles state transitions with validation
type Manager struct {
	current  StateContext
	previous StateContext
	history  []StateID
}

// NewManager creates a state manager with initial state
func NewManager(initial StateContext) *Manager {
	return &Manager{
		current: initial,
		history: []StateID{initial.ID()},
	}
}

// Current returns the current state
func (m *Manager) Current() StateContext {
	return m.current
}

// CurrentID returns the current state ID
func (m *Manager) CurrentID() StateID {
	return m.current.ID()
}

// Previous returns the previous state (may be nil)
func (m *Manager) Previous() StateContext {
	return m.previous
}

// TransitionTo attempts to transition to a new state
// Returns the OnEnter command if successful, error if transition is invalid
func (m *Manager) TransitionTo(next StateContext) (tea.Cmd, error) {
	if !m.current.CanTransitionTo(next.ID()) {
		return nil, TransitionError{From: m.current.ID(), To: next.ID()}
	}

	// Exit current state
	m.current.OnExit()

	// Update state
	m.previous = m.current
	m.current = next
	m.history = append(m.history, next.ID())

	// Enter new state
	return next.OnEnter(), nil
}

// ForceTransition transitions without validation (use sparingly)
func (m *Manager) ForceTransition(next StateContext) tea.Cmd {
	m.current.OnExit()
	m.previous = m.current
	m.current = next
	m.history = append(m.history, next.ID())
	return next.OnEnter()
}

// History returns the state transition history
func (m *Manager) History() []StateID {
	return m.history
}

// --- Transition Rules ---

// ValidTransitions defines allowed transitions from each state
var ValidTransitions = map[StateID][]StateID{
	Dashboard: {
		Menu, Input, Running, WizardType, Stack, Help, Update, Confirm, QuickCommit, CommitChoice, Reflog, Stash, Vercel,
	},
	Menu: {
		Dashboard, Input, Running, WizardType, Stack,
	},
	Input: {
		Dashboard, Running, Confirm,
	},
	Running: {
		Output, PostCommit, Dashboard,
	},
	Output: {
		Dashboard,
	},
	WizardType: {
		Dashboard, WizardScope,
	},
	WizardScope: {
		Dashboard, WizardType, WizardSummary,
	},
	WizardSummary: {
		Dashboard, WizardScope, WizardPreview,
	},
	WizardPreview: {
		Dashboard, WizardSummary, Running,
	},
	Stack: {
		Dashboard, Running,
	},
	Help: {
		Dashboard,
	},
	Update: {
		Dashboard, Running,
	},
	Confirm: {
		Dashboard, Running,
	},
	Startup: {
		Dashboard,
	},
	PostCommit: {
		Dashboard, Running,
	},
	QuickCommit: {
		Dashboard, Running,
	},
	CommitChoice: {
		Dashboard, QuickCommit,
	},
	Reflog: {
		Dashboard, Running,
	},
	Stash: {
		Dashboard, Running,
	},
	Vercel: {
		Dashboard, Running,
	},
}

// IsValidTransition checks if a transition is allowed
func IsValidTransition(from, to StateID) bool {
	allowed, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, target := range allowed {
		if target == to {
			return true
		}
	}
	return false
}
