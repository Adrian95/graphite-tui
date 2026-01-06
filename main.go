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
		version: "v1.1.0",
		items: []menuItem{
			{
				title: "Start", 
				desc: "Create branch & commit", 
				guide: "Use this when you start a new task. It creates a new branch and commits your initial work.",
				command: "gt c -am", 
				needsInput: true,
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
				guide: "Made a mistake or received feedback? This updates the current commit without creating a new one.",
				command: "gt m -a",
			},
			{
				title: "Sync", 
				desc: "Update local stack", 
				guide: "Pulls the latest changes from main and updates your branch stack. Run this often!",
				command: "gt sync",
			},
			{
				title: "Done", 
				desc: "Merge & Cleanup", 
				guide: "Merges the current stack and deletes local branches. Use only when your PR is approved and merged.",
				command: "gt merge && gt sync", 
				isComplex: true,
			},
			{
				title: "Fold", 
				desc: "Squash stack", 
				guide: "Combines multiple commits/branches into one. Useful if you accidentally branched off a branch.",
				command: "gt fold",
			},
			{
				title: "Ghost Fix", 
				desc: "Rescue ghost branch", 
				guide: "Fixes the 'working on a merged branch' issue. Moves your work to a new clean branch.",
				command: "", 
				isComplex: true, 
				needsInput: true,
			},
			{
				title: "Status", 
				desc: "View stack", 
				guide: "Shows where you are in the dependency stack. Think of it as a map of your work.",
				command: "gt ls",
			},
			{
				title: "Update Tool",
				desc: "Update Graphite TUI",
				guide: "Updates this tool to the latest version from GitHub.",
				command: "go install github.com/adrian/graphite-tui@latest",
			},
		},
		textInput: ti,
		spinner:   s,
		viewport:  vp,
		skipHooks: false, // Default to running hooks
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, spinner.Tick)
}

// --- Logic ---

type cmdFinishedMsg struct {
	output string
	err    error
}

func executeCommand(commandStr string, inputArg string, skipHooks bool) tea.Cmd {
	return func() tea.Msg {
		fullCmd := commandStr
		
		// Append hook skipping if enabled and command supports it
		// We target 'gt c' (commit) and 'gt s' (submit/push) mostly, but git commit is the main one.
		// 'gt c' passes flags to git commit.
		// 'gt m' (modify) also passes flags.
		if skipHooks {
			if strings.Contains(commandStr, "gt c") || strings.Contains(commandStr, "gt m") {
				fullCmd += " --no-verify"
			}
			// For gt s (submit), it might run build steps? 
			// Generally --no-verify is for git commit hooks.
		}

		if inputArg != "" {
			fullCmd = fmt.Sprintf(`%s "%s"`, fullCmd, inputArg)
		}
		
		// Use sh -c to allow chaining
		cmd := exec.Command("sh", "-c", fullCmd)
		out, err := cmd.CombinedOutput()
		return cmdFinishedMsg{output: string(out), err: err}
	}
}

func executeGhostFix(branchName string) tea.Cmd {
	return func() tea.Msg {
		script := fmt.Sprintf(`gt c -am "%s" && gt rebase main && gt sync`, branchName)
		cmd := exec.Command("sh", "-c", script)
		out, err := cmd.CombinedOutput()
		return cmdFinishedMsg{output: string(out), err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global Quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		
		// Back Navigation
		if msg.String() == "esc" || (msg.String() == "q" && m.state != viewInput) {
			if m.state == viewMenu {
				return m, tea.Quit
			}
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
			
			// Toggle Hooks
			case "h":
				m.skipHooks = !m.skipHooks

			case "enter":
				selected := m.items[m.cursor]
				
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
		}
		// Clean up ANSI codes if needed, though viewport handles most
		m.viewport.SetContent(outputContent)
		return m, nil

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
			
			// Visual selection logic
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
		
		// Status / Settings Bar
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

		// Guide Box
		selectedItem := m.items[m.cursor]
		guideBox := guideStyle.Render(
			lipgloss.NewStyle().Foreground(lipgloss.Color(colorYellow)).Bold(true).Render("💡 Tip: ") + 
			selectedItem.guide,
		)

		content = lipgloss.JoinVertical(lipgloss.Left, 
			listTitleStyle.Render("Select an action:"),
			listContent,
			"\n",
			settingsBar,
			guideBox,
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
