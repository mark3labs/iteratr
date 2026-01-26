package wizard

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette (Catppuccin Mocha) - used by renderHintBar
var (
	colorSubtext0 = lipgloss.Color("#a6adc8") // Subtext0
	colorSubtext1 = lipgloss.Color("#bac2de") // Subtext1
	colorSurface2 = lipgloss.Color("#585b70") // Surface2
)

// Hint bar styles
var (
	styleHintKey = lipgloss.NewStyle().
			Foreground(colorSubtext1).
			Bold(true)

	styleHintDesc = lipgloss.NewStyle().
			Foreground(colorSubtext0)

	styleHintSeparator = lipgloss.NewStyle().
				Foreground(colorSurface2)
)

// renderHintBar renders a hint bar with the given key-description pairs.
// Example: renderHintBar("↑↓", "navigate", "enter", "select", "esc", "back")
// Returns: "↑↓ navigate • enter select • esc back"
func renderHintBar(pairs ...string) string {
	if len(pairs) == 0 || len(pairs)%2 != 0 {
		return ""
	}

	var result string
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		desc := pairs[i+1]

		if i > 0 {
			result += " " + styleHintSeparator.Render("•") + " "
		}

		result += styleHintKey.Render(key) + " " + styleHintDesc.Render(desc)
	}

	return result
}
