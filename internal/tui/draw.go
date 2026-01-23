package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
)

// DrawText renders plain text at a position
func DrawText(scr uv.Screen, area uv.Rectangle, text string) {
	uv.NewStyledString(text).Draw(scr, area)
}

// DrawStyled renders lipgloss-styled content at a position
func DrawStyled(scr uv.Screen, area uv.Rectangle, style lipgloss.Style, text string) {
	content := style.Width(area.Dx()).Height(area.Dy()).Render(text)
	uv.NewStyledString(content).Draw(scr, area)
}

// FillArea clears an area with a styled background
func FillArea(scr uv.Screen, area uv.Rectangle, style lipgloss.Style) {
	fill := style.Width(area.Dx()).Height(area.Dy()).Render("")
	uv.NewStyledString(fill).Draw(scr, area)
}

// DrawPanel renders a panel with a title header and returns the inner content area.
// Focus is indicated by the header color (no borders).
func DrawPanel(scr uv.Screen, area uv.Rectangle, title string, focused bool) uv.Rectangle {
	headerHeight := 0

	// Draw title header if provided
	if title != "" {
		headerHeight = 1
		titleStyle := stylePanelTitle
		if focused {
			titleStyle = stylePanelTitleFocused
		}
		titleText := titleStyle.Render(title)

		titleArea := uv.Rectangle{
			Min: uv.Position{X: area.Min.X, Y: area.Min.Y},
			Max: uv.Position{X: area.Max.X, Y: area.Min.Y + 1},
		}
		uv.NewStyledString(titleText).Draw(scr, titleArea)
	}

	// Return content area below the header
	innerHeight := area.Dy() - headerHeight
	if innerHeight < 0 {
		innerHeight = 0
	}

	return uv.Rectangle{
		Min: uv.Position{X: area.Min.X, Y: area.Min.Y + headerHeight},
		Max: uv.Position{X: area.Max.X, Y: area.Min.Y + headerHeight + innerHeight},
	}
}

// DrawScrollIndicator renders a scroll position indicator
func DrawScrollIndicator(scr uv.Screen, area uv.Rectangle, percent float64) {
	indicator := fmt.Sprintf(" %d%% ", int(percent*100))
	indicatorStyle := styleScrollIndicator

	// Position at bottom-right of area
	indicatorArea := uv.Rectangle{
		Min: uv.Position{X: area.Max.X - len(indicator), Y: area.Max.Y - 1},
		Max: uv.Position{X: area.Max.X, Y: area.Max.Y},
	}

	DrawStyled(scr, indicatorArea, indicatorStyle, indicator)
}

// DrawHorizontalDivider renders a horizontal dividing line
func DrawHorizontalDivider(scr uv.Screen, area uv.Rectangle, style lipgloss.Style) {
	divider := style.Width(area.Dx()).Render("─")
	uv.NewStyledString(divider).Draw(scr, area)
}

// DrawVerticalDivider renders a vertical dividing line
func DrawVerticalDivider(scr uv.Screen, area uv.Rectangle, style lipgloss.Style) {
	// Build vertical line character by character
	for i := 0; i < area.Dy(); i++ {
		lineArea := uv.Rectangle{
			Min: uv.Position{X: area.Min.X, Y: area.Min.Y + i},
			Max: uv.Position{X: area.Min.X + 1, Y: area.Min.Y + i + 1},
		}
		divider := style.Render("│")
		uv.NewStyledString(divider).Draw(scr, lineArea)
	}
}
