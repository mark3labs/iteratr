package tui

import (
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// Standard key representations for consistent hints across the app.
// Use arrow symbols (↑↓) as primary, with j/k mentioned as backup where applicable.
const (
	KeyUpDown   = "↑/↓"    // Arrow keys for navigation
	KeyUpDownJK = "↑↓/jk"  // Arrows with vim backup
	KeyArrows   = "arrows" // Generic arrow reference
	KeyEnter    = "enter"
	KeySpace    = "space"
	KeyEsc      = "esc"
	KeyTab      = "tab"
	KeyCtrlC    = "ctrl+c"
	KeyCtrlX    = "ctrl+x"   // Prefix modifier key
	KeyCtrlXL   = "ctrl+x l" // Toggle logs
	KeyCtrlXB   = "ctrl+x b" // Toggle sidebar
	KeyCtrlXN   = "ctrl+x n" // Create note
	KeyCtrlXT   = "ctrl+x t" // Create task
	KeyCtrlXP   = "ctrl+x p" // Pause/resume
	KeyPgUpDown = "pgup/pgdn"
	KeyHomeEnd  = "home/end"
	KeyI        = "i"
)

// RenderHint renders a single key-description pair.
// Example: RenderHint("enter", "select") -> "enter select"
func RenderHint(key, desc string) string {
	s := theme.Current().S()
	return s.HintKey.Render(key) + " " + s.HintDesc.Render(desc)
}

// RenderHintBar renders a hint bar with multiple key-description pairs.
// Pairs are separated by " . " (bullet point separator).
// Example: RenderHintBar("up/down", "scroll", "enter", "select", "esc", "back")
// Returns: "up/down scroll . enter select . esc back"
func RenderHintBar(pairs ...string) string {
	if len(pairs) == 0 || len(pairs)%2 != 0 {
		return ""
	}

	s := theme.Current().S()
	var result string

	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		desc := pairs[i+1]

		if i > 0 {
			result += " " + s.HintSeparator.Render(".") + " "
		}

		result += s.HintKey.Render(key) + " " + s.HintDesc.Render(desc)
	}

	return result
}

// Common hint bar presets for consistency.

// HintScrollWithVim returns hints for scrollable viewports with vim keys.
// "up/down/jk scroll . pgup/pgdn page"
func HintScrollWithVim() string {
	return RenderHintBar(KeyUpDownJK, "scroll", KeyPgUpDown, "page")
}

// HintModal returns standard modal hints.
// "tab cycle . enter submit . esc close"
func HintModal() string {
	return RenderHintBar(KeyTab, "cycle", KeyEnter, "submit", KeyEsc, "close")
}

// HintLogs returns hints for the log viewer modal.
// "up/down scroll . esc close"
func HintLogs() string {
	return RenderHintBar(KeyUpDown, "scroll", KeyEsc, "close")
}

// HintInput returns hints for input fields.
// When focused: "enter send . esc cancel"
// When not focused: "i type message"
func HintInputFocused() string {
	return RenderHintBar(KeyEnter, "send", KeyEsc, "cancel")
}

func HintInputBlurred() string {
	return RenderHint(KeyI, "type message")
}

// HintStatus returns hints for the status bar.
// "ctrl+x p pause . ctrl+x l logs . ctrl+c quit"
func HintStatus() string {
	return RenderHintBar(KeyCtrlXP, "pause", KeyCtrlXL, "logs", KeyCtrlC, "quit")
}
