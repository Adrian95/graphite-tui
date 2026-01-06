package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Theme & Styles ---

const (
	colorBlack     = "#000000"
	colorWhite     = "#EDEDED"
	colorGray      = "#666666"
	colorLightGray = "#999999"
	colorBlue      = "#0070F3" // Vercel Blue
	colorPurple    = "#7928CA" // Vercel Purple
	colorRed       = "#FF4500" // Error Red
	colorGreen     = "#00C64F" // Success Green
	colorYellow    = "#F5A623" // Warning/Guide
)

var (
	// Layout
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	// Header
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWhite)).
			Bold(true).
			MarginBottom(1)

	subHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			MarginLeft(1)
	
	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			MarginLeft(1).
			Italic(true)

	// List
	listTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorWhite)).
				Bold(true).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color(colorBlue)).
				PaddingLeft(1)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			Italic(true).
			MarginLeft(2)
	
	guideStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorYellow)).
			MarginTop(1).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorGray))

	// Input
	inputTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			Bold(true).
			MarginBottom(1)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorGray)).
			Padding(0, 1).
			Width(60)

	// Output
	outputHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWhite)).
			Background(lipgloss.Color(colorBlack)).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color(colorGray)).
			Padding(0, 1).
			Width(80)

	statusSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	statusErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorRed))
	
	// Toggles
	toggleOnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	toggleOffStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGray))
)

// --- Model Definitions ---

type state int

const (
	viewMenu state = iota
	viewInput
	viewRunning
	viewOutput
	viewWizardType
	viewWizardScope
	viewWizardSummary
	viewStack
)

type menuItem struct {
	title       string
	desc        string
	guide       string // Beginner guidance text
	command     string
	needsInput  bool
	isComplex   bool
	confirm     bool
}

type commitType struct {
	label string
	desc  string
}

type stackItem struct {
	name    string
	current bool
	level   int // for indentation
}

type model struct {
	state      state
	items      []menuItem
	cursor     int
	textInput  textinput.Model
	viewport   viewport.Model
	spinner    spinner.Model
	output     string
	width      int
	height     int
	err        error
	
	// Options
	skipHooks bool
	version   string
	suggestion string

	// Wizard Data
	commitTypes    []commitType
	wizardTypeIdx  int
	wizardScope    string
	wizardSummary  string
	
	// Stack Data
	stackItems []stackItem
	stackCursor int
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Type your commit message..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50
	ti.Prompt = "" // clean look

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBlue))

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorGray)).
		Padding(0, 1)

	return model{
		state: viewMenu,
		version: "v1.2.0",
		items: []menuItem{
			{
				title: "Start", 
				desc: "Create branch & commit (Wizard)", 
				guide: "Starts a new task. We'll guide you through creating a perfect commit message.",
				command: "gt c -am", 
				isComplex: true, // Switched to complex for Wizard
			},
			{
				title: "Preview", 
				desc: "Push & Open PR", 
				guide: "Uploads your changes to GitHub and opens a Pull Request for review.",
				command: "gt s",
			},
			{
				title: "Fix", 
				desc: "Amend changes", 
				guide: "Made a mistake? This updates the current commit without creating a new one.",
				command: "gt m -a",
			},
			{
				title: "Sync", 
				desc: "Update local stack", 
				guide: "Pulls latest changes. Run this often to stay up to date!",
				command: "gt sync",
			},
			{
				title: "Done", 
				desc: "Merge & Cleanup", 
				guide: "Merges the current stack and deletes local branches.",
				command: "gt merge && gt sync", 
				isComplex: true,
			},
			{
				title: "Fold", 
				desc: "Squash stack", 
				guide: "Combines multiple commits/branches into one.",
				command: "gt fold",
			},
			{
				title: "Ghost Fix", 
				desc: "Rescue ghost branch", 
				guide: "Fixes the 'working on a merged branch' issue.",
				command: "", 
				isComplex: true, 
				needsInput: true,
			},
			{
				title: "Stack Map", 
				desc: "Visual Dashboard", 
				guide: "See your branch stack visually and jump between branches.",
				command: "gt ls", // We'll hijack this
				isComplex: true,
			},
			{
				title: "Update Tool",
				desc: "Update Graphite TUI",
				guide: "Updates this tool to the latest version from GitHub.",
				command: "go install github.com/Adrian95/graphite-tui@latest",
			},
		},
		commitTypes: []commitType{
			{"feat", "A new feature"},
			{"fix", "A bug fix"},
			{"docs", "Documentation only changes"},
			{"style", "Changes that do not affect the meaning of the code"},
			{"refactor", "A code change that neither fixes a bug nor adds a feature"},
			{"perf", "A code change that improves performance"},
			{"test", "Adding missing tests or correcting existing tests"},
			{"chore", "Changes to the build process or auxiliary tools"},
		},
		textInput: ti,
		spinner:   s,
		viewport:  vp,
		skipHooks: false, 
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, spinner.Tick, checkGitStatus)
}

// --- Logic ---

type cmdFinishedMsg struct {
	output  string
	err     error
	command string
}

type statusMsg string
type stackLoadedMsg []stackItem

func checkGitStatus() tea.Msg {
	// Check for uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		return statusMsg("suggestion: You have uncommitted changes. Try 'Start' (new) or 'Fix' (amend).")
	}
	
	// Check if ahead of remote (simple check)
	cmd = exec.Command("git", "status", "-sb")
	out, err = cmd.Output()
	if err == nil && strings.Contains(string(out), "ahead") {
		return statusMsg("suggestion: Your branch is ahead. Try 'Preview' to push.")
	}
	
	return statusMsg("")
}

func loadStack() tea.Cmd {
	return func() tea.Msg {
		// Mockable command execution for gt ls
		cmd := exec.Command("gt", "ls")
		out, err := cmd.Output()
		if err != nil {
			// fallback if gt not installed or fails
			return stackLoadedMsg{} 
		}
		
		var items []stackItem
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			
			// Simple parser
			// * branch-name
			//   branch-name
			current := strings.Contains(line, "*")
			cleanName := strings.TrimSpace(line)
			cleanName = strings.TrimPrefix(cleanName, "* ")
			
			// Try to guess level by indentation? gt ls usually uses tree chars
			// For now, flat list is better than nothing, but let's try to detect indent
			indent := len(line) - len(strings.TrimLeft(line, " "))
			
			items = append(items, stackItem{
				name:    cleanName,
				current: current,
				level:   indent / 2, // approximation
			})
		}
		return stackLoadedMsg(items)
	}
}

func executeCommand(commandStr string, inputArg string, skipHooks bool) tea.Cmd {
	return func() tea.Msg {
		fullCmd := commandStr
		if skipHooks {
			if strings.Contains(commandStr, "gt c") || strings.Contains(commandStr, "gt m") {
				fullCmd += " --no-verify"
			}
		}

		if inputArg != "" {
			fullCmd = fmt.Sprintf(`%s "%s"`, fullCmd, inputArg)
		}
		
		cmd := exec.Command("sh", "-c", fullCmd)
		out, err := cmd.CombinedOutput()
		return cmdFinishedMsg{output: string(out), err: err, command: commandStr}
	}
}

func executeCheckout(branch string) tea.Cmd {
	return executeCommand("gt checkout "+branch, "", false)
}

func executeGhostFix(branchName string) tea.Cmd {
	return func() tea.Msg {
		script := fmt.Sprintf(`gt c -am "%s" && gt rebase main && gt sync`, branchName)
		cmd := exec.Command("sh", "-c", script)
		out, err := cmd.CombinedOutput()
		return cmdFinishedMsg{output: string(out), err: err, command: "GHOST FIX"}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		
		if msg.String() == "esc" || (msg.String() == "q" && m.state == viewMenu) {
			return m, tea.Quit
		}
		
		// Back navigation
		if msg.String() == "esc" {
			m.state = viewMenu
			return m, nil
		}

		switch m.state {
		case viewMenu:
			switch msg.String() {
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
				selected := m.items[m.cursor]
				
				// Handle Special Features
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

				if selected.needsInput {
					m.state = viewInput
					m.textInput.Reset()
					if selected.title == "Ghost Fix" {
						m.textInput.Placeholder = "New branch name..."
					} else {
						m.textInput.Placeholder = "Commit message..."
					}
					return m, textinput.Blink
				}
				
				m.state = viewRunning
				return m, executeCommand(selected.command, "", m.skipHooks)
			}

		case viewInput:
			switch msg.String() {
			case "enter":
				inputVal := m.textInput.Value()
				selected := m.items[m.cursor]
				m.state = viewRunning
				
				if selected.title == "Ghost Fix" {
					return m, executeGhostFix(inputVal)
				}
				return m, executeCommand(selected.command, inputVal, m.skipHooks)
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		
		// --- Wizard Flow ---
		case viewWizardType:
			switch msg.String() {
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

		case viewWizardScope:
			switch msg.String() {
			case "enter":
				m.wizardScope = m.textInput.Value()
				m.state = viewWizardSummary
				m.textInput.Reset()
				m.textInput.Placeholder = "Add new login button"
				return m, textinput.Blink
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case viewWizardSummary:
			switch msg.String() {
			case "enter":
				m.wizardSummary = m.textInput.Value()
				if m.wizardSummary == "" {
					return m, nil // prevent empty commits
				}
				
				// Construct commit message
				// type(scope): summary OR type: summary
				msgStr := m.commitTypes[m.wizardTypeIdx].label
				if m.wizardScope != "" {
					msgStr += fmt.Sprintf("(%s)", m.wizardScope)
				}
				msgStr += fmt.Sprintf(": %s", m.wizardSummary)

				m.state = viewRunning
				return m, executeCommand("gt c -am", msgStr, m.skipHooks)
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		// --- Stack Flow ---
		case viewStack:
			switch msg.String() {
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

		case viewOutput:
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case cmdFinishedMsg:
		m.state = viewOutput
		m.err = msg.err
		outputContent := msg.output
		if msg.err != nil {
			outputContent = fmt.Sprintf("Error: %v\n\n%s", msg.err, msg.output)
		} else {
			if strings.TrimSpace(outputContent) == "" {
				outputContent = "Command executed successfully."
			}
			if strings.Contains(msg.command, "go install") {
				outputContent += "\n\n✨ Update complete! Please restart the tool to apply changes."
			}
		}
		m.viewport.SetContent(outputContent)
		
		// Re-check status after any command
		return m, checkGitStatus

	case statusMsg:
		m.suggestion = string(msg)

	case stackLoadedMsg:
		m.stackItems = msg
		// Try to find current branch to set cursor
		for i, item := range m.stackItems {
			if item.current {
				m.stackCursor = i
				break
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 12 
	}

	return m, nil
}

// --- View ---

func (m model) View() string {
	// Header
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Render("Graphite"),
		subHeaderStyle.Render("/ Speedrun"),
		versionStyle.Render(m.version),
	)

	var content string

	switch m.state {
	case viewMenu:
		var listItems []string
		for i, item := range m.items {
			title := item.title
			desc := item.desc
			
			if m.cursor == i {
				line := lipgloss.JoinHorizontal(lipgloss.Left,
					selectedItemStyle.Render(title),
					descStyle.Render(desc),
				)
				listItems = append(listItems, line)
			} else {
				line := lipgloss.JoinHorizontal(lipgloss.Left,
					itemStyle.Render(title),
					descStyle.Render(desc),
				)
				listItems = append(listItems, line)
			}
		}
		
		listContent := lipgloss.JoinVertical(lipgloss.Left, listItems...)
		
		hooksStatus := "Hooks: ON"
		hooksStyle := toggleOffStyle
		if m.skipHooks {
			hooksStatus = "Hooks: SKIPPED"
			hooksStyle = toggleOnStyle
		}
		
		settingsBar := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorGray)).Render("Press 'h' to toggle git hooks: "),
			hooksStyle.Render(hooksStatus),
		)

		// Co-Pilot / Guide Logic
		guideText := m.items[m.cursor].guide
		guideTitle := "💡 Tip: "
		guideColor := colorYellow
		
		// Override with Smart Suggestion if available
		if m.suggestion != "" && strings.HasPrefix(m.suggestion, "suggestion:") {
			guideText = strings.TrimPrefix(m.suggestion, "suggestion: ")
			guideTitle = "🤖 Co-Pilot: "
			guideColor = colorBlue
		}

		guideBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color(guideColor)).
			MarginTop(1).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorGray)).
			Render(lipgloss.NewStyle().Bold(true).Render(guideTitle) + guideText)

		content = lipgloss.JoinVertical(lipgloss.Left, 
			listTitleStyle.Render("Select an action:"),
			listContent,
			"\n",
			settingsBar,
			guideBox,
		)

	case viewWizardType:
		var types []string
		for i, t := range m.commitTypes {
			prefix := "  "
			style := itemStyle
			if i == m.wizardTypeIdx {
				prefix = "> "
				style = selectedItemStyle
			}
			types = append(types, style.Render(fmt.Sprintf("%s%s: %s", prefix, t.label, t.desc)))
		}
		content = lipgloss.JoinVertical(lipgloss.Left,
			inputTitleStyle.Render("Step 1: Select Commit Type"),
			lipgloss.JoinVertical(lipgloss.Left, types...),
		)

	case viewWizardScope:
		content = lipgloss.JoinVertical(lipgloss.Left,
			inputTitleStyle.Render("Step 2: Enter Scope (Optional)"),
			subHeaderStyle.Render("e.g. auth, ui, api"),
			inputBoxStyle.Render(m.textInput.View()),
		)

	case viewWizardSummary:
		content = lipgloss.JoinVertical(lipgloss.Left,
			inputTitleStyle.Render("Step 3: Enter Summary"),
			subHeaderStyle.Render("e.g. add new login button"),
			inputBoxStyle.Render(m.textInput.View()),
		)

	case viewStack:
		var items []string
		if len(m.stackItems) == 0 {
			items = append(items, itemStyle.Render("No stack found or empty."))
		} else {
			for i, s := range m.stackItems {
				prefix := "  "
				// Indentation
				indent := strings.Repeat("  ", s.level)
				marker := " "
				style := itemStyle
				
				if s.current {
					marker = "*"
				}
				
				if i == m.stackCursor {
					prefix = "> "
					style = selectedItemStyle
				}

				label := fmt.Sprintf("%s%s%s %s", prefix, indent, marker, s.name)
				items = append(items, style.Render(label))
			}
		}
		content = lipgloss.JoinVertical(lipgloss.Left,
			inputTitleStyle.Render("Stack Map (GPS)"),
			lipgloss.JoinVertical(lipgloss.Left, items...),
			"\n"+subHeaderStyle.Render("Enter to checkout • r to refresh • Esc back"),
		)

	case viewInput:
		title := m.items[m.cursor].title
		content = lipgloss.JoinVertical(lipgloss.Left,
			inputTitleStyle.Render(fmt.Sprintf("%s > Input", title)),
			inputBoxStyle.Render(m.textInput.View()),
			"\n"+subHeaderStyle.Render("Press Enter to confirm • Esc to cancel"),
			"\n"+lipgloss.NewStyle().Foreground(lipgloss.Color(colorYellow)).Render("Note: Git hooks will be skipped if enabled."),
		)

	case viewRunning:
		content = lipgloss.NewStyle().Margin(2, 0).Render(
			fmt.Sprintf("%s Running %s...", m.spinner.View(), m.items[m.cursor].title),
		)

	case viewOutput:
		status := statusSuccessStyle.Render("● Success")
		if m.err != nil {
			status = statusErrorStyle.Render("● Error")
		}
		
		topBar := lipgloss.JoinHorizontal(lipgloss.Left,
			status,
			subHeaderStyle.Render(" • Press q to close"),
		)

		content = lipgloss.JoinVertical(lipgloss.Left,
			topBar,
			m.viewport.View(),
		)
	}

	return docStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			header,
			"\n",
			content,
		),
	)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
