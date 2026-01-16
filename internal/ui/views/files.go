package views

import (
	"github.com/Adrian95/graphite-tui/internal/git"
	"github.com/Adrian95/graphite-tui/internal/ui"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// FileItem wraps git.ChangedFile to implement list.Item
type FileItem struct {
	File git.ChangedFile
}

func (i FileItem) FilterValue() string { return i.File.Path }

func (i FileItem) Title() string {
	icon := "•"
	switch i.File.Status {
	case "M", "MM":
		icon = "M"
	case "A", "AM":
		icon = "+"
	case "D":
		icon = "-"
	case "R":
		icon = "R"
	case "??":
		icon = "?"
	}
	return icon + " " + i.File.Path
}

func (i FileItem) Description() string {
	switch i.File.Status {
	case "M", "MM":
		return "Modified"
	case "A", "AM":
		return "Added"
	case "D":
		return "Deleted"
	case "R":
		return "Renamed"
	case "??":
		return "Untracked"
	default:
		return "Unknown"
	}
}

func NewFileList() list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color(ui.ColorAccent)).BorderLeftForeground(lipgloss.Color(ui.ColorAccent))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color(ui.ColorFg)).BorderLeftForeground(lipgloss.Color(ui.ColorAccent))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Changed Files"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // Disable filtering for now to keep it simple, or enable if requested
	l.Styles.Title = ui.BoxTitleStyle
	l.DisableQuitKeybindings()

	return l
}

func FilesToItems(files []git.ChangedFile) []list.Item {
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = FileItem{File: f}
	}
	return items
}
