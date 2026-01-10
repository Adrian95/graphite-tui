package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- View States ---

type state int

const (
	viewDashboard state = iota // Default live dashboard
	viewMenu                   // Full menu
	viewInput
	viewRunning
	viewOutput
	viewWizardType
	viewWizardScope
	viewWizardSummary
	viewWizardPreview // NEW: Preview commit before executing
	viewStack
	viewHelp
	viewUpdate
	viewConfirm    // NEW: Confirmation dialog for destructive actions
	viewStartup    // First-run prompt
	viewPostCommit // Post-commit prompt to share/push
)

// --- Stack Item ---

type stackItem struct {
	name    string
	current bool
	level   int
}

// --- Model ---

type model struct {
	state     state
	prevState state // For returning from output view

	// UI Components
	textInput textinput.Model
	viewport  viewport.Model
	spinner   spinner.Model

	// Dimensions
	width  int
	height int

	// Dashboard state
	branch        string
	stack         []string
	ahead         int
	behind        int
	changedFiles  []ChangedFile
	lastRefresh   time.Time
	suggestion    string
	hasStack      bool
	onMain        bool
	gtInitialized bool

	// Menu state
	items  []menuItem
	cursor int

	// Wizard state
	commitTypes   []commitType
	wizardTypeIdx int
	wizardScope   string
	wizardSummary string

	// Stack map state
	stackItems  []stackItem
	stackCursor int

	// Output state
	output  string
	err     error
	command string

	// Options
	skipHooks bool

	// Validation feedback
	wizardError string

	// Ghost fix mode (for input view)
	isGhostFix bool

	// Post-commit flow
	justCommitted bool
	lastCommitMsg string

	// Confirmation dialog state
	confirmAction  string // "merge", "ghost"
	confirmBranch  string // branch name for ghost fix
	pendingCommand string // command to run on confirm

	// Flash message state
	flashMessage string
	flashExpiry  time.Time

	// Version state
	latestVersion   string
	checkingUpdate  bool
	updateAvailable string

	// File list expansion
	filesExpanded bool

	// Speed box navigation (0=Ship, 1=Iterate, 2=Reset, 3=Undo)
	speedCursor int
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
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0080")) // Hot Pink

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")). // Dark Gray
		Padding(0, 1)

	return model{
		state:       viewDashboard,
		items:       getMenuItems(),
		commitTypes: getCommitTypes(),
		textInput:   ti,
		spinner:     s,
		viewport:    vp,
		skipHooks:   false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		spinner.Tick,
		RefreshNow(),             // Initial status check
		checkForUpdates(),        // Check for updates
		tickEvery(3*time.Second), // Auto-refresh every 3s
	)
}

// --- Update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// --- Keyboard Input ---
	case tea.KeyMsg:
		key := msg.String()

		// Global quit
		if key == "ctrl+c" {
			return m, tea.Quit
		}

		// Handle based on current state
		switch m.state {

		case viewDashboard:
			return m.handleDashboardKeys(key)

		case viewMenu:
			return m.handleMenuKeys(key)

		case viewInput:
			return m.handleInputKeys(key, msg)

		case viewWizardType:
			return m.handleWizardTypeKeys(key)

		case viewWizardScope:
			return m.handleWizardScopeKeys(key, msg)

		case viewWizardSummary:
			return m.handleWizardSummaryKeys(key, msg)

		case viewWizardPreview:
			return m.handleWizardPreviewKeys(key)

		case viewConfirm:
			return m.handleConfirmKeys(key)

		case viewStack:
			return m.handleStackKeys(key)

		case viewHelp:
			if key == "esc" || key == "?" || key == "q" {
				m.state = viewDashboard
			}

		case viewUpdate:
			return m.handleUpdateKeys(key)

		case viewPostCommit:
			return m.handlePostCommitKeys(key)

		case viewOutput:
			if key == "esc" || key == "enter" || key == "q" {
				m.state = viewDashboard
				return m, RefreshNow()
			}
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd

		case viewRunning:
			// Just wait for command to finish
		}

	// --- Status Update ---
	case statusMsg:
		m.branch = msg.branch
		m.stack = msg.stack
		m.ahead = msg.ahead
		m.behind = msg.behind
		m.changedFiles = msg.changedFiles
		m.hasStack = msg.hasStack
		m.onMain = msg.onMain
		m.suggestion = msg.suggestion
		m.gtInitialized = msg.gtInitialized
		m.lastRefresh = time.Now()

		// Check if we should show startup prompt
		if m.state == viewDashboard && !m.hasStack && m.onMain && len(m.changedFiles) == 0 {
			// First time on main with no stack - could show startup
			// But let's not interrupt if they've been using it
		}

	// --- Command Finished ---
	case cmdFinishedMsg:
		m.err = msg.err
		m.command = msg.command

		if msg.err != nil {
			// Combine stderr and stdout for error display
			m.output = msg.stderr
			if m.output == "" {
				m.output = msg.output
			}
			m.state = viewOutput
			m.justCommitted = false // Reset on error
		} else {
			m.output = msg.output
			// If we just committed successfully, show post-commit prompt
			if m.justCommitted {
				m.state = viewPostCommit
			} else {
				m.state = viewOutput
			}
		}

		m.viewport.SetContent(m.output)
		return m, RefreshNow()

	// --- Stack Loaded ---
	case stackLoadedMsg:
		m.stackItems = msg
		for i, item := range m.stackItems {
			if item.current {
				m.stackCursor = i
				break
			}
		}

	// --- Version Check ---
	case versionCheckMsg:
		m.checkingUpdate = false
		if msg.err == nil && isNewerVersion(currentVersion, msg.latestVersion) {
			m.latestVersion = msg.latestVersion
			m.updateAvailable = msg.latestVersion
		} else if msg.err == nil {
			m.latestVersion = msg.latestVersion
		}

	// --- Update Complete ---
	case updateCompleteMsg:
		m.state = viewOutput
		if msg.success {
			m.output = msg.message + "\n\nPlease restart the application to use the new version."
			m.err = nil
		} else {
			m.output = msg.message
			m.err = msg.err
		}
		m.viewport.SetContent(m.output)

	// --- Uninstall Complete ---
	case uninstallCompleteMsg:
		if msg.success {
			fmt.Println("Graphite TUI has been uninstalled. Goodbye!")
			return m, tea.Quit
		}
		m.state = viewOutput
		m.err = msg.err
		m.output = "Uninstall failed: " + msg.err.Error()
		m.viewport.SetContent(m.output)

	// --- Ticker ---
	case tickMsg:
		// Clear expired flash messages
		if m.flashMessage != "" && time.Now().After(m.flashExpiry) {
			m.flashMessage = ""
		}
		// Auto-refresh status (skip during command execution to avoid race conditions)
		if m.state != viewRunning {
			cmds = append(cmds, RefreshNow())
		}
		cmds = append(cmds, tickEvery(3*time.Second))

	// --- Spinner ---
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	// --- Window Resize ---
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 12
	}

	return m, tea.Batch(cmds...)
}

// --- Key Handlers ---

func (m model) handleDashboardKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit

	// Speed box navigation
	case "up", "k":
		if m.speedCursor > 0 {
			m.speedCursor--
		}
		return m, nil

	case "down", "j":
		if m.speedCursor < 3 {
			m.speedCursor++
		}
		return m, nil

	case "enter":
		// Execute selected speed action with smart behavior
		return m.executeSpeedAction()

	case "i": // Init Graphite
		m.state = viewRunning
		return m, executeInteractive("gt init --no-interactive", "", m.skipHooks)

	case "s": // Start
		m.state = viewWizardType
		m.wizardTypeIdx = 0
		m.wizardError = "" // Clear any previous validation errors
		return m, nil

	case "p": // Share (push to GitHub & open PR)
		m.state = viewRunning
		return m, executeSubmit(m.skipHooks)

	case "f": // Iterate (amend + push)
		m.state = viewRunning
		return m, executeIterate(m.skipHooks)

	case "y": // Sync
		m.state = viewRunning
		return m, executeSync(m.skipHooks)

	case "d": // Done - show confirmation first
		m.confirmAction = "merge"
		m.state = viewConfirm
		return m, nil

	case "g": // GPS / Stack Map
		m.state = viewStack
		m.stackCursor = 0
		return m, loadStack()

	case "x": // Rescue (Ghost Fix)
		m.state = viewInput
		m.isGhostFix = true
		m.textInput.Reset()
		m.textInput.Placeholder = "New branch name to rescue your work..."
		return m, textinput.Blink

	case "m": // Menu
		m.state = viewMenu
		m.cursor = 0
		return m, nil

	case "?": // Help
		m.state = viewHelp
		return m, nil

	case "u": // Update
		m.state = viewUpdate
		m.checkingUpdate = true
		return m, checkForUpdates()

	case "h": // Toggle hooks with flash feedback
		m.skipHooks = !m.skipHooks
		if m.skipHooks {
			m.flashMessage = "Hooks OFF - commits will skip pre-commit hooks"
		} else {
			m.flashMessage = "Hooks ON - pre-commit hooks will run"
		}
		m.flashExpiry = time.Now().Add(3 * time.Second)
		return m, nil

	case "r": // Manual refresh
		return m, RefreshNow()

	case "tab": // Toggle file list expansion
		m.filesExpanded = !m.filesExpanded
		return m, nil

	case "R": // Hard Reset - show confirmation first
		if len(m.changedFiles) > 0 {
			m.confirmAction = "reset"
			m.state = viewConfirm
			return m, nil
		}
		m.flashMessage = "Nothing to reset - working tree is clean"
		m.flashExpiry = time.Now().Add(3 * time.Second)
		return m, nil

	case "z": // Undo last Graphite operation
		m.state = viewRunning
		return m, executeUndo()
	}

	return m, nil
}

func (m model) handleMenuKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "m":
		m.state = viewDashboard
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
		if m.skipHooks {
			m.flashMessage = "Hooks OFF - commits will skip pre-commit hooks"
		} else {
			m.flashMessage = "Hooks ON - pre-commit hooks will run"
		}
		m.flashExpiry = time.Now().Add(3 * time.Second)

	case "enter":
		if m.cursor < 0 || m.cursor >= len(m.items) {
			return m, nil
		}
		selected := m.items[m.cursor]

		if selected.title == "Start" {
			m.state = viewWizardType
			m.wizardTypeIdx = 0
			return m, nil
		}

		if selected.title == "Stack Map" {
			m.state = viewStack
			m.stackCursor = 0
			return m, loadStack()
		}

		if selected.title == "Ghost Fix" {
			m.state = viewInput
			m.textInput.Reset()
			m.textInput.Placeholder = "New branch name..."
			return m, textinput.Blink
		}

		if selected.command != "" {
			m.state = viewRunning
			return m, executeInteractive(selected.command, "", m.skipHooks)
		}
	}

	return m, nil
}

func (m model) handleInputKeys(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.state = viewDashboard
		m.isGhostFix = false
		return m, nil

	case "enter":
		inputVal := m.textInput.Value()
		if inputVal == "" {
			return m, nil
		}

		// Handle ghost fix - route through confirmation
		if m.isGhostFix {
			m.isGhostFix = false
			m.confirmAction = "ghost"
			m.confirmBranch = inputVal
			m.state = viewConfirm
			return m, nil
		}

		if m.cursor < 0 || m.cursor >= len(m.items) {
			return m, nil
		}
		selected := m.items[m.cursor]

		if selected.title == "Ghost Fix" {
			m.confirmAction = "ghost"
			m.confirmBranch = inputVal
			m.state = viewConfirm
			return m, nil
		}

		m.state = viewRunning
		return m, executeInteractive(selected.command, inputVal, m.skipHooks)
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) handleWizardTypeKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.state = viewDashboard
		return m, nil

	case "up", "k":
		if m.wizardTypeIdx > 0 {
			m.wizardTypeIdx--
		}

	case "down", "j":
		if m.wizardTypeIdx < len(m.commitTypes)-1 {
			m.wizardTypeIdx++
		}

	case "enter":
		m.state = viewWizardScope
		m.textInput.Reset()
		m.textInput.Placeholder = "auth, ui, api (optional)"
		return m, textinput.Blink
	}

	return m, nil
}

func (m model) handleWizardScopeKeys(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.state = viewDashboard
		return m, nil

	case "enter":
		m.wizardScope = m.textInput.Value()
		m.state = viewWizardSummary
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
		m.state = viewDashboard
		return m, nil

	case "backspace":
		// Back to scope step
		if m.textInput.Value() == "" {
			m.state = viewWizardScope
			m.textInput.Reset()
			m.textInput.SetValue(m.wizardScope)
			return m, textinput.Blink
		}

	case "enter":
		m.wizardSummary = m.textInput.Value()
		if m.wizardSummary == "" {
			m.wizardError = "Summary is required - describe what you changed!"
			return m, nil
		}
		m.wizardError = "" // Clear any previous error

		// Build commit message for preview
		msgStr := m.commitTypes[m.wizardTypeIdx].label
		if m.wizardScope != "" {
			msgStr += fmt.Sprintf("(%s)", m.wizardScope)
		}
		msgStr += fmt.Sprintf(": %s", m.wizardSummary)
		m.lastCommitMsg = msgStr

		// Go to preview instead of executing directly
		m.state = viewWizardPreview
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) handleWizardPreviewKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		// Go back to summary to edit
		m.state = viewWizardSummary
		m.textInput.Reset()
		m.textInput.SetValue(m.wizardSummary)
		return m, textinput.Blink

	case "backspace":
		// Also go back to summary
		m.state = viewWizardSummary
		m.textInput.Reset()
		m.textInput.SetValue(m.wizardSummary)
		return m, textinput.Blink

	case "enter":
		// Execute the commit
		m.justCommitted = true
		m.state = viewRunning
		return m, executeCommit(m.lastCommitMsg, m.skipHooks)
	}

	return m, nil
}

func (m model) handleConfirmKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "enter":
		// Execute the confirmed action
		switch m.confirmAction {
		case "merge":
			m.state = viewRunning
			return m, executeInteractive("gt merge --no-interactive && gt sync --no-interactive", "", m.skipHooks)
		case "ghost":
			m.state = viewRunning
			return m, executeGhostFix(m.confirmBranch, m.skipHooks)
		case "reset":
			m.state = viewRunning
			return m, executeHardReset()
		}
		m.state = viewDashboard
		return m, nil

	case "n", "esc":
		// Cancel - go back to dashboard
		m.confirmAction = ""
		m.confirmBranch = ""
		m.state = viewDashboard
		return m, nil
	}

	return m, nil
}

func (m model) handleStackKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.state = viewDashboard
		return m, nil

	case "up", "k":
		if m.stackCursor > 0 {
			m.stackCursor--
		}

	case "down", "j":
		if m.stackCursor < len(m.stackItems)-1 {
			m.stackCursor++
		}

	case "enter":
		if len(m.stackItems) > 0 {
			target := m.stackItems[m.stackCursor].name
			m.state = viewRunning
			return m, executeCheckout(target)
		}

	case "r":
		return m, loadStack()
	}

	return m, nil
}

func (m model) handleUpdateKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.state = viewDashboard
		return m, nil

	case "u":
		if m.latestVersion != "" && isNewerVersion(currentVersion, m.latestVersion) {
			m.state = viewRunning
			return m, performUpdate()
		}

	case "x":
		// Confirm uninstall
		m.state = viewRunning
		return m, performUninstall()
	}

	return m, nil
}

func (m model) handlePostCommitKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "enter": // Yes, share it!
		m.justCommitted = false
		m.state = viewRunning
		return m, executeSubmit(m.skipHooks)

	case "n", "esc": // No, go back to dashboard
		m.justCommitted = false
		m.state = viewDashboard
		return m, RefreshNow()

	case "f": // Fix - amend the commit
		m.justCommitted = false
		m.state = viewRunning
		return m, executeInteractive("gt modify -a --no-interactive", "", m.skipHooks)
	}

	return m, nil
}

// executeSpeedAction handles the selected speed action with smart context-aware behavior
func (m model) executeSpeedAction() (tea.Model, tea.Cmd) {
	switch m.speedCursor {
	case 0: // Ship
		if len(m.changedFiles) > 0 {
			if m.onMain {
				// On main with changes → create new branch + commit + submit
				m.state = viewRunning
				return m, executeTurboShip(m.skipHooks)
			}
			// On feature branch with changes → amend + submit
			m.state = viewRunning
			return m, executeIterate(m.skipHooks)
		} else if m.ahead > 0 {
			// No changes but have unpushed commits → just submit
			m.state = viewRunning
			return m, executeSubmit(m.skipHooks)
		}
		// Nothing to ship
		m.flashMessage = "Nothing to ship - no changes or unpushed commits"
		m.flashExpiry = time.Now().Add(3 * time.Second)
		return m, nil

	case 1: // Iterate
		if len(m.changedFiles) > 0 {
			m.state = viewRunning
			return m, executeIterate(m.skipHooks)
		}
		m.flashMessage = "Nothing to amend - no changes"
		m.flashExpiry = time.Now().Add(3 * time.Second)
		return m, nil

	case 2: // Reset
		if len(m.changedFiles) > 0 {
			m.confirmAction = "reset"
			m.state = viewConfirm
			return m, nil
		}
		m.flashMessage = "Nothing to reset - working tree is clean"
		m.flashExpiry = time.Now().Add(3 * time.Second)
		return m, nil

	case 3: // Undo
		m.state = viewRunning
		return m, executeUndo()
	}

	return m, nil
}

// executeSelectedMenuItem handles executing the currently selected menu item from dashboard
func (m model) executeSelectedMenuItem() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return m, nil
	}
	selected := m.items[m.cursor]

	// Handle special menu items
	if selected.title == "Start" {
		m.state = viewWizardType
		m.wizardTypeIdx = 0
		return m, nil
	}

	if selected.title == "Stack Map" {
		m.state = viewStack
		m.stackCursor = 0
		return m, loadStack()
	}

	if selected.title == "Ghost Fix" {
		m.state = viewInput
		m.isGhostFix = true
		m.textInput.Reset()
		m.textInput.Placeholder = "New branch name to rescue your work..."
		return m, textinput.Blink
	}

	// Done requires confirmation
	if selected.title == "Done" {
		m.confirmAction = "merge"
		m.state = viewConfirm
		return m, nil
	}

	// Regular commands
	if selected.command != "" {
		m.state = viewRunning
		return m, executeInteractive(selected.command, "", m.skipHooks)
	}

	return m, nil
}

// --- Main ---

func main() {
	// Check if git repo
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		fmt.Println("Not a git repository. Please run from a git project directory.")
		os.Exit(1)
	}

	// Check if graphite is installed
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
