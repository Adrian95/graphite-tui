package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(0).
				Foreground(lipgloss.Color("205")).
				SetString("> ")

	shortcutStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))
)

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
	command     string // The base command, or empty if it requires input/complex logic
	needsInput  bool   // If true, prompts for input (commit msg)
	needsConfirm bool  // If true, might ask for confirmation (not implemented yet, direct execution for speedrun)
	isComplex   bool   // If true, runs a custom function (like Ghost Fix)
}

type model struct {
	state      state
	items      []menuItem
	cursor     int
	textInput  textinput.Model
	viewport   viewport.Model
	spinner    spinner.Model
	output     string
	currentCmd *exec.Cmd
	width      int
	height     int
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "feat: description"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vp := viewport.New(80, 20)

	return model{
		state: viewMenu,
		items: []menuItem{
			{title: "START", desc: "Create branch & commit", command: "gt c -am", needsInput: true},
			{title: "PREVIEW", desc: "Push & Open PR (gt s)", command: "gt s"},
			{title: "FIX", desc: "Amend changes (gt m -a)", command: "gt m -a"}, // User can run PREVIEW after
			{title: "DONE", desc: "Merge & Cleanup (gt merge && gt sync)", command: "gt merge && gt sync", isComplex: true},
			{title: "SYNC", desc: "Update local (gt sync)", command: "gt sync"},
			{title: "FOLD", desc: "Squash stack (gt fold)", command: "gt fold"},
			{title: "UNDO", desc: "Undo last command (gt undo)", command: "gt undo"},
			{title: "GHOST FIX", desc: "Fix ghost branch", command: "", isComplex: true},
			{title: "STATUS", desc: "Where am I? (gt ls)", command: "gt ls"},
		},
		textInput: ti,
		spinner:   s,
		viewport:  vp,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, spinner.Tick)
}

type cmdFinishedMsg struct {
	output string
	err    error
}

func executeCommand(commandStr string, inputArg string) tea.Cmd {
	return func() tea.Msg {
		// Handle complex commands or chained commands
		var cmd *exec.Cmd
		
		// Simple shell execution wrapper to handle && and arguments
		fullCmd := commandStr
		if inputArg != "" {
			// specifically for gt c -am, we append the quoted message
			fullCmd = fmt.Sprintf(`%s "%s"`, commandStr, inputArg)
		}

		// Use sh -c to allow chaining (&&) and proper argument parsing
		cmd = exec.Command("sh", "-c", fullCmd)
		
		out, err := cmd.CombinedOutput()
		if err != nil {
			return cmdFinishedMsg{output: string(out), err: err}
		}
		return cmdFinishedMsg{output: string(out), err: nil}
	}
}

func executeGhostFix(branchName string) tea.Cmd {
	return func() tea.Msg {
		// 1. gt c -am "feat: correct branch name"
		// 2. gt rebase main
		// 3. gt sync
		// Note: This is complex. For simplicity, we'll try to chain them in shell.
		
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
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state != viewInput {
				return m, tea.Quit
			}
		case "esc":
			if m.state == viewInput || m.state == viewOutput {
				m.state = viewMenu
				return m, nil
			}
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
			case "enter":
				selected := m.items[m.cursor]
				if selected.needsInput {
					m.state = viewInput
					m.textInput.Reset()
					if selected.title == "GHOST FIX" {
						m.textInput.Placeholder = "feat: correct branch name"
					} else {
						m.textInput.Placeholder = "feat: new thing"
					}
					return m, textinput.Blink
				}
				m.state = viewRunning
				return m, executeCommand(selected.command, "")
			}

		case viewInput:
			switch msg.String() {
			case "enter":
				inputVal := m.textInput.Value()
				selected := m.items[m.cursor]
				m.state = viewRunning
				
				if selected.title == "GHOST FIX" {
					return m, executeGhostFix(inputVal)
				}
				return m, executeCommand(selected.command, inputVal)
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		
		case viewOutput:
			// Allow scrolling in viewport?
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case cmdFinishedMsg:
		m.state = viewOutput
		outputContent := ""
		if msg.err != nil {
			outputContent = fmt.Sprintf("Error: %v\n\nOutput:\n%s", msg.err, msg.output)
		} else {
			outputContent = fmt.Sprintf("Success!\n\n%s", msg.output)
		}
		m.output = outputContent
		m.viewport.SetContent(outputContent)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 10 // Leave room for header/footer
	}

	return m, nil
}

func (m model) View() string {
	if m.state == viewRunning {
		return fmt.Sprintf("\n %s Running %s...\n\n", m.spinner.View(), m.items[m.cursor].title)
	}

	if m.state == viewOutput {
		return fmt.Sprintf(
			"%s\n\n%s\n\n(Press q to quit, esc for menu)",
			titleStyle.Render("Command Result"),
			m.viewport.View(),
		)
	}

	if m.state == viewInput {
		return fmt.Sprintf(
			"Enter commit message/branch name:\n\n%s\n\n(Esc to cancel)",
			m.textInput.View(),
		)
	}

	// Menu View
	s := titleStyle.Render("Graphite Speedrun TUI") + "\n\n"

	for i, item := range m.items {
		cursor := "  "
		lineStyle := itemStyle
		if m.cursor == i {
			cursor = "> "
			lineStyle = selectedItemStyle
		}

		s += fmt.Sprintf("%s%s\n  %s\n\n", 
			lineStyle.Render(cursor+item.title), 
			commandStyle.Render(" "+item.command),
			shortcutStyle.Render(item.desc),
		)
	}

	s += "\n(j/k to navigate • enter to select • q to quit)\n"
	return s
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
