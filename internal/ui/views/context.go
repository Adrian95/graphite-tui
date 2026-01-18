package views

import (
	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// RenderContext provides rendering parameters
type RenderContext struct {
	Width  int
	Height int
}

// NewRenderContext creates a render context with defaults
func NewRenderContext(width, height int) RenderContext {
	if width == 0 {
		width = 100
	}
	if height == 0 {
		height = 30
	}
	return RenderContext{Width: width, Height: height}
}

// SidebarWidth returns the standard sidebar width
func (c RenderContext) SidebarWidth() int {
	return 24
}

// MainWidth returns the main content width
func (c RenderContext) MainWidth() int {
	w := c.Width - c.SidebarWidth() - 4
	if w < 50 {
		return 50
	}
	return w
}

// --- Common Render Helpers ---

// RenderLayout combines sidebar, main content, and footer
func RenderLayout(sidebar, main, footer string) string {
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	return ui.DocStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content, footer))
}

// RenderCentered renders content in the document style
func RenderCentered(content string) string {
	return ui.DocStyle.Render(content)
}
