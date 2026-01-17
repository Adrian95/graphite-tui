package vercel

import (
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// CopyToClipboard copies a string to clipboard
func CopyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "linux":
			cmd = exec.Command("xclip", "-selection", "clipboard")
		case "windows":
			cmd = exec.Command("cmd", "/c", "clip")
		default:
			return ActionMsg{Err: nil, Message: "Clipboard not supported"}
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return ActionMsg{Err: err}
		}
		if err := cmd.Start(); err != nil {
			return ActionMsg{Err: err}
		}
		_, _ = stdin.Write([]byte(text))
		_ = stdin.Close()
		_ = cmd.Wait()

		return ActionMsg{Message: "Copied to clipboard"}
	}
}

// OpenURL opens a URL in the default browser
func OpenURL(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", url)
		default:
			return ActionMsg{Err: nil, Message: "Open not supported"}
		}
		if err := cmd.Start(); err != nil {
			return ActionMsg{Err: err}
		}
		return ActionMsg{Message: "Opened in browser"}
	}
}
