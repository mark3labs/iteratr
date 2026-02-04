package wizard

import (
	"image"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestButtonBar_InitialState(t *testing.T) {
	t.Parallel()

	buttons := []Button{
		{Label: "Cancel", State: ButtonNormal},
		{Label: "Next", State: ButtonDisabled},
	}
	bar := NewButtonBar(buttons)

	// Verify initial state
	if bar.focusIndex != -1 {
		t.Errorf("initial focusIndex = %d; want -1 (no focus)", bar.focusIndex)
	}
	if bar.IsFocused() {
		t.Error("should not be focused initially")
	}
	if bar.FocusedButton() != ButtonNone {
		t.Errorf("FocusedButton() = %v; want ButtonNone", bar.FocusedButton())
	}
}

func TestButtonBar_Focus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		buttons        []Button
		expectedIdx    int
		expectedButton ButtonID
	}{
		{
			name: "focus_on_two_enabled_buttons",
			buttons: []Button{
				{Label: "Cancel", State: ButtonNormal},
				{Label: "Next", State: ButtonNormal},
			},
			expectedIdx:    1, // Rightmost enabled (Next)
			expectedButton: ButtonNext,
		},
		{
			name: "focus_with_disabled_next",
			buttons: []Button{
				{Label: "Cancel", State: ButtonNormal},
				{Label: "Next", State: ButtonDisabled},
			},
			expectedIdx:    0, // Only Cancel enabled
			expectedButton: ButtonBack,
		},
		{
			name: "focus_with_all_disabled",
			buttons: []Button{
				{Label: "Cancel", State: ButtonDisabled},
				{Label: "Next", State: ButtonDisabled},
			},
			expectedIdx:    0, // Falls back to first
			expectedButton: ButtonBack,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bar := NewButtonBar(tt.buttons)
			bar.Focus()

			if bar.focusIndex != tt.expectedIdx {
				t.Errorf("focusIndex = %d; want %d", bar.focusIndex, tt.expectedIdx)
			}
			if !bar.IsFocused() {
				t.Error("should be focused")
			}
			if bar.FocusedButton() != tt.expectedButton {
				t.Errorf("FocusedButton() = %v; want %v", bar.FocusedButton(), tt.expectedButton)
			}
		})
	}
}

func TestButtonBar_FocusFirst(t *testing.T) {
	t.Parallel()

	buttons := []Button{
		{Label: "Back", State: ButtonNormal},
		{Label: "Next", State: ButtonNormal},
	}
	bar := NewButtonBar(buttons)
	bar.FocusFirst()

	if bar.focusIndex != 0 {
		t.Errorf("focusIndex = %d; want 0 (should focus first button)", bar.focusIndex)
	}
	if bar.FocusedButton() != ButtonBack {
		t.Errorf("FocusedButton() = %v; want ButtonBack", bar.FocusedButton())
	}
}

func TestButtonBar_FocusLast(t *testing.T) {
	t.Parallel()

	buttons := []Button{
		{Label: "Back", State: ButtonNormal},
		{Label: "Next", State: ButtonNormal},
	}
	bar := NewButtonBar(buttons)
	bar.FocusLast()

	if bar.focusIndex != 1 {
		t.Errorf("focusIndex = %d; want 1 (should focus last button)", bar.focusIndex)
	}
	if bar.FocusedButton() != ButtonNext {
		t.Errorf("FocusedButton() = %v; want ButtonNext", bar.FocusedButton())
	}
}

func TestButtonBar_Blur(t *testing.T) {
	t.Parallel()

	buttons := []Button{
		{Label: "Cancel", State: ButtonNormal},
		{Label: "Next", State: ButtonNormal},
	}
	bar := NewButtonBar(buttons)
	bar.Focus()

	// Verify focused
	if !bar.IsFocused() {
		t.Error("should be focused before blur")
	}

	// Blur
	bar.Blur()
	if bar.focusIndex != -1 {
		t.Errorf("focusIndex = %d after blur; want -1", bar.focusIndex)
	}
	if bar.IsFocused() {
		t.Error("should not be focused after blur")
	}
	if bar.FocusedButton() != ButtonNone {
		t.Errorf("FocusedButton() = %v after blur; want ButtonNone", bar.FocusedButton())
	}
}

func TestButtonBar_FocusNext(t *testing.T) {
	t.Parallel()

	buttons := []Button{
		{Label: "Back", State: ButtonNormal},
		{Label: "Next", State: ButtonNormal},
	}
	bar := NewButtonBar(buttons)
	bar.FocusFirst()

	// Move from first to second
	if bar.focusIndex != 0 {
		t.Errorf("initial focusIndex = %d; want 0", bar.focusIndex)
	}
	moved := bar.FocusNext()
	if !moved {
		t.Error("FocusNext() should return true when moving to next button")
	}
	if bar.focusIndex != 1 {
		t.Errorf("focusIndex after FocusNext() = %d; want 1", bar.focusIndex)
	}

	// Try to move past last
	moved = bar.FocusNext()
	if moved {
		t.Error("FocusNext() should return false when at last button")
	}
	if bar.focusIndex != 1 {
		t.Errorf("focusIndex = %d; want 1 (should stay at last button)", bar.focusIndex)
	}
}

func TestButtonBar_FocusPrev(t *testing.T) {
	t.Parallel()

	buttons := []Button{
		{Label: "Back", State: ButtonNormal},
		{Label: "Next", State: ButtonNormal},
	}
	bar := NewButtonBar(buttons)
	bar.FocusLast()

	// Move from last to first
	if bar.focusIndex != 1 {
		t.Errorf("initial focusIndex = %d; want 1", bar.focusIndex)
	}
	moved := bar.FocusPrev()
	if !moved {
		t.Error("FocusPrev() should return true when moving to previous button")
	}
	if bar.focusIndex != 0 {
		t.Errorf("focusIndex after FocusPrev() = %d; want 0", bar.focusIndex)
	}

	// Try to move before first
	moved = bar.FocusPrev()
	if moved {
		t.Error("FocusPrev() should return false when at first button")
	}
	if bar.focusIndex != 0 {
		t.Errorf("focusIndex = %d; want 0 (should stay at first button)", bar.focusIndex)
	}
}

func TestButtonBar_FocusNavigation_SkipsDisabled(t *testing.T) {
	t.Parallel()

	buttons := []Button{
		{Label: "Back", State: ButtonNormal},
		{Label: "Middle", State: ButtonDisabled},
		{Label: "Next", State: ButtonNormal},
	}
	bar := NewButtonBar(buttons)
	bar.focusIndex = 0 // Start at first

	// FocusNext should skip disabled button at index 1
	moved := bar.FocusNext()
	if !moved {
		t.Error("FocusNext() should return true when skipping disabled button")
	}
	if bar.focusIndex != 2 {
		t.Errorf("focusIndex = %d; want 2 (should skip disabled button)", bar.focusIndex)
	}

	// FocusPrev should skip disabled button at index 1
	moved = bar.FocusPrev()
	if !moved {
		t.Error("FocusPrev() should return true when skipping disabled button")
	}
	if bar.focusIndex != 0 {
		t.Errorf("focusIndex = %d; want 0 (should skip disabled button)", bar.focusIndex)
	}
}

func TestButtonBar_ButtonAtPosition(t *testing.T) {
	t.Parallel()

	buttons := []Button{
		{Label: "Back", State: ButtonNormal},
		{Label: "Next", State: ButtonDisabled},
	}
	bar := NewButtonBar(buttons)

	// Set button areas (simulate render)
	areas := []uv.Rectangle{
		{Min: image.Point{X: 10, Y: 5}, Max: image.Point{X: 20, Y: 6}}, // Back button
		{Min: image.Point{X: 25, Y: 5}, Max: image.Point{X: 35, Y: 6}}, // Next button (disabled)
	}
	bar.SetButtonAreas(areas)

	// Click on first button (enabled)
	btnID := bar.ButtonAtPosition(15, 5)
	if btnID != ButtonBack {
		t.Errorf("ButtonAtPosition(15, 5) = %v; want ButtonBack", btnID)
	}

	// Click on second button (disabled) - should return ButtonNone
	btnID = bar.ButtonAtPosition(30, 5)
	if btnID != ButtonNone {
		t.Errorf("ButtonAtPosition(30, 5) on disabled button = %v; want ButtonNone", btnID)
	}

	// Click outside buttons
	btnID = bar.ButtonAtPosition(0, 0)
	if btnID != ButtonNone {
		t.Errorf("ButtonAtPosition(0, 0) outside buttons = %v; want ButtonNone", btnID)
	}
}

func TestButtonBar_Render_Visual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		buttons     []Button
		focusIdx    int
		description string
	}{
		{
			name: "two_buttons_no_focus",
			buttons: []Button{
				{Label: "← Back", State: ButtonNormal},
				{Label: "Next →", State: ButtonNormal},
			},
			focusIdx:    -1,
			description: "Both buttons normal, no focus",
		},
		{
			name: "focus_on_first",
			buttons: []Button{
				{Label: "← Back", State: ButtonNormal},
				{Label: "Next →", State: ButtonNormal},
			},
			focusIdx:    0,
			description: "First button focused",
		},
		{
			name: "focus_on_second",
			buttons: []Button{
				{Label: "← Back", State: ButtonNormal},
				{Label: "Next →", State: ButtonNormal},
			},
			focusIdx:    1,
			description: "Second button focused",
		},
		{
			name: "disabled_next",
			buttons: []Button{
				{Label: "← Back", State: ButtonNormal},
				{Label: "Next →", State: ButtonDisabled},
			},
			focusIdx:    0,
			description: "Next button disabled, Back focused",
		},
		{
			name: "cancel_next",
			buttons: []Button{
				{Label: "Cancel", State: ButtonNormal},
				{Label: "Finish", State: ButtonNormal},
			},
			focusIdx:    1,
			description: "Cancel/Finish buttons, Finish focused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bar := NewButtonBar(tt.buttons)
			bar.focusIndex = tt.focusIdx
			bar.SetWidth(60)

			output := bar.Render()

			// Verify output is non-empty
			if output == "" {
				t.Error("Render() returned empty string")
			}

			// Verify button labels are present
			for _, btn := range tt.buttons {
				if !strings.Contains(output, btn.Label) {
					t.Errorf("Render() output missing button label %q", btn.Label)
				}
			}
		})
	}
}

func TestButtonBar_CreateBackNextButtons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		backEnabled  bool
		nextEnabled  bool
		nextLabel    string
		expectedLen  int
		expectedBack ButtonState
		expectedNext ButtonState
	}{
		{
			name:         "both_enabled",
			backEnabled:  true,
			nextEnabled:  true,
			nextLabel:    "Next →",
			expectedLen:  2,
			expectedBack: ButtonNormal,
			expectedNext: ButtonNormal,
		},
		{
			name:         "back_disabled",
			backEnabled:  false,
			nextEnabled:  true,
			nextLabel:    "Finish",
			expectedLen:  2,
			expectedBack: ButtonDisabled,
			expectedNext: ButtonNormal,
		},
		{
			name:         "next_disabled",
			backEnabled:  true,
			nextEnabled:  false,
			nextLabel:    "Next →",
			expectedLen:  2,
			expectedBack: ButtonNormal,
			expectedNext: ButtonDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buttons := CreateBackNextButtons(tt.backEnabled, tt.nextEnabled, tt.nextLabel)

			if len(buttons) != tt.expectedLen {
				t.Errorf("button count = %d; want %d", len(buttons), tt.expectedLen)
			}
			if buttons[0].State != tt.expectedBack {
				t.Errorf("back button state = %v; want %v", buttons[0].State, tt.expectedBack)
			}
			if buttons[1].State != tt.expectedNext {
				t.Errorf("next button state = %v; want %v", buttons[1].State, tt.expectedNext)
			}
			if buttons[1].Label != tt.nextLabel {
				t.Errorf("next button label = %q; want %q", buttons[1].Label, tt.nextLabel)
			}
			if buttons[0].Label != "← Back" {
				t.Errorf("back button label = %q; want \"← Back\"", buttons[0].Label)
			}
		})
	}
}

func TestButtonBar_CreateCancelNextButtons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nextEnabled bool
		nextLabel   string
	}{
		{
			name:        "next_enabled",
			nextEnabled: true,
			nextLabel:   "Next →",
		},
		{
			name:        "next_disabled",
			nextEnabled: false,
			nextLabel:   "Finish",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buttons := CreateCancelNextButtons(tt.nextEnabled, tt.nextLabel)

			if len(buttons) != 2 {
				t.Errorf("button count = %d; want 2", len(buttons))
			}
			if buttons[0].Label != "Cancel" {
				t.Errorf("first button label = %q; want \"Cancel\"", buttons[0].Label)
			}
			if buttons[0].State != ButtonNormal {
				t.Errorf("Cancel button state = %v; want ButtonNormal (always enabled)", buttons[0].State)
			}
			if buttons[1].Label != tt.nextLabel {
				t.Errorf("second button label = %q; want %q", buttons[1].Label, tt.nextLabel)
			}

			expectedState := ButtonNormal
			if !tt.nextEnabled {
				expectedState = ButtonDisabled
			}
			if buttons[1].State != expectedState {
				t.Errorf("next button state = %v; want %v", buttons[1].State, expectedState)
			}
		})
	}
}
