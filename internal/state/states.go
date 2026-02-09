package state

import (
	"time"

	"github.com/Adrian95/graphite-tui/v2/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Dashboard State ---

// DashboardData holds dashboard-specific state
type DashboardData struct {
	Branch        string
	Stack         []string
	Ahead         int
	Behind        int
	ChangedFiles  []git.ChangedFile
	HasStack      bool
	OnMain        bool
	GtInitialized bool
	Suggestion    string
	LastRefresh   time.Time
	FilesExpanded bool
}

// DashboardState represents the main dashboard view
type DashboardState struct {
	Data DashboardData
}

func NewDashboardState() *DashboardState {
	return &DashboardState{}
}

func (s *DashboardState) ID() StateID { return Dashboard }

func (s *DashboardState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Dashboard, target)
}

func (s *DashboardState) OnEnter() tea.Cmd { return nil }
func (s *DashboardState) OnExit()          {}

// --- Running State ---

type RunningData struct {
	Command string
}

type RunningState struct {
	Data RunningData
}

func NewRunningState(command string) *RunningState {
	return &RunningState{
		Data: RunningData{Command: command},
	}
}

func (s *RunningState) ID() StateID { return Running }

func (s *RunningState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Running, target)
}

func (s *RunningState) OnEnter() tea.Cmd { return nil }
func (s *RunningState) OnExit()          {}

// --- Output State ---

type OutputData struct {
	Output  string
	Error   error
	Command string
}

type OutputState struct {
	Data OutputData
}

func NewOutputState(output string, err error, command string) *OutputState {
	return &OutputState{
		Data: OutputData{
			Output:  output,
			Error:   err,
			Command: command,
		},
	}
}

func (s *OutputState) ID() StateID { return Output }

func (s *OutputState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Output, target)
}

func (s *OutputState) OnEnter() tea.Cmd { return nil }
func (s *OutputState) OnExit()          {}

// --- Stack State ---

type StackData struct {
	Items  []git.StackItem
	Cursor int
}

type StackState struct {
	Data StackData
}

func NewStackState() *StackState {
	return &StackState{}
}

func (s *StackState) ID() StateID { return Stack }

func (s *StackState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Stack, target)
}

func (s *StackState) OnEnter() tea.Cmd {
	return git.LoadStack()
}

func (s *StackState) OnExit() {}

// --- Help State ---

type HelpState struct{}

func NewHelpState() *HelpState { return &HelpState{} }

func (s *HelpState) ID() StateID { return Help }

func (s *HelpState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Help, target)
}

func (s *HelpState) OnEnter() tea.Cmd { return nil }
func (s *HelpState) OnExit()          {}

// --- Update State ---

type UpdateData struct {
	LatestVersion  string
	CurrentVersion string
	Checking       bool
}

type UpdateState struct {
	Data UpdateData
}

func NewUpdateState(currentVersion string) *UpdateState {
	return &UpdateState{
		Data: UpdateData{
			CurrentVersion: currentVersion,
			Checking:       true,
		},
	}
}

func (s *UpdateState) ID() StateID { return Update }

func (s *UpdateState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Update, target)
}

func (s *UpdateState) OnEnter() tea.Cmd { return nil }
func (s *UpdateState) OnExit()          {}

// --- Confirm State ---

type ConfirmAction string

const (
	ConfirmMerge ConfirmAction = "merge"
	ConfirmReset ConfirmAction = "reset"
)

type ConfirmData struct {
	Action ConfirmAction
}

type ConfirmState struct {
	Data ConfirmData
}

func NewConfirmState(action ConfirmAction) *ConfirmState {
	return &ConfirmState{
		Data: ConfirmData{
			Action: action,
		},
	}
}

func (s *ConfirmState) ID() StateID { return Confirm }

func (s *ConfirmState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Confirm, target)
}

func (s *ConfirmState) OnEnter() tea.Cmd { return nil }
func (s *ConfirmState) OnExit()          {}

// --- PostCommit State ---

type PostCommitData struct {
	CommitMessage string
}

type PostCommitState struct {
	Data PostCommitData
}

func NewPostCommitState(commitMsg string) *PostCommitState {
	return &PostCommitState{
		Data: PostCommitData{CommitMessage: commitMsg},
	}
}

func (s *PostCommitState) ID() StateID { return PostCommit }

func (s *PostCommitState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(PostCommit, target)
}

func (s *PostCommitState) OnEnter() tea.Cmd { return nil }
func (s *PostCommitState) OnExit()          {}
