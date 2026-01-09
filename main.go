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
	viewDashboard state = iota // NEW: Default live dashboard
	viewMenu                   // Full menu
	viewInput
	viewRunning
	viewOutput
	viewWizardType
	viewWizardScope
	viewWizardSummary
	viewStack
	viewHelp
	viewUpdate
	viewStartup // NEW: First-run prompt
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

	// Version state
	latestVersion   string
	checkingUpdate  bool
	updateAvailable string
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
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBlue))

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorGray)).
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
		RefreshNow(),                      // Initial status check
		checkForUpdates(),                 // Check for updates
		tickEvery(3*time.Second),          // Auto-refresh every 3s
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

		case viewStack:
			return m.handleStackKeys(key)

		case viewHelp:
			if key == "esc" || key == "?" || key == "q" {
				m.state = viewDashboard
			}

		case viewUpdate:
			return m.handleUpdateKeys(key)

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
		m.state = viewOutput
		m.err = msg.err
		m.command = msg.command

		if msg.err != nil {
			// Combine stderr and stdout for error display
			m.output = msg.stderr
			if m.output == "" {
				m.output = msg.output
			}
		} else {
			m.output = msg.output
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
		// Auto-refresh status
		cmds = append(cmds, RefreshNow())
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
	case "q", "esc":
		return m, tea.Quit

	case "i": // Init Graphite
		m.state = viewRunning
		return m, executeInteractive("gt init --no-interactive", "", m.skipHooks)

	case "s": // Start
		m.state = viewWizardType
		m.wizardTypeIdx = 0
		m.wizardError = "" // Clear any previous validation errors
		return m, nil

	case "p": // Preview
		m.state = viewRunning
		return m, executeInteractive("gt submit --no-interactive", "", m.skipHooks)

	case "f": // Fix
		m.state = viewRunning
		return m, executeInteractive("gt modify -a --no-interactive --no-edit", "", m.skipHooks)

	case "y": // Sync
		m.state = viewRunning
		return m, executeInteractive("gt sync --no-interactive", "", m.skipHooks)

	case "d": // Done
		m.state = viewRunning
		return m, executeInteractive("gt merge --no-interactive && gt sync --no-interactive", "", m.skipHooks)

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

	case "h": // Toggle hooks
		m.skipHooks = !m.skipHooks
		return m, nil

	case "r": // Manual refresh
		return m, RefreshNow()
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

		// Handle ghost fix from dashboard shortcut
		if m.isGhostFix {
			m.state = viewRunning
			m.isGhostFix = false
			return m, executeGhostFix(inputVal, m.skipHooks)
		}

		if m.cursor < 0 || m.cursor >= len(m.items) {
			return m, nil
		}
		selected := m.items[m.cursor]
		m.state = viewRunning

		if selected.title == "Ghost Fix" {
			return m, executeGhostFix(inputVal, m.skipHooks)
		}
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

	case "enter":
		m.wizardSummary = m.textInput.Value()
		if m.wizardSummary == "" {
			m.wizardError = "Summary is required - describe what you changed!"
			return m, nil
		}
		m.wizardError = "" // Clear any previous error

		// Build commit message
		msgStr := m.commitTypes[m.wizardTypeIdx].label
		if m.wizardScope != "" {
			msgStr += fmt.Sprintf("(%s)", m.wizardScope)
		}
		msgStr += fmt.Sprintf(": %s", m.wizardSummary)

		m.state = viewRunning
		return m, executeCommit(msgStr, m.skipHooks)
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
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

// --- View ---

func (m model) View() string {
	header := renderHeader(currentVersion, m.lastRefresh, m.updateAvailable)

	var content string

	switch m.state {

	case viewDashboard:
		// Branch info panel
		branchPanel := renderBranchInfo(m.branch, m.stack, m.ahead, m.behind)

		// Changed files panel
		filesPanel := renderChangedFiles(m.changedFiles)

		// Shortcuts bar
		shortcuts := renderShortcutsBarWithInit(m.skipHooks, m.gtInitialized)

		// Co-pilot / guide
		guide := ""
		if m.cursor >= 0 && m.cursor < len(m.items) {
			guide = m.items[m.cursor].guide
		}
		copilot := renderCopilot(m.suggestion, guide)

		content = lipgloss.JoinVertical(lipgloss.Left,
			branchPanel,
			"",
			filesPanel,
			"",
			shortcuts,
			"",
			copilot,
		)

	case viewMenu:
		var listItems []string
		for i, item := range m.items {
			listItems = append(listItems, renderMenuItem(item.title, item.desc, m.cursor == i))
		}

		listContent := lipgloss.JoinVertical(lipgloss.Left, listItems...)

		hooksStatus := "Hooks: ON"
		hooksStyle := toggleOffStyle
		if m.skipHooks {
			hooksStatus = "Hooks: SKIPPED"
			hooksStyle = toggleOnStyle
		}

		settingsBar := lipgloss.JoinHorizontal(lipgloss.Left,
			subtitleStyle.Render("Press 'h' to toggle: "),
			hooksStyle.Render(hooksStatus),
		)

		guide := ""
		if m.cursor >= 0 && m.cursor < len(m.items) {
			guide = m.items[m.cursor].guide
		}
		guideBox := renderCopilot(m.suggestion, guide)

		content = lipgloss.JoinVertical(lipgloss.Left,
			panelTitleStyle.Render("Select an action:"),
			"",
			listContent,
			"",
			settingsBar,
			"",
			guideBox,
			"",
			subtitleStyle.Render("Press Esc to go back"),
		)

	case viewWizardType:
		content = renderWizardTypes(m.commitTypes, m.wizardTypeIdx)

	case viewWizardScope:
		content = lipgloss.JoinVertical(lipgloss.Left,
			inputTitleStyle.Render("Step 2: Enter Scope (Optional)"),
			subtitleStyle.Render("e.g. auth, ui, api"),
			inputBoxStyle.Render(m.textInput.View()),
		)

	case viewWizardSummary:
		summaryContent := []string{
			inputTitleStyle.Render("Step 3: Enter Summary"),
			subtitleStyle.Render("e.g. add new login button"),
			inputBoxStyle.Render(m.textInput.View()),
		}
		if m.wizardError != "" {
			summaryContent = append(summaryContent, errorStyle.Render("⚠ "+m.wizardError))
		}
		content = lipgloss.JoinVertical(lipgloss.Left, summaryContent...)

	case viewStack:
		content = renderStackMap(m.stackItems, m.stackCursor)

	case viewInput:
		var title string
		if m.isGhostFix {
			title = "Rescue (Ghost Fix)"
		} else if m.cursor >= 0 && m.cursor < len(m.items) {
			title = m.items[m.cursor].title
		}
		content = lipgloss.JoinVertical(lipgloss.Left,
			inputTitleStyle.Render(fmt.Sprintf("%s > Enter branch name", title)),
			subtitleStyle.Render("This will save your work to a fresh branch"),
			inputBoxStyle.Render(m.textInput.View()),
			"",
			subtitleStyle.Render("Press Enter to confirm • Esc to cancel"),
		)

	case viewRunning:
		content = lipgloss.NewStyle().Margin(2, 0).Render(
			fmt.Sprintf("%s Running...", m.spinner.View()),
		)

	case viewOutput:
		if m.err != nil {
			content = renderError(m.err, m.output, m.command)
		} else {
			content = renderSuccess(m.output, m.command)
		}

	case viewHelp:
		content = renderHelp()

	case viewUpdate:
		content = renderUpdateView(currentVersion, m.latestVersion, m.checkingUpdate)
	}

	// Handle empty branch on first load
	if m.branch == "" && m.state == viewDashboard {
		content = lipgloss.JoinVertical(lipgloss.Left,
			panelStyle.Width(50).Render(
				subtitleStyle.Render("Loading git status..."),
			),
			"",
			renderShortcutsBar(m.skipHooks),
		)
	}

	return docStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			content,
		),
	)
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
