package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Adrian95/graphite-tui/v2/internal/config"
	"github.com/Adrian95/graphite-tui/v2/internal/git"
	"github.com/Adrian95/graphite-tui/v2/internal/state"
	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/Adrian95/graphite-tui/v2/internal/ui/views"
	"github.com/Adrian95/graphite-tui/v2/internal/vercel"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Model ---

type pendingAction int

const (
	pendingNone pendingAction = iota
	pendingSubmit
	pendingSync
	pendingIterate
	pendingTurboShip
)

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
	focusIndex int // 0: Files, 1: Vercel, 2: Stack

	// Dimensions
	width  int
	height int

	// Dashboard data
	dashboardData views.DashboardViewData

	// Stack state
	stackData views.StackViewData

	// Vercel state
	vercelData   views.VercelViewData
	vercelClient *vercel.Client

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

	// Merged ancestor resolution
	mergedAncestorData views.MergedAncestorViewData
	pendingAction      pendingAction
	pendingCommitMsg   string
}

func initialModel() model {
	vercel.LoadEnvFiles(".env.local", ".env")

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

	vcfg := vercel.LoadConfig()
	var vclient *vercel.Client
	vercelData := views.VercelViewData{Enabled: vcfg.Enabled()}
	if vcfg.Enabled() {
		vclient = vercel.NewClient(vcfg)
	}

	return model{
		stateID:    state.Dashboard,
		textInput:  ti,
		spinner:    s,
		viewport:   vp,
		fileList:   fl,
		vercelData: vercelData,
		vercelClient: vclient,
		skipHooks:  false,
		mergedAncestorData: views.MergedAncestorViewData{
			Selected: 0,
		},
		dashboardData: views.DashboardViewData{
			CurrentVersion: config.GetCurrentVersion(),
			VercelSummary:  vercelData,
			StackData:      views.StackViewData{},
		},
	}
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textinput.Blink,
		m.spinner.Tick,
		ui.RefreshNow(),
		git.LoadStack(),
		config.CheckForUpdates(),
		ui.TickEvery(3 * time.Second),
	}

	if m.vercelClient != nil {
		cmds = append(cmds, vercel.FetchStatus(m.vercelClient, nil))
		cmds = append(cmds, ui.VercelTickEvery(30*time.Second))
	}

	return tea.Batch(cmds...)
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
		case state.Confirm:
			return m.handleConfirmKeys(key)
		case state.QuickCommit:
			return m.handleQuickCommitKeys(key, msg)
		case state.CommitChoice:
			return m.handleCommitChoiceKeys(key)
		case state.Stack:
			return m.handleStackKeys(key)
		case state.Help:
			if key == "esc" || key == "?" || key == "q" {
				m.stateID = state.Dashboard
			}
		case state.Update:
			return m.handleUpdateKeys(key)
		case state.PostCommit:
			return m.handlePostCommitKeys(key)
		case state.MergedAncestor:
			return m.handleMergedAncestorKeys(key)
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
		m.dashboardData.GtInitialized = msg.GtInitialized
		m.dashboardData.OnMain = msg.OnMain
		m.dashboardData.StatusLoaded = true

		// Metrics
		m.dashboardData.LinesAdded = msg.LinesAdded
		m.dashboardData.LinesRemoved = msg.LinesRemoved
		m.dashboardData.NewFiles = msg.NewFiles
		m.dashboardData.ModFiles = msg.ModFiles
		m.dashboardData.DelFiles = msg.DelFiles
		m.dashboardData.StagedCount = msg.StagedCount

		// Only update file list if files actually changed (dirty check)
		if !filesEqual(m.dashboardData.ChangedFiles, msg.ChangedFiles) {
			m.dashboardData.ChangedFiles = msg.ChangedFiles
			items := views.FilesToItems(msg.ChangedFiles)
			cmd = m.fileList.SetItems(items)
			cmds = append(cmds, cmd)
		}

	case git.StackStatusMsg:
		_ = msg

	case git.StatusMsg:
		// Fallback for legacy calls if any
		m.dashboardData.Branch = msg.Branch
		m.dashboardData.Ahead = msg.Ahead
		m.dashboardData.Behind = msg.Behind
		m.dashboardData.GtInitialized = msg.GtInitialized
		m.dashboardData.OnMain = msg.OnMain
		m.dashboardData.StatusLoaded = true

		if !filesEqual(m.dashboardData.ChangedFiles, msg.ChangedFiles) {
			m.dashboardData.ChangedFiles = msg.ChangedFiles
			items := views.FilesToItems(msg.ChangedFiles)
			cmd = m.fileList.SetItems(items)
			cmds = append(cmds, cmd)
		}

	case git.MergedAncestorCheckMsg:
		if msg.Err != nil {
			m.pendingAction = pendingNone
			m.pendingCommitMsg = ""
			m.stateID = state.Output
			m.outputData = views.OutputViewData{
				IsRunning: false,
				Command:   "check merged ancestors",
				Output:    msg.Err.Error(),
				Error:     msg.Err,
			}
			m.viewport.SetContent(m.outputData.Output)
			return m, nil
		}

		if len(msg.Issues) == 0 {
			return m, m.runPendingAction()
		}

		m.mergedAncestorData = views.MergedAncestorViewData{
			Issues:        msg.Issues,
			Selected:      0,
			CurrentBranch: msg.CurrentBranch,
			TrunkBranch:   msg.TrunkBranch,
		}
		m.stateID = state.MergedAncestor
		return m, nil

	case git.CmdFinishedMsg:
		m.outputData.Error = msg.Err
		m.outputData.Command = msg.Command

		// Invalidate cache for commands that modify files or branches
		ui.InvalidateCache()

		if msg.Err != nil {
			m.outputData.Output = msg.Stderr
			if m.outputData.Output == "" {
				m.outputData.Output = msg.Output
			}
			m.stateID = state.Output
			m.justCommitted = false
			m.pendingAction = pendingNone
			m.pendingCommitMsg = ""
		} else {
			m.outputData.Output = msg.Output
			if msg.Command == "unlink PR" && m.pendingAction == pendingSubmit {
				m.pendingAction = pendingNone
				m.pendingCommitMsg = ""
				m.stateID = state.Running
				m.outputData = views.OutputViewData{IsRunning: true, Command: "gt submit"}
				return m, git.ExecuteSubmit(m.skipHooks)
			}

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
		// Load stack after branch-modifying commands
		if isBranchModifyingCommand(msg.Command) {
			return m, tea.Batch(ui.RefreshNow(), git.LoadStack())
		}
		return m, ui.RefreshNow()

	case git.StackLoadedMsg:
		m.stackData.Items = msg
		for i, item := range msg {
			if item.Current {
				m.stackData.Cursor = i
				break
			}
		}
		m.dashboardData.StackData = m.stackData

	case vercel.StatusMsg:
		m.vercelData.Enabled = msg.Enabled
		m.vercelData.Statuses = msg.Statuses
		m.vercelData.Summary = msg.Summary
		m.vercelData.Cursor = 0
		if msg.Err != nil {
			m.vercelData.HasError = true
			m.vercelData.ErrorString = msg.Err.Error()
		} else {
			m.vercelData.HasError = false
			m.vercelData.ErrorString = ""
		}

	case vercel.ActionMsg:
		if msg.Err != nil {
			m.setFlash(msg.Err.Error())
		} else if msg.Message != "" {
			m.setFlash(msg.Message)
		}

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
			cmds = append(cmds, ui.SmartRefresh(m.focusIndex))
		}
		cmds = append(cmds, ui.TickEvery(ui.GetRefreshInterval(m.focusIndex)))
		return m, tea.Batch(cmds...)

	case ui.VercelTickMsg:
		// Always refresh Vercel when client is available
		if m.vercelClient != nil {
			branches := m.collectVercelBranches()
			cmds = append(cmds, vercel.FetchStatus(m.vercelClient, branches))
			cmds = append(cmds, ui.VercelTickEvery(30*time.Second))
		}
		return m, tea.Batch(cmds...)

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

	// Global quit (except in file list where q might conflict)
	if key == "q" && m.focusIndex != 0 {
		return m, tea.Quit
	}

	// Global update shortcut (except in file list where u = unstage)
	if key == "u" && m.focusIndex != 0 {
		m.stateID = state.Update
		m.checkingUpdate = true
		return m, config.CheckForUpdates()
	}

	// Global help shortcut
	if key == "?" {
		m.stateID = state.Help
		return m, nil
	}

	if key == "tab" || key == "right" {
		m.focusIndex = (m.focusIndex + 1) % 3
		if m.focusIndex == 0 {
			m.fileList.Styles.Title = ui.BoxTitleStyle.Foreground(lipgloss.Color(ui.ColorAccent))
		} else {
			m.fileList.Styles.Title = ui.BoxTitleStyle
		}
		return m, nil
	}

	if key == "shift+tab" || key == "left" {
		m.focusIndex = (m.focusIndex - 1 + 3) % 3
		if m.focusIndex == 0 {
			m.fileList.Styles.Title = ui.BoxTitleStyle.Foreground(lipgloss.Color(ui.ColorAccent))
		} else {
			m.fileList.Styles.Title = ui.BoxTitleStyle
		}
		return m, nil
	}

	if key == "esc" {
		m.focusIndex = 0
		return m, nil
	}

	// File List Focus
	if m.focusIndex == 0 {
		var cmd tea.Cmd

		switch key {
		case "space":
			if item, ok := m.fileList.SelectedItem().(views.FileItem); ok {
				return m, git.ExecuteStage(item.File.Path, !item.File.Staged)
			}
		case "a":
			return m, git.ExecuteStageAll(true)
		case "u":
			return m, git.ExecuteStageAll(false)
		case "q":
			return m, tea.Quit
		default:
			// Pass through to list for navigation keys
			m.fileList, cmd = m.fileList.Update(msg)
			return m, cmd
		}

		return m, nil
	}

	// Vercel Focus
	if m.focusIndex == 1 {
		return m.handleVercelKeys(key)
	}

	// Stack Focus
	if m.focusIndex == 2 {
		return m.handleStackPanelKeys(key)
	}

	// Workflow keys (no panel focused)
	switch key {

	case "i":
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt init"}
		return m, git.ExecuteInteractive("gt init --no-interactive", "", m.skipHooks)

	case "s":
		localStatus := git.CheckLocalStatus().(git.LocalStatusMsg)
		hasChanges := len(localStatus.ChangedFiles) > 0
		if hasChanges {
			if m.dashboardData.OnMain {
				m.stateID = state.QuickCommit
				m.textInput.Reset()
				m.textInput.Placeholder = "feat: add new feature"
				m.quickCommitData = views.QuickCommitViewData{
					FileCount: len(localStatus.ChangedFiles),
					IsAmend:   false,
				}
				return m, textinput.Blink
			}
			m.stateID = state.CommitChoice
			m.commitChoiceData = views.CommitChoiceViewData{
				Branch:    m.dashboardData.Branch,
				FileCount: len(localStatus.ChangedFiles),
				Selected:  0,
			}
			return m, nil
		} else if m.dashboardData.Ahead > 0 {
			return m.startMergedAncestorCheck(pendingSubmit)
		}
		m.setFlash("Nothing to ship")
		return m, nil

	case "f":
		localStatus := git.CheckLocalStatus().(git.LocalStatusMsg)
		if len(localStatus.ChangedFiles) > 0 {
			m.stateID = state.QuickCommit
			m.textInput.Reset()
			m.textInput.Placeholder = "fix: update implementation"
			m.quickCommitData = views.QuickCommitViewData{
				FileCount: len(localStatus.ChangedFiles),
				IsAmend:   true,
			}
			return m, textinput.Blink
		}
		m.setFlash("Nothing to amend")
		return m, nil

	case "y":
		return m.startMergedAncestorCheck(pendingSync)

	case "d":
		m.confirmData = views.ConfirmViewData{Action: "merge"}
		m.stateID = state.Confirm
		return m, nil

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
		localStatus := git.CheckLocalStatus().(git.LocalStatusMsg)
		if len(localStatus.ChangedFiles) > 0 || localStatus.StagedCount > 0 {
			m.confirmData = views.ConfirmViewData{Action: "reset"}
			m.stateID = state.Confirm
			return m, nil
		}
		m.setFlash("Nothing to reset - working tree is clean")
		return m, nil

	case "z":
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt undo"}
		return m, git.ExecuteUndo()
	}

	return m, nil
}

func (m *model) setFlash(msg string) {
	m.flashMessage = msg
	m.flashExpiry = time.Now().Add(3 * time.Second)
	m.dashboardData.FlashMessage = msg
}

func (m model) collectVercelBranches() []string {
	branches := []string{}
	if m.dashboardData.Branch != "" {
		branches = append(branches, m.dashboardData.Branch)
	}
	for _, item := range m.stackData.Items {
		if item.Name != "" {
			branches = append(branches, item.Name)
		}
	}
	return branches
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
		m.pendingCommitMsg = ""
		m.stateID = state.Dashboard
		return m, nil

	case "enter":
		commitMsg := strings.TrimSpace(m.textInput.Value())
		if commitMsg == "" {
			m.quickCommitData.ErrorMsg = "Message required"
			return m, nil
		}

		m.lastCommitMsg = commitMsg
		m.quickCommitData.InputValue = commitMsg
		m.pendingCommitMsg = commitMsg

		if m.quickCommitData.IsAmend {
			m.justCommitted = false
			return m.startMergedAncestorCheck(pendingIterate)
		}

		m.justCommitted = true
		return m.startMergedAncestorCheck(pendingTurboShip)
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

func (m model) handleVercelKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.focusIndex = 0
		return m, nil

	case "up", "k":
		if m.vercelData.Cursor > 0 {
			m.vercelData.Cursor--
		}
		return m, nil

	case "down", "j":
		if m.vercelData.Cursor < len(m.vercelData.Statuses)-1 {
			m.vercelData.Cursor++
		}
		return m, nil

	case "enter":
		if len(m.vercelData.Statuses) > 0 {
			status := m.vercelData.Statuses[m.vercelData.Cursor]
			if status.ProductionURL != "" {
				return m, vercel.OpenURL(status.ProductionURL)
			}
			if status.PreviewURL != "" {
				return m, vercel.OpenURL(status.PreviewURL)
			}
		}
	}

	return m, nil
}

func (m model) handleStackPanelKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.focusIndex = 0
		return m, nil

	case "up", "k":
		if m.stackData.Cursor > 0 {
			m.stackData.Cursor--
		}
		return m, nil

	case "down", "j":
		if m.stackData.Cursor < len(m.stackData.Items)-1 {
			m.stackData.Cursor++
		}
		return m, nil

	case "enter":
		if len(m.stackData.Items) > 0 {
			target := m.stackData.Items[m.stackData.Cursor].Name
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "gt checkout"}
			return m, git.ExecuteCheckout(target)
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

func (m model) handleMergedAncestorKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.pendingAction = pendingNone
		m.pendingCommitMsg = ""
		m.stateID = state.Dashboard
		return m, ui.RefreshNow()

	case "up", "k":
		if m.mergedAncestorData.Selected > 0 {
			m.mergedAncestorData.Selected--
		}
		return m, nil

	case "down", "j":
		if m.mergedAncestorData.Selected < 2 {
			m.mergedAncestorData.Selected++
		}
		return m, nil

	case "enter":
		selection := m.mergedAncestorData.Selected
		switch selection {
		case 0:
			branches := []string{}
			for _, issue := range m.mergedAncestorData.Issues {
				branches = append(branches, issue.Branch)
			}
			m.pendingCommitMsg = ""
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "resolve merged ancestor"}
			return m, git.ExecuteResolveMergedAncestors(m.mergedAncestorData.TrunkBranch, m.mergedAncestorData.CurrentBranch, branches, m.skipHooks)
		case 1:
			m.pendingAction = pendingNone
			m.pendingCommitMsg = ""
			m.stateID = state.Dashboard
			return m, ui.RefreshNow()
		case 2:
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "unlink PR"}
			m.pendingAction = pendingSubmit
			return m, git.ExecuteTreatAsNew(m.skipHooks)
		}
	}

	return m, nil
}

func (m model) handlePostCommitKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "enter":
		m.justCommitted = false
		m.pendingCommitMsg = ""
		return m.startMergedAncestorCheck(pendingSubmit)

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

// --- View ---

func (m model) View() string {
	ctx := views.NewRenderContext(m.width, m.height)

	// Sync file list to dashboard data
	m.dashboardData.FileList = m.fileList
	m.dashboardData.FileBoxFocused = m.focusIndex == 0
	m.dashboardData.VercelFocused = m.focusIndex == 1
	m.dashboardData.StackFocused = m.focusIndex == 2
	m.dashboardData.VercelSummary = m.vercelData

	switch m.stateID {
	case state.Dashboard:
		return views.RenderDashboard(ctx, m.dashboardData)

	case state.QuickCommit:
		return views.RenderCentered(views.RenderQuickCommit(m.quickCommitData, m.textInput.View()))

	case state.CommitChoice:
		return views.RenderCentered(views.RenderCommitChoice(m.commitChoiceData))

	case state.Confirm:
		return views.RenderCentered(views.RenderConfirm(m.confirmData))

	case state.Stack:
		return views.RenderCentered(views.RenderStack(m.stackData))

	case state.MergedAncestor:
		return views.RenderCentered(views.RenderMergedAncestor(m.mergedAncestorData))

	case state.Update:
		data := views.UpdateViewData{
			CurrentVersion: config.GetCurrentVersion(),
			LatestVersion:  m.latestVersion,
			IsChecking:     m.checkingUpdate,
			IsNewer:        m.latestVersion != "" && config.IsNewerVersion(config.GetCurrentVersion(), m.latestVersion),
			SpinnerView:    m.spinner.View(),
		}
		return views.RenderCentered(views.RenderUpdate(data))

	case state.Running, state.Output, state.PostCommit:
		m.outputData.SpinnerView = m.spinner.View()
		return views.RenderCentered(views.RenderOutput(m.outputData))

	case state.Help:
		return views.RenderCentered(views.RenderHelp())

	default:
		return views.RenderDashboard(ctx, m.dashboardData)
	}
}

func (m *model) startMergedAncestorCheck(action pendingAction) (tea.Model, tea.Cmd) {
	m.pendingAction = action
	if action != pendingTurboShip && action != pendingIterate {
		m.pendingCommitMsg = ""
	}
	m.stateID = state.Running
	m.outputData = views.OutputViewData{IsRunning: true, Command: "Checking stack"}
	return m, git.CheckMergedAncestors()
}

func (m *model) runPendingAction() tea.Cmd {
	defer func() {
		m.pendingAction = pendingNone
		m.pendingCommitMsg = ""
	}()
	switch m.pendingAction {
	case pendingSubmit:
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt submit"}
		return git.ExecuteSubmit(m.skipHooks)
	case pendingSync:
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "gt sync"}
		return git.ExecuteSync(m.skipHooks)
	case pendingIterate:
		if strings.TrimSpace(m.pendingCommitMsg) != "" {
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "iterate"}
			return git.ExecuteIterateWithMsg(m.pendingCommitMsg, m.skipHooks)
		}
		if m.quickCommitData.InputValue != "" {
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "iterate"}
			return git.ExecuteIterateWithMsg(m.quickCommitData.InputValue, m.skipHooks)
		}
		if m.lastCommitMsg != "" {
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "iterate"}
			return git.ExecuteIterateWithMsg(m.lastCommitMsg, m.skipHooks)
		}
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "iterate"}
		return git.ExecuteIterate(m.skipHooks)
	case pendingTurboShip:
		if strings.TrimSpace(m.pendingCommitMsg) != "" {
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "turbo ship"}
			return git.ExecuteTurboShipWithMsg(m.pendingCommitMsg, m.skipHooks)
		}
		if m.quickCommitData.InputValue != "" {
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "turbo ship"}
			return git.ExecuteTurboShipWithMsg(m.quickCommitData.InputValue, m.skipHooks)
		}
		if m.lastCommitMsg != "" {
			m.stateID = state.Running
			m.outputData = views.OutputViewData{IsRunning: true, Command: "turbo ship"}
			return git.ExecuteTurboShipWithMsg(m.lastCommitMsg, m.skipHooks)
		}
		m.stateID = state.Running
		m.outputData = views.OutputViewData{IsRunning: true, Command: "turbo ship"}
		return git.ExecuteTurboShip(m.skipHooks)
	default:
		m.stateID = state.Dashboard
		return ui.RefreshNow()
	}
}

// --- Helpers ---

// filesEqual compares two slices of ChangedFile for equality
func filesEqual(a, b []git.ChangedFile) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Path != b[i].Path || a[i].Status != b[i].Status || a[i].Staged != b[i].Staged {
			return false
		}
	}
	return true
}

// isBranchModifyingCommand returns true if the command modifies branch structure
func isBranchModifyingCommand(cmd string) bool {
	branchCommands := []string{
		"gt create", "gt checkout", "gt merge", "gt sync",
		"gt undo", "turbo ship",
	}
	for _, bc := range branchCommands {
		if cmd == bc || len(cmd) > len(bc) && cmd[:len(bc)] == bc {
			return true
		}
	}
	return false
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
