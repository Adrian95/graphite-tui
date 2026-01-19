package views

import (
	"github.com/Adrian95/graphite-tui/v2/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// LayoutMode represents different responsive layout modes
type LayoutMode int

const (
	LayoutGrid2x2  LayoutMode = iota // 2x2 grid for wide screens
	LayoutStack1x4                   // Single column stack for medium screens
	LayoutSingle                     // Single panel for narrow screens
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
	return 28
}

// MainWidth returns the main content width
func (c RenderContext) MainWidth() int {
	w := c.Width - c.GetSidebarWidth() - 4
	if w < 40 {
		return 40
	}
	return w
}

// MainSplitWidth returns half-width for panel layout
func (c RenderContext) MainSplitWidth() int {
	w := c.MainWidth()
	gap := 2
	return (w - gap) / 2
}

// GetLayoutMode returns the appropriate layout mode based on screen width
func (c RenderContext) GetLayoutMode() LayoutMode {
	totalWidth := c.Width
	if totalWidth >= 140 {
		return LayoutGrid2x2
	} else if totalWidth >= 80 {
		return LayoutStack1x4
	}
	return LayoutSingle
}

// GetSidebarWidth returns responsive sidebar width
func (c RenderContext) GetSidebarWidth() int {
	mode := c.GetLayoutMode()
	switch mode {
	case LayoutGrid2x2:
		if c.Width >= 200 {
			return 28
		}
		return 24
	case LayoutStack1x4:
		return 20
	case LayoutSingle:
		return 16
	}
	return 20
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
