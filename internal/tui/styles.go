package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// renderModalTitle renders a title with block pattern decoration.
// Creates format: "Title ▀▄▀▄▀▄▀▄" with a gradient from primary to secondary.
// Uses the same block characters as the logo (▀ ▄) for visual consistency.
//
// Both the title text and the gradient pattern are explicitly drawn on the
// modal's BgBase background. This is required because the half-block glyphs
// (▄ ▀) are partially transparent and would otherwise show the previous
// terminal background state, creating an uneven/checkered look against the
// modal surface.
func renderModalTitle(title string, width int) string {
	t := theme.Current()
	bg := string(t.BgBase)

	// Calculate remaining width for pattern
	titleLen := len(title)
	remainingWidth := width - titleLen - 1 // -1 for space
	if remainingWidth <= 0 {
		return t.S().ModalTitle.UnsetAlign().Background(lipgloss.Color(bg)).Render(title)
	}

	// Build pattern with block characters
	pattern := strings.Repeat("▄▀", remainingWidth/2)
	if remainingWidth%2 == 1 {
		pattern += "▄"
	}

	// Style title and apply gradient (with bg) to pattern
	titleStyle := t.S().ModalTitle.UnsetAlign().Background(lipgloss.Color(bg))
	styledTitle := titleStyle.Render(title)
	styledPattern := theme.ApplyGradientOnBg(pattern, string(t.Primary), string(t.Secondary), bg)

	// The single space between title and pattern also needs the modal bg so the
	// gap doesn't show through to the underlying terminal background.
	spacer := lipgloss.NewStyle().Background(lipgloss.Color(bg)).Render(" ")

	return styledTitle + spacer + styledPattern
}
