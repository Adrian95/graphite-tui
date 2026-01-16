package state

import (
	"time"

	"github.com/Adrian95/graphite-tui/internal/git"
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
	SpeedCursor   int
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

// --- Menu State ---

type MenuData struct {
	Items  []git.MenuItem
	Cursor int
}

type MenuState struct {
	Data MenuData
}

func NewMenuState(items []git.MenuItem) *MenuState {
	return &MenuState{
		Data: MenuData{Items: items, Cursor: 0},
	}
}

func (s *MenuState) ID() StateID { return Menu }

func (s *MenuState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Menu, target)
}

func (s *MenuState) OnEnter() tea.Cmd { return nil }
func (s *MenuState) OnExit()          {}

// --- Input State ---

type InputData struct {
	Title       string
	Placeholder string
	Value       string
	IsGhostFix  bool
}

type InputState struct {
	Data InputData
}

func NewInputState(title, placeholder string, isGhostFix bool) *InputState {
	return &InputState{
		Data: InputData{
			Title:       title,
			Placeholder: placeholder,
			IsGhostFix:  isGhostFix,
		},
	}
}

func (s *InputState) ID() StateID { return Input }

func (s *InputState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Input, target)
}

func (s *InputState) OnEnter() tea.Cmd { return nil }
func (s *InputState) OnExit()          {}

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

// --- Wizard States ---

type WizardData struct {
	Stage         int // 1=Type, 2=Scope, 3=Summary, 4=Preview
	CommitTypes   []git.CommitType
	TypeIdx       int
	Scope         string
	RecentScopes  []string
	Summary       string
	ErrorMsg      string
	CommitMessage string
}

// WizardTypeState - Step 1: Select commit type
type WizardTypeState struct {
	Data *WizardData
}

func NewWizardTypeState(types []git.CommitType) *WizardTypeState {
	return &WizardTypeState{
		Data: &WizardData{
			Stage:       1,
			CommitTypes: types,
		},
	}
}

func (s *WizardTypeState) ID() StateID { return WizardType }

func (s *WizardTypeState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(WizardType, target)
}

func (s *WizardTypeState) OnEnter() tea.Cmd { return nil }
func (s *WizardTypeState) OnExit()          {}

// WizardScopeState - Step 2: Enter scope
type WizardScopeState struct {
	Data *WizardData
}

func NewWizardScopeState(data *WizardData) *WizardScopeState {
	data.Stage = 2
	return &WizardScopeState{Data: data}
}

func (s *WizardScopeState) ID() StateID { return WizardScope }

func (s *WizardScopeState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(WizardScope, target)
}

func (s *WizardScopeState) OnEnter() tea.Cmd { return nil }
func (s *WizardScopeState) OnExit()          {}

// WizardSummaryState - Step 3: Enter summary
type WizardSummaryState struct {
	Data *WizardData
}

func NewWizardSummaryState(data *WizardData) *WizardSummaryState {
	data.Stage = 3
	return &WizardSummaryState{Data: data}
}

func (s *WizardSummaryState) ID() StateID { return WizardSummary }

func (s *WizardSummaryState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(WizardSummary, target)
}

func (s *WizardSummaryState) OnEnter() tea.Cmd { return nil }
func (s *WizardSummaryState) OnExit()          {}

// WizardPreviewState - Step 4: Preview and confirm
type WizardPreviewState struct {
	Data *WizardData
}

func NewWizardPreviewState(data *WizardData) *WizardPreviewState {
	data.Stage = 4
	return &WizardPreviewState{Data: data}
}

func (s *WizardPreviewState) ID() StateID { return WizardPreview }

func (s *WizardPreviewState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(WizardPreview, target)
}

func (s *WizardPreviewState) OnEnter() tea.Cmd { return nil }
func (s *WizardPreviewState) OnExit()          {}

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
	ConfirmGhost ConfirmAction = "ghost"
	ConfirmReset ConfirmAction = "reset"
)

type ConfirmData struct {
	Action     ConfirmAction
	BranchName string // For ghost fix
}

type ConfirmState struct {
	Data ConfirmData
}

func NewConfirmState(action ConfirmAction, branchName string) *ConfirmState {
	return &ConfirmState{
		Data: ConfirmData{
			Action:     action,
			BranchName: branchName,
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

// --- Reflog State ---

type ReflogData struct {
	Items  []git.ReflogItem
	Cursor int
}

type ReflogState struct {
	Data ReflogData
}

func NewReflogState() *ReflogState {
	return &ReflogState{}
}

func (s *ReflogState) ID() StateID { return Reflog }

func (s *ReflogState) CanTransitionTo(target StateID) bool {
	return IsValidTransition(Reflog, target)
}

func (s *ReflogState) OnEnter() tea.Cmd {
	return git.LoadReflog()
}

func (s *ReflogState) OnExit() {}
