package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// renderModalTitle renders a title with block pattern decoration.
// Creates format: "Title ▀▄▀▄▀▄▀▄" with a gradient from primary to secondary.
// Uses the same block characters as the logo (▀ ▄) for visual consistency.
func renderModalTitle(title string, width int) string {
	t := theme.Current()
	styledTitle := t.S().ModalTitle.Render(title)
	titleWidth := lipgloss.Width(styledTitle)

	remainingWidth := width - titleWidth - 1 // -1 for space before pattern
	if remainingWidth <= 0 {
		return styledTitle
	}

	// Build gradient pattern in segments for performance
	// Use ~8 color stops across the width instead of per-character
	const maxStops = 8
	segmentSize := remainingWidth / maxStops
	if segmentSize < 1 {
		segmentSize = 1
	}

	var pattern strings.Builder
	for i := 0; i < remainingWidth; i += segmentSize {
		pos := float64(i) / float64(remainingWidth)
		color := theme.InterpolateColor(string(t.Primary), string(t.Secondary), pos)
		charStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

		// Build alternating ▀▄ pattern for this segment
		end := i + segmentSize
		if end > remainingWidth {
			end = remainingWidth
		}
		var segmentPattern strings.Builder
		for j := i; j < end; j++ {
			if j%2 == 0 {
				segmentPattern.WriteRune('▄')
			} else {
				segmentPattern.WriteRune('▀')
			}
		}
		pattern.WriteString(charStyle.Render(segmentPattern.String()))
	}

	return styledTitle + " " + pattern.String()
}
