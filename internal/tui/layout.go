package tui

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"
)

// splitVertical splits the area vertically into two parts.
// The first part has the given fixed height; the second takes the remainder.
// Replaces the removed uv.SplitVertical(area, uv.Fixed(n)).
func splitVertical(area uv.Rectangle, fixedHeight int) (top uv.Rectangle, bottom uv.Rectangle) {
	rects := layout.Vertical(layout.Len(fixedHeight), layout.Fill(1)).Split(area)
	if len(rects) >= 2 {
		return rects[0], rects[1]
	}
	return area, uv.Rectangle{}
}

// splitHorizontal splits the area horizontally into two parts.
// The first part has the given fixed width; the second takes the remainder.
// Replaces the removed uv.SplitHorizontal(area, uv.Fixed(n)).
func splitHorizontal(area uv.Rectangle, fixedWidth int) (left uv.Rectangle, right uv.Rectangle) {
	rects := layout.Horizontal(layout.Len(fixedWidth), layout.Fill(1)).Split(area)
	if len(rects) >= 2 {
		return rects[0], rects[1]
	}
	return area, uv.Rectangle{}
}

// Layout breakpoints and dimensions
const (
	// CompactWidthBreakpoint is the minimum width for desktop mode
	CompactWidthBreakpoint = 100
	// CompactHeightBreakpoint is the minimum height for desktop mode
	CompactHeightBreakpoint = 25
	// SidebarWidthDesktop is the width of the sidebar in desktop mode
	SidebarWidthDesktop = 45
	// StatusHeight is the height of the status bar in rows
	StatusHeight = 1
)

// LayoutMode represents the layout mode based on terminal size
type LayoutMode int

const (
	// LayoutDesktop is the full layout with sidebar
	LayoutDesktop LayoutMode = iota
	// LayoutCompact is the compact layout without sidebar
	LayoutCompact
)

// Layout defines the rectangular regions for all UI components
type Layout struct {
	Mode    LayoutMode
	Area    uv.Rectangle
	Content uv.Rectangle
	Main    uv.Rectangle
	Sidebar uv.Rectangle
	Status  uv.Rectangle
}

// IsCompact returns true if the layout is in compact mode
func (l Layout) IsCompact() bool {
	return l.Mode == LayoutCompact
}

// CalculateLayout computes the layout rectangles based on terminal dimensions
// and sidebar visibility state. When sidebarHidden is true in desktop mode,
// the main area expands to fill the full content width.
func CalculateLayout(width, height int, sidebarHidden bool) Layout {
	// Determine layout mode based on breakpoints
	mode := LayoutDesktop
	if width < CompactWidthBreakpoint || height < CompactHeightBreakpoint {
		mode = LayoutCompact
	}

	// Create the full area
	area := uv.Rectangle{
		Max: uv.Position{X: width, Y: height},
	}

	// Split vertically: content | status
	contentRect, statusRect := splitVertical(area, area.Dy()-StatusHeight)

	// Split content horizontally: main | sidebar (desktop mode only)
	var mainRect, sidebarRect uv.Rectangle
	if mode == LayoutDesktop && !sidebarHidden {
		// Desktop mode with sidebar visible
		// Calculate sidebar width (max 45, or 1/3 of content width)
		sidebarWidth := SidebarWidthDesktop
		if contentRect.Dx()/3 < sidebarWidth {
			sidebarWidth = contentRect.Dx() / 3
		}

		// Split horizontally: main (flexible) | gap (1 char) | sidebar (fixed width)
		mainRect, sidebarRect = splitHorizontal(contentRect, contentRect.Dx()-sidebarWidth)
		mainRect.Max.X -= 1 // 1-char gap between main and sidebar
	} else {
		// Compact mode or sidebar hidden: no sidebar, main takes full content width
		mainRect = contentRect
		sidebarRect = uv.Rectangle{} // Empty rectangle
	}

	return Layout{
		Mode:    mode,
		Area:    area,
		Content: contentRect,
		Main:    mainRect,
		Sidebar: sidebarRect,
		Status:  statusRect,
	}
}
