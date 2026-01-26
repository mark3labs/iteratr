package wizard

import (
	"strings"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// ButtonState represents the visual state of a button.
type ButtonState int

const (
	ButtonNormal   ButtonState = iota // Normal state (enabled)
	ButtonDisabled                    // Disabled state (grayed out)
	ButtonFocused                     // Focused/highlighted state
)

// ButtonID identifies which button was clicked or activated.
type ButtonID int

const (
	ButtonNone ButtonID = iota // No button
	ButtonBack                 // Back/Cancel button (left)
	ButtonNext                 // Next/Finish button (right)
)

// Button represents a single button in the button bar.
type Button struct {
	Label string
	State ButtonState
}

// ButtonBar manages a set of buttons with consistent styling.
type ButtonBar struct {
	buttons     []Button
	width       int
	focusIndex  int            // -1 = no focus, 0 = first button, 1 = second button
	buttonAreas []uv.Rectangle // Cached button hit areas (set during Render)
}

// NewButtonBar creates a new button bar with the given buttons.
func NewButtonBar(buttons []Button) *ButtonBar {
	return &ButtonBar{
		buttons:    buttons,
		width:      60,
		focusIndex: -1, // No focus by default
	}
}

// SetWidth updates the width for the button bar.
func (b *ButtonBar) SetWidth(width int) {
	b.width = width
}

// Focus sets focus on the button bar, starting with the rightmost enabled button.
func (b *ButtonBar) Focus() {
	// Find rightmost enabled button (typically Next/Finish)
	for i := len(b.buttons) - 1; i >= 0; i-- {
		if b.buttons[i].State != ButtonDisabled {
			b.focusIndex = i
			return
		}
	}
	// If all disabled, focus first anyway (unlikely)
	if len(b.buttons) > 0 {
		b.focusIndex = 0
	}
}

// Blur removes focus from the button bar.
func (b *ButtonBar) Blur() {
	b.focusIndex = -1
}

// IsFocused returns true if any button has focus.
func (b *ButtonBar) IsFocused() bool {
	return b.focusIndex >= 0
}

// FocusNext moves focus to the next enabled button. Returns false if at end.
func (b *ButtonBar) FocusNext() bool {
	for i := b.focusIndex + 1; i < len(b.buttons); i++ {
		if b.buttons[i].State != ButtonDisabled {
			b.focusIndex = i
			return true
		}
	}
	return false // At end, no more buttons
}

// FocusPrev moves focus to the previous enabled button. Returns false if at start.
func (b *ButtonBar) FocusPrev() bool {
	for i := b.focusIndex - 1; i >= 0; i-- {
		if b.buttons[i].State != ButtonDisabled {
			b.focusIndex = i
			return true
		}
	}
	return false // At start, no more buttons
}

// FocusedButton returns the ID of the currently focused button.
func (b *ButtonBar) FocusedButton() ButtonID {
	if b.focusIndex < 0 || b.focusIndex >= len(b.buttons) {
		return ButtonNone
	}
	// First button is Back/Cancel, second is Next/Finish
	if b.focusIndex == 0 {
		return ButtonBack
	}
	return ButtonNext
}

// ButtonAtPosition returns the ButtonID at the given screen coordinates.
// Returns ButtonNone if no button at that position or button is disabled.
func (b *ButtonBar) ButtonAtPosition(x, y int) ButtonID {
	for i, area := range b.buttonAreas {
		if x >= area.Min.X && x < area.Max.X && y >= area.Min.Y && y < area.Max.Y {
			if b.buttons[i].State != ButtonDisabled {
				if i == 0 {
					return ButtonBack
				}
				return ButtonNext
			}
			return ButtonNone // Clicked on disabled button
		}
	}
	return ButtonNone
}

// SetButtonAreas sets the screen coordinates for button hit detection.
// Called by parent during rendering once button positions are known.
func (b *ButtonBar) SetButtonAreas(areas []uv.Rectangle) {
	b.buttonAreas = areas
}

// Render renders the button bar with proper spacing and styling.
func (b *ButtonBar) Render() string {
	if len(b.buttons) == 0 {
		return ""
	}

	// Define button styles
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4")).
		Background(lipgloss.Color("#313244")).
		Padding(0, 2).
		MarginLeft(1).
		MarginRight(1)

	disabledStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086")).
		Background(lipgloss.Color("#181825")).
		Padding(0, 2).
		MarginLeft(1).
		MarginRight(1)

	focusedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("#b4befe")).
		Bold(true).
		Padding(0, 2).
		MarginLeft(1).
		MarginRight(1)

	// Render each button, applying focus state from focusIndex
	var renderedButtons []string
	for i, btn := range b.buttons {
		var rendered string
		// Check if this button is focused (focusIndex overrides stored state)
		isFocused := b.focusIndex == i && btn.State != ButtonDisabled

		switch {
		case btn.State == ButtonDisabled:
			rendered = disabledStyle.Render(btn.Label)
		case isFocused:
			rendered = focusedStyle.Render(btn.Label)
		default: // ButtonNormal
			rendered = normalStyle.Render(btn.Label)
		}
		renderedButtons = append(renderedButtons, rendered)
	}

	// Join buttons with spacing
	result := strings.Join(renderedButtons, "")

	// Center the button bar
	return lipgloss.Place(b.width, 1, lipgloss.Center, lipgloss.Center, result)
}

// CreateBackNextButtons creates standard Back/Next button set.
// backEnabled: whether Back button is enabled
// nextEnabled: whether Next button is enabled (false if step invalid)
// nextLabel: custom label for next button (e.g., "Next", "Finish")
func CreateBackNextButtons(backEnabled, nextEnabled bool, nextLabel string) []Button {
	buttons := make([]Button, 0, 2)

	// Back button
	backState := ButtonNormal
	if !backEnabled {
		backState = ButtonDisabled
	}
	buttons = append(buttons, Button{
		Label: "← Back",
		State: backState,
	})

	// Next/Finish button
	nextState := ButtonNormal
	if !nextEnabled {
		nextState = ButtonDisabled
	}
	buttons = append(buttons, Button{
		Label: nextLabel,
		State: nextState,
	})

	return buttons
}

// CreateCancelNextButtons creates Cancel/Next button set (for step 1).
func CreateCancelNextButtons(nextEnabled bool, nextLabel string) []Button {
	buttons := make([]Button, 0, 2)

	// Cancel button
	buttons = append(buttons, Button{
		Label: "Cancel",
		State: ButtonNormal,
	})

	// Next button
	nextState := ButtonNormal
	if !nextEnabled {
		nextState = ButtonDisabled
	}
	buttons = append(buttons, Button{
		Label: nextLabel,
		State: nextState,
	})

	return buttons
}
