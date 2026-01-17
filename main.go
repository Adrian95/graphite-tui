package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Adrian95/graphite-tui/internal/config"
	"github.com/Adrian95/graphite-tui/internal/git"
	"github.com/Adrian95/graphite-tui/internal/state"
	"github.com/Adrian95/graphite-tui/internal/ui"
	"github.com/Adrian95/graphite-tui/internal/ui/views"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Model ---

type model struct {
	// State management
	stateID   state.StateID
	prevState state.StateID

	// UI Components
	textInput textinput.Model
	viewport  viewport.Model
	spinner   spinner.Model
	fileList  list.Model

	// Focus management
	focusIndex int // 0: Speed Box, 1: File List

	// Dimensions
	width  int
	height int

	// Dashboard data
	dashboardData views.DashboardViewData

	// Menu state
	items  []git.MenuItem
	cursor int

	// Wizard state
	wizardData views.WizardViewData

	// Stack state
	stackData views.StackViewData

	// Reflog state
	reflogData views.ReflogViewData

	// Output state
	outputData views.OutputViewData

	// Confirm state
	confirmData views.ConfirmViewData

	// Quick commit state
	quickCommitData views.QuickCommitViewData

	// Commit choice state (amend vs stack)
	commitChoiceData views.CommitChoiceViewData

	// Options
	skipHooks bool

	// Flash message
	flashMessage string
	flashExpiry  time.Time

	// Version state
	latestVersion   string
	checkingUpdate  bool
	updateAvailable string

	// Post-commit
	justCommitted bool
	lastCommitMsg string
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50
	ti.Prompt = ""

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ColorAccent))

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ui.ColorBorder)).
		Padding(0, 1)

	fl := views.NewFileList()

	return model{
		stateID:    state.Dashboard,
		items:      git.GetMenuItems(),
		wizardData: views.WizardViewData{CommitTypes: git.GetCommitTypes()},
		textInput:  ti,
		spinner:    s,
		viewport:   vp,
		fileList:   fl,
		skipHooks:  false,
		dashboardData: views.DashboardViewData{
			CurrentVersion: config.GetCurrentVersion(),
		},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		ui.RefreshNow(),
		config.CheckForUpdates(),
		ui.TickEvery(3*time.Second),
	)
}

// --- Update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		key := msg.String()

		if key == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.stateID {
		case state.Dashboard:
			return m.handleDashboardKeys(msg)
		case state.Menu:
			return m.handleMenuKeys(key)
		case state.Input:
			return m.handleInputKeys(key, msg)
		case state.WizardType:
			return m.handleWizardTypeKeys(key)
		case state.WizardScope:
			return m.handleWizardScopeKeys(key, msg)
		case state.WizardSummary:
			return m.handleWizardSummaryKeys(key, msg)
		case state.WizardPreview:
			return m.handleWizardPreviewKeys(key)
		case state.Confirm:
			return m.handleConfirmKeys(key)
		case state.QuickCommit:
			return m.handleQuickCommitKeys(key, msg)
		case state.CommitChoice:
			return m.handleCommitChoiceKeys(key)
		case state.Stack:
			return m.handleStackKeys(key)
		case state.Reflog:
			return m.handleReflogKeys(key)
		case state.Help:
			if key == "esc" || key == "?" || key == "q" {
				m.stateID = state.Dashboard
			}
		case state.Update:
			return m.handleUpdateKeys(key)
		case state.PostCommit:
			return m.handlePostCommitKeys(key)
		case state.Output:
			if key == "esc" || key == "enter" || key == "q" {
				m.stateID = state.Dashboard
				return m, ui.RefreshNow()
			}
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		case state.Running:
			// Wait for command
		}

	case git.LocalStatusMsg:
		m.dashboardData.Branch = msg.Branch
		m.dashboardData.Ahead = msg.Ahead
		m.dashboardData.Behind = msg.Behind
		m.dashboardData.ChangedFiles = msg.ChangedFiles
		m.dashboardData.GtInitialized = msg.GtInitialized
		m.dashboardData.OnMain = msg.OnMain
		// Update file list
		items := views.FilesToItems(msg.ChangedFiles)
		cmd = m.fileList.SetItems(items)
		cmds = append(cmds, cmd)

	case git.StackStatusMsg:
		// We can add stack info to dashboardData if we want to display it in the future
		// For now, it might be used to check if we have a stack
		// The original code put stack into StatusMsg.Stack, but DashboardViewData didn't use it directly
		// except maybe for some logic?
		// Actually, StatusMsg had .Stack, but DashboardViewData did NOT have .Stack field.
		// It seems Stack was only used for `StackLoadedMsg` which is separate.
		// Wait, `StatusMsg` was calculating the stack but `DashboardViewData` ignores it?
		// Let's check `views/dashboard.go` again.

		// `DashboardViewData` has `Branch`, `Ahead`, `Behind`.
		// It doesn't seem to visualize the stack in the dashboard main view, only `StackView` does.
		// However, `git.CheckGitStatus` was fetching it.
		// `views/dashboard.go` uses `Ahead`/`Behind`.

		// So `StackStatusMsg` is potentially unused in Dashboard?
		// Ah, `HasStack` might be useful.
		_ = msg

	case git.StatusMsg:
		// Fallback for legacy calls if any
		m.dashboardData.Branch = msg.Branch
		m.dashboardData.Ahead = msg.Ahead
		m.dashboardData.Behind = msg.Behind
		m.dashboardData.ChangedFiles = msg.ChangedFiles
		m.dashboardData.GtInitialized = msg.GtInitialized
		m.dashboardData.OnMain = msg.OnMain
		items := views.FilesToItems(msg.ChangedFiles)
		cmd = m.fileList.SetItems(items)
		cmds = append(cmds, cmd)

	case git.CmdFinishedMsg:
		m.outputData.Error = msg.Err
		m.outputData.Command = msg.Command

		if msg.Err != nil {
			m.outputData.Output = msg.Stderr
			if m.outputData.Output == "" {
				m.outputData.Output = msg.Output
			}
			m.stateID = state.Output
			m.justCommitted = false
		} else {
			m.outputData.Output = msg.Output
			if m.justCommitted {
				m.stateID = state.PostCommit
				m.outputData.IsPostCommit = true
				m.outputData.CommitMessage = m.lastCommitMsg
			} else {
				m.stateID = state.Output
				m.outputData.IsPostCommit = false
			}
		}

		m.outputData.IsRunning = false
		m.viewport.SetContent(m.outputData.Output)
		return m, ui.RefreshNow()

	case git.StackLoadedMsg:
		m.stackData.Items = msg
		for i, item := range msg {
			if item.Current {
				m.stackData.Cursor = i
				break
			}
		}

	case git.ReflogLoadedMsg:
		m.reflogData.Items = msg
		m.reflogData.Cursor = 0

	case config.VersionCheckMsg:
		m.checkingUpdate = false
		if msg.Err == nil && config.IsNewerVersion(config.GetCurrentVersion(), msg.LatestVersion) {
			m.latestVersion = msg.LatestVersion
			m.updateAvailable = msg.LatestVersion
			m.dashboardData.UpdateAvailable = msg.LatestVersion
		} else if msg.Err == nil {
			m.latestVersion = msg.LatestVersion
		}

	case config.UpdateCompleteMsg:
		m.stateID = state.Output
		m.outputData.IsRunning = false
		if msg.Success {
			m.outputData.Output = msg.Message + "\n\nPlease restart the application."
			m.outputData.Error = nil
		} else {
			m.outputData.Output = msg.Message
			m.outputData.Error = msg.Err
		}
		m.viewport.SetContent(m.outputData.Output)

	case config.UninstallCompleteMsg:
		if msg.Success {
			fmt.Println("Graphite TUI has been uninstalled. Goodbye!")
			return m, tea.Quit
		}
		m.stateID = state.Output
		m.outputData.Error = msg.Err
		m.outputData.Output = "Uninstall failed: " + msg.Err.Error()
		m.viewport.SetContent(m.outputData.Output)

	case ui.TickMsg:
		if m.flashMessage != "" && time.Now().After(m.flashExpiry) {
			m.flashMessage = ""
			m.dashboardData.FlashMessage = ""
		}
		if m.stateID != state.Running {
			cmds = append(cmds, ui.RefreshNow())
		}
		cmds = append(cmds, ui.TickEvery(3*time.Second))

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 12
	}

	return m, tea.Batch(cmds...)
}

// --- Key Handlers ---

func (m model) handleDashboardKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global Dashboard keys (Quit, Tab)
	if key == "q" && m.focusIndex != 1 { // Only quit if not filtering/in list
		return m, tea.Quit
	}

	if key == "tab" {
		m.focusIndex = (m.focusIndex + 1) % 2
		// Adjust list styling based on focus
		if m.focusIndex == 1 {
			m.fileList.Styles.Title = ui.BoxTitleStyle.Foreground(lipgloss.Color(ui.ColorAccent))
		} else {
			m.fileList.Styles.Title = ui.BoxTitleStyle
		}
		return m, nil
	}

	// File List Focus
	if m.focusIndex == 1 {
		var cmd tea.Cmd
		m.fileList, cmd = m.fileList.Update(msg)
		return m, cmd
	}

	// Speed Box / Standard Dashboard Focus (Index 0)
	switch key {

	case "up", "k":
		if m.dashboardData.SpeedCursor > 0 {
			m.dashboardData.SpeedCursor--
		}
		return m, nil

	case "down", "j":
		if m.dashboardData.SpeedCursor < 3 {
			m.dashboardData.SpeedCursor++
		}
		return m, nil

	case "enter":
		return m.executeSpeedAction()

	case "i":
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt init"}
		return m, git.ExecuteInteractive("gt init --no-interactive", "", m.skipHooks)

	case "s":
		m.stateID = state.WizardType
		m.wizardData = views.WizardViewData{
			Stage:       1,
			CommitTypes: git.GetCommitTypes(),
		}
		return m, nil

	case "p":
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt submit"}
		return m, git.ExecuteSubmit(m.skipHooks)

	case "f":
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "iterate"}
		return m, git.ExecuteIterate(m.skipHooks)

	case "y":
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt sync"}
		return m, git.ExecuteSync(m.skipHooks)

	case "d":
		m.confirmData = views.ConfirmViewData{Action: "merge"}
		m.stateID = state.Confirm
		return m, nil

	case "g":
		m.stateID = state.Stack
		m.stackData.Cursor = 0
		return m, git.LoadStack()

	case "x":
		m.stateID = state.Input
		m.textInput.Reset()
		m.textInput.Placeholder = "New branch name to rescue your work..."
		m.confirmData = views.ConfirmViewData{Action: "ghost"}
		return m, textinput.Blink

	case "m":
		m.stateID = state.Menu
		m.cursor = 0
		return m, nil

	case "?":
		m.stateID = state.Help
		return m, nil

	case "u":
		m.stateID = state.Update
		m.checkingUpdate = true
		return m, config.CheckForUpdates()

	case "h":
		m.skipHooks = !m.skipHooks
		m.dashboardData.SkipHooks = m.skipHooks
		if m.skipHooks {
			m.setFlash("Hooks OFF - commits will skip pre-commit hooks")
		} else {
			m.setFlash("Hooks ON - pre-commit hooks will run")
		}
		return m, nil

	case "r":
		return m, ui.RefreshNow()

	case "R":
		if len(m.dashboardData.ChangedFiles) > 0 {
			m.confirmData = views.ConfirmViewData{Action: "reset"}
			m.stateID = state.Confirm
			return m, nil
		}
		m.setFlash("Nothing to reset - working tree is clean")
		return m, nil

	case "z":
		m.stateID = state.Reflog
		return m, git.LoadReflog()
	}

	return m, nil
}

func (m *model) setFlash(msg string) {
	m.flashMessage = msg
	m.flashExpiry = time.Now().Add(3 * time.Second)
	m.dashboardData.FlashMessage = msg
}

func (m model) handleMenuKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "m":
		m.stateID = state.Dashboard
		return m, nil

	case "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}

	case "h":
		m.skipHooks = !m.skipHooks
		m.dashboardData.SkipHooks = m.skipHooks

	case "enter":
		if m.cursor < 0 || m.cursor >= len(m.items) {
			return m, nil
		}
		selected := m.items[m.cursor]

		if selected.Title == "Start" {
			m.stateID = state.WizardType
			m.wizardData = views.WizardViewData{Stage: 1, CommitTypes: git.GetCommitTypes()}
			return m, nil
		}

		if selected.Title == "Stack Map" {
			m.stateID = state.Stack
			return m, git.LoadStack()
		}

		if selected.Title == "Ghost Fix" {
			m.stateID = state.Input
			m.textInput.Reset()
			m.textInput.Placeholder = "New branch name..."
			m.confirmData = views.ConfirmViewData{Action: "ghost"}
			return m, textinput.Blink
		}

		if selected.Command != "" {
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: selected.Title}
			return m, git.ExecuteInteractive(selected.Command, "", m.skipHooks)
		}
	}

	return m, nil
}

func (m model) handleInputKeys(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.stateID = state.Dashboard
		return m, nil

	case "enter":
		inputVal := m.textInput.Value()
		if inputVal == "" {
			return m, nil
		}

		if m.confirmData.Action == "ghost" {
			m.confirmData.BranchName = inputVal
			m.stateID = state.Confirm
			return m, nil
		}

		m.stateID = state.Running
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) handleWizardTypeKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.stateID = state.Dashboard
		return m, nil

	case "up", "k":
		if m.wizardData.TypeIdx > 0 {
			m.wizardData.TypeIdx--
		}

	case "down", "j":
		if m.wizardData.TypeIdx < len(m.wizardData.CommitTypes)-1 {
			m.wizardData.TypeIdx++
		}

	case "enter":
		m.stateID = state.WizardScope
		m.wizardData.Stage = 2
		m.wizardData.RecentScopes = git.GetRecentScopes()
		m.textInput.Reset()
		m.textInput.Placeholder = "auth, ui, api (optional)"
		return m, textinput.Blink
	}

	return m, nil
}

func (m model) handleWizardScopeKeys(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.stateID = state.Dashboard
		return m, nil

	case "enter":
		m.wizardData.Scope = m.textInput.Value()
		m.stateID = state.WizardSummary
		m.wizardData.Stage = 3
		m.textInput.Reset()
		m.textInput.Placeholder = "Add new login button"
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) handleWizardSummaryKeys(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.stateID = state.Dashboard
		return m, nil

	case "backspace":
		if m.textInput.Value() == "" {
			m.stateID = state.WizardScope
			m.wizardData.Stage = 2
			m.textInput.Reset()
			m.textInput.SetValue(m.wizardData.Scope)
			return m, textinput.Blink
		}

	case "enter":
		m.wizardData.Summary = m.textInput.Value()
		if m.wizardData.Summary == "" {
			m.wizardData.ErrorMsg = "Summary is required!"
			return m, nil
		}
		m.wizardData.ErrorMsg = ""

		// Build commit message
		msgStr := m.wizardData.CommitTypes[m.wizardData.TypeIdx].Label
		if m.wizardData.Scope != "" {
			msgStr += fmt.Sprintf("(%s)", m.wizardData.Scope)
		}
		msgStr += fmt.Sprintf(": %s", m.wizardData.Summary)
		m.wizardData.CommitMessage = msgStr
		m.lastCommitMsg = msgStr

		m.stateID = state.WizardPreview
		m.wizardData.Stage = 4
		m.wizardData.FileCount = len(m.dashboardData.ChangedFiles)
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.wizardData.InputValue = m.textInput.Value()
	return m, cmd
}

func (m model) handleWizardPreviewKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "backspace":
		m.stateID = state.WizardSummary
		m.wizardData.Stage = 3
		m.textInput.Reset()
		m.textInput.SetValue(m.wizardData.Summary)
		return m, textinput.Blink

	case "enter":
		m.justCommitted = true
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt create"}
		return m, git.ExecuteCommit(m.lastCommitMsg, m.skipHooks)
	}

	return m, nil
}

func (m model) handleConfirmKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "enter":
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true}

		switch m.confirmData.Action {
		case "merge":
			m.outputData.Command = "merge"
			return m, git.ExecuteInteractive("gt merge --no-interactive && gt sync --no-interactive", "", m.skipHooks)
		case "ghost":
			m.outputData.Command = "Ghost Fix"
			return m, git.ExecuteGhostFix(m.confirmData.BranchName, m.skipHooks)
		case "reset":
			m.outputData.Command = "hard reset"
			return m, git.ExecuteHardReset()
		}
		m.stateID = state.Dashboard
		return m, nil

	case "n", "esc":
		m.stateID = state.Dashboard
		return m, nil
	}

	return m, nil
}

func (m model) handleQuickCommitKeys(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.stateID = state.Dashboard
		return m, nil

	case "enter":
		commitMsg := m.textInput.Value()
		if commitMsg == "" {
			m.quickCommitData.ErrorMsg = "Message required"
			return m, nil
		}

		m.lastCommitMsg = commitMsg
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true}

		if m.quickCommitData.IsAmend {
			m.outputData.Command = "iterate"
			return m, git.ExecuteIterateWithMsg(commitMsg, m.skipHooks)
		}

		m.justCommitted = true
		m.outputData.Command = "turbo ship"
		return m, git.ExecuteTurboShipWithMsg(commitMsg, m.skipHooks)
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.quickCommitData.InputValue = m.textInput.Value()
	m.quickCommitData.ErrorMsg = "" // Clear error on typing
	return m, cmd
}

func (m model) handleCommitChoiceKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		m.commitChoiceData.Selected = 0 // Amend
		return m, nil

	case "down", "j":
		m.commitChoiceData.Selected = 1 // Stack
		return m, nil

	case "enter":
		// Transition to QuickCommit with the chosen mode
		m.stateID = state.QuickCommit
		m.textInput.Reset()
		if m.commitChoiceData.Selected == 0 {
			m.textInput.Placeholder = "fix: update implementation"
		} else {
			m.textInput.Placeholder = "feat: add new feature"
		}
		m.quickCommitData = views.QuickCommitViewData{
			FileCount: m.commitChoiceData.FileCount,
			IsAmend:   m.commitChoiceData.Selected == 0,
		}
		return m, textinput.Blink

	case "esc", "q":
		m.stateID = state.Dashboard
		return m, nil
	}

	return m, nil
}

func (m model) handleStackKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.stateID = state.Dashboard
		return m, nil

	case "up", "k":
		if m.stackData.Cursor > 0 {
			m.stackData.Cursor--
		}

	case "down", "j":
		if m.stackData.Cursor < len(m.stackData.Items)-1 {
			m.stackData.Cursor++
		}

	case "enter":
		if len(m.stackData.Items) > 0 {
			target := m.stackData.Items[m.stackData.Cursor].Name
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "gt checkout"}
			return m, git.ExecuteCheckout(target)
		}

	case "r":
		return m, git.LoadStack()
	}

	return m, nil
}

func (m model) handleReflogKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.stateID = state.Dashboard
		return m, nil

	case "up", "k":
		if m.reflogData.Cursor > 0 {
			m.reflogData.Cursor--
		}

	case "down", "j":
		if m.reflogData.Cursor < len(m.reflogData.Items)-1 {
			m.reflogData.Cursor++
		}

	case "enter":
		if len(m.reflogData.Items) > 0 {
			// We will hard reset to this hash
			target := m.reflogData.Items[m.reflogData.Cursor]
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "Reset to " + target.Hash}

			// Custom reset command
			cmd := exec.Command("git", "reset", "--hard", target.Hash)
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return git.CmdFinishedMsg{Err: err, Command: "Reset to " + target.Hash, Output: "Reset complete."}
			})
		}
	}

	return m, nil
}

func (m model) handleUpdateKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.stateID = state.Dashboard
		return m, nil

	case "u":
		if m.latestVersion != "" && config.IsNewerVersion(config.GetCurrentVersion(), m.latestVersion) {
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "update"}
			return m, config.PerformUpdate()
		}

	case "x":
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "uninstall"}
		return m, config.PerformUninstall()
	}

	return m, nil
}

func (m model) handlePostCommitKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "enter":
		m.justCommitted = false
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt submit"}
		return m, git.ExecuteSubmit(m.skipHooks)

	case "n", "esc":
		m.justCommitted = false
		m.stateID = state.Dashboard
		return m, ui.RefreshNow()

	case "f":
		m.justCommitted = false
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt modify"}
		return m, git.ExecuteFix(m.skipHooks)
	}

	return m, nil
}

func (m model) executeSpeedAction() (tea.Model, tea.Cmd) {
	switch m.dashboardData.SpeedCursor {
	case 0: // Ship
		if len(m.dashboardData.ChangedFiles) > 0 {
			if m.dashboardData.OnMain {
				// On main: go straight to QuickCommit (create new branch)
				m.stateID = state.QuickCommit
				m.textInput.Reset()
				m.textInput.Placeholder = "feat: add new feature"
				m.quickCommitData = views.QuickCommitViewData{
					FileCount: len(m.dashboardData.ChangedFiles),
					IsAmend:   false,
				}
				return m, textinput.Blink
			}
			// On feature branch: show choice dialog (amend vs stack)
			m.stateID = state.CommitChoice
			m.commitChoiceData = views.CommitChoiceViewData{
				Branch:    m.dashboardData.Branch,
				FileCount: len(m.dashboardData.ChangedFiles),
				Selected:  0, // Default to Amend
			}
			return m, nil
		} else if m.dashboardData.Ahead > 0 {
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "gt submit"}
			return m, git.ExecuteSubmit(m.skipHooks)
		}
		m.setFlash("Nothing to ship - no changes or unpushed commits")
		return m, nil

	case 1: // Iterate
		if len(m.dashboardData.ChangedFiles) > 0 {
			// Show quick commit wizard in amend mode
			m.stateID = state.QuickCommit
			m.textInput.Reset()
			m.textInput.Placeholder = "fix: update implementation"
			m.quickCommitData = views.QuickCommitViewData{
				FileCount: len(m.dashboardData.ChangedFiles),
				IsAmend:   true,
			}
			return m, textinput.Blink
		}
		m.setFlash("Nothing to amend - no changes")
		return m, nil

	case 2: // Reset
		if len(m.dashboardData.ChangedFiles) > 0 {
			m.confirmData = views.ConfirmViewData{Action: "reset"}
			m.stateID = state.Confirm
			return m, nil
		}
		m.setFlash("Nothing to reset - working tree is clean")
		return m, nil

	case 3: // Undo
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt undo"}
		return m, git.ExecuteUndo()
	}

	return m, nil
}

// --- View ---

func (m model) View() string {
	ctx := views.NewRenderContext(m.width, m.height)

	// Sync file list to dashboard data
	m.dashboardData.FileList = m.fileList

	switch m.stateID {
	case state.Dashboard, state.Menu:
		return views.RenderDashboard(ctx, m.dashboardData)

	case state.WizardType:
		return views.RenderCentered(views.RenderWizardType(m.wizardData))

	case state.WizardScope:
		return views.RenderCentered(views.RenderWizardScope(m.wizardData, m.textInput.View()))

	case state.WizardSummary:
		return views.RenderCentered(views.RenderWizardSummary(m.wizardData, m.textInput.View()))

	case state.WizardPreview:
		return views.RenderCentered(views.RenderWizardPreview(m.wizardData))

	case state.Input:
		data := views.InputViewData{
			Title:       "Rescue Mode",
			Description: "Name a new branch to save your work:",
			InputView:   m.textInput.View(),
		}
		return views.RenderCentered(views.RenderInput(data))

	case state.QuickCommit:
		return views.RenderCentered(views.RenderQuickCommit(m.quickCommitData, m.textInput.View()))

	case state.CommitChoice:
		return views.RenderCentered(views.RenderCommitChoice(m.commitChoiceData))

	case state.Confirm:
		return views.RenderCentered(views.RenderConfirm(m.confirmData))

	case state.Stack:
		return views.RenderCentered(views.RenderStack(m.stackData))

	case state.Reflog:
		return views.RenderCentered(views.RenderReflog(m.reflogData))

	case state.Running, state.Output, state.PostCommit:
		m.outputData.SpinnerView = m.spinner.View()
		return views.RenderCentered(views.RenderOutput(m.outputData))

	case state.Help:
		return views.RenderCentered(views.RenderHelp())

	case state.Update:
		data := views.UpdateViewData{
			CurrentVersion: config.GetCurrentVersion(),
			LatestVersion:  m.latestVersion,
			IsChecking:     m.checkingUpdate,
			IsNewer:        m.latestVersion != "" && config.IsNewerVersion(config.GetCurrentVersion(), m.latestVersion),
			SpinnerView:    m.spinner.View(),
		}
		return views.RenderCentered(views.RenderUpdate(data))

	default:
		return views.RenderDashboard(ctx, m.dashboardData)
	}
}

// --- Main ---

func main() {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		fmt.Println("Not a git repository. Please run from a git project directory.")
		os.Exit(1)
	}

	if _, err := exec.LookPath("gt"); err != nil {
		fmt.Println("Graphite CLI (gt) not found. Please install it first:")
		fmt.Println("  https://graphite.dev/docs/cli/installation")
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
