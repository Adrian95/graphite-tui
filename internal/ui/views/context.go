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

// Cached layout calculations to avoid recalculating on every render
var (
	cachedWidth       int
	cachedHeight      int
	cachedLayoutMode  LayoutMode
	cachedSidebarW    int
	cachedMainW       int
	cachedMainSplitW  int
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
	// Update cache if dimensions changed
	if width != cachedWidth || height != cachedHeight {
		cachedWidth = width
		cachedHeight = height
		// Recalculate layout mode
		if width >= 140 {
			cachedLayoutMode = LayoutGrid2x2
		} else if width >= 80 {
			cachedLayoutMode = LayoutStack1x4
		} else {
			cachedLayoutMode = LayoutSingle
		}
		// Recalculate sidebar width
		switch cachedLayoutMode {
		case LayoutGrid2x2:
			if width >= 200 {
				cachedSidebarW = 28
			} else {
				cachedSidebarW = 24
			}
		case LayoutStack1x4:
			cachedSidebarW = 20
		case LayoutSingle:
			cachedSidebarW = 16
		default:
			cachedSidebarW = 20
		}
		// Recalculate main width
		cachedMainW = width - cachedSidebarW - 4
		if cachedMainW < 40 {
			cachedMainW = 40
		}
		// Recalculate split width
		cachedMainSplitW = (cachedMainW - 2) / 2
	}
	return RenderContext{Width: width, Height: height}
}

// SidebarWidth returns the standard sidebar width
func (c RenderContext) SidebarWidth() int {
	return cachedSidebarW
}

// MainWidth returns the main content width
func (c RenderContext) MainWidth() int {
	return cachedMainW
}

// MainSplitWidth returns half-width for panel layout
func (c RenderContext) MainSplitWidth() int {
	return cachedMainSplitW
}

// GetLayoutMode returns the appropriate layout mode based on screen width
func (c RenderContext) GetLayoutMode() LayoutMode {
	return cachedLayoutMode
}

// GetSidebarWidth returns responsive sidebar width
func (c RenderContext) GetSidebarWidth() int {
	return cachedSidebarW
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
