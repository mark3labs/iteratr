package wizard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ButtonState represents the visual state of a button.
type ButtonState int

const (
	ButtonNormal   ButtonState = iota // Normal state (enabled)
	ButtonDisabled                    // Disabled state (grayed out)
	ButtonFocused                     // Focused/highlighted state
)

// Button represents a single button in the button bar.
type Button struct {
	Label string
	State ButtonState
}

// ButtonBar manages a set of buttons with consistent styling.
type ButtonBar struct {
	buttons []Button
	width   int
}

// NewButtonBar creates a new button bar with the given buttons.
func NewButtonBar(buttons []Button) *ButtonBar {
	return &ButtonBar{
		buttons: buttons,
		width:   60,
	}
}

// SetWidth updates the width for the button bar.
func (b *ButtonBar) SetWidth(width int) {
	b.width = width
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

	// Render each button
	var renderedButtons []string
	for _, btn := range b.buttons {
		var rendered string
		switch btn.State {
		case ButtonDisabled:
			rendered = disabledStyle.Render(btn.Label)
		case ButtonFocused:
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
