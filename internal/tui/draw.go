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

// DrawBorder renders a border around an area and returns the inner content area
func DrawBorder(scr uv.Screen, area uv.Rectangle, style lipgloss.Style) uv.Rectangle {
	// Render border frame
	border := style.Width(area.Dx()).Height(area.Dy()).Render("")
	uv.NewStyledString(border).Draw(scr, area)

	// Return inner content area (accounting for border thickness)
	// Normal border takes 1 cell on each side
	innerX := area.Min.X + 1
	innerY := area.Min.Y + 1
	innerWidth := area.Dx() - 2
	innerHeight := area.Dy() - 2

	// Ensure dimensions are not negative
	if innerWidth < 0 {
		innerWidth = 0
	}
	if innerHeight < 0 {
		innerHeight = 0
	}

	return uv.Rectangle{
		Min: uv.Position{X: innerX, Y: innerY},
		Max: uv.Position{X: innerX + innerWidth, Y: innerY + innerHeight},
	}
}

// DrawPanel renders a bordered panel with optional title and returns the inner area
func DrawPanel(scr uv.Screen, area uv.Rectangle, title string, focused bool) uv.Rectangle {
	// Select border style based on focus
	borderStyle := styleBorder.Copy()
	if focused {
		borderStyle = borderStyle.BorderForeground(colorPrimary)
	}

	// Draw the border
	inner := DrawBorder(scr, area, borderStyle)

	// Draw title if provided
	if title != "" {
		titleStyle := stylePanelTitle
		if focused {
			titleStyle = stylePanelTitle.Copy().Foreground(colorPrimary)
		}
		titleText := titleStyle.Render(" " + title + " ")

		// Draw title at top-left of border (overlaying the border line)
		titleArea := uv.Rectangle{
			Min: uv.Position{X: area.Min.X + 2, Y: area.Min.Y},
			Max: uv.Position{X: area.Min.X + 2 + len(title) + 2, Y: area.Min.Y + 1},
		}
		uv.NewStyledString(titleText).Draw(scr, titleArea)
	}

	return inner
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
