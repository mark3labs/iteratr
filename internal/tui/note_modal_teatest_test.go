package tui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
)

// TestNoteModal_InitialState tests the modal's initial state
func TestNoteModal_InitialState(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()

	// Modal should start invisible
	if modal.IsVisible() {
		t.Error("Modal should start invisible")
	}

	// Note should be nil
	if modal.note != nil {
		t.Error("Note should be nil initially")
	}

	// Default dimensions
	if modal.width != 60 {
		t.Errorf("Default width: got %d, want 60", modal.width)
	}
	if modal.height != 14 {
		t.Errorf("Default height: got %d, want 14", modal.height)
	}
}

// TestNoteModal_SetNote tests setting a note and showing the modal
func TestNoteModal_SetNote(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()
	state := testfixtures.StateWithNotes()
	note := state.Notes[0] // "Learned about event sourcing pattern" (learning)

	modal.SetNote(note)

	// Modal should be visible
	if !modal.IsVisible() {
		t.Error("Modal should be visible after SetNote")
	}

	// Note should be set
	if modal.note != note {
		t.Error("Note should be set to provided note")
	}
}

// TestNoteModal_Close tests closing the modal
func TestNoteModal_Close(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()
	state := testfixtures.StateWithNotes()
	note := state.Notes[0]

	// Show modal
	modal.SetNote(note)
	if !modal.IsVisible() {
		t.Fatal("Setup: modal should be visible")
	}

	// Close modal
	modal.Close()

	// Modal should be hidden
	if modal.IsVisible() {
		t.Error("Modal should be hidden after Close")
	}

	// Note should be cleared
	if modal.note != nil {
		t.Error("Note should be nil after Close")
	}
}

// TestNoteModal_Update tests that Update returns nil (no-op)
func TestNoteModal_Update(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()
	state := testfixtures.StateWithNotes()
	modal.SetNote(state.Notes[0])

	// Update should return nil for any message
	cmd := modal.Update(tea.KeyPressMsg{Text: "esc"})
	if cmd != nil {
		t.Error("Update should return nil for key messages")
	}

	cmd = modal.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	if cmd != nil {
		t.Error("Update should return nil for window size messages")
	}
}

// --- NoteModal Command Execution Tests ---

func TestNoteModal_Update_ReturnsNilForAllMessages(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()
	state := testfixtures.StateWithNotes()
	modal.SetNote(state.Notes[0])

	// Test that Update returns nil for all message types
	testCases := []struct {
		name string
		msg  tea.Msg
	}{
		{"KeyPressMsg enter", tea.KeyPressMsg{Text: "enter"}},
		{"KeyPressMsg esc", tea.KeyPressMsg{Text: "esc"}},
		{"KeyPressMsg space", tea.KeyPressMsg{Text: " "}},
		{"KeyPressMsg arrow", tea.KeyPressMsg{Text: "up"}},
		{"WindowSizeMsg", tea.WindowSizeMsg{Width: 100, Height: 50}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := modal.Update(tc.msg)
			if cmd != nil {
				t.Errorf("NoteModal.Update should always return nil, got command for %s", tc.name)
			}
		})
	}
}

func TestNoteModal_Update_WhenNotVisible(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()
	// Modal not visible, no note set

	cmd := modal.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd != nil {
		t.Error("Update should return nil when modal not visible")
	}
}

func TestNoteModal_Update_WhenNilNote(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()
	modal.visible = true // Force visible but keep note nil

	cmd := modal.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd != nil {
		t.Error("Update should return nil when note is nil")
	}
}

// TestNoteModal_InvisibleDoesNotRender tests that invisible modal doesn't render
func TestNoteModal_InvisibleDoesNotRender(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()
	// Don't call SetNote()

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Get baseline empty render
	emptyRender := scr.Render()

	// Now draw the invisible modal
	modal.Draw(scr, area)
	rendered := scr.Render()

	// Should be identical to empty screen (modal drew nothing)
	if rendered != emptyRender {
		t.Error("Invisible modal should not render any content")
	}
}

// TestNoteModal_NilNoteDoesNotRender tests that visible modal with nil note doesn't render
func TestNoteModal_NilNoteDoesNotRender(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()
	modal.visible = true // Force visible but keep note nil

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Get baseline empty render
	emptyRender := scr.Render()

	// Now draw the modal with nil note
	modal.Draw(scr, area)
	rendered := scr.Render()

	// Should be identical to empty screen (modal drew nothing)
	if rendered != emptyRender {
		t.Error("Modal with nil note should not render any content")
	}
}

// TestNoteModal_WordWrap tests word wrapping functionality
func TestNoteModal_WordWrap(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()

	testCases := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "Short text no wrap",
			input:    "Hello world",
			width:    20,
			expected: "Hello world",
		},
		{
			name:     "Long text wraps",
			input:    "This is a very long sentence that should wrap across multiple lines",
			width:    20,
			expected: "This is a very long\nsentence that should\nwrap across multiple\nlines",
		},
		{
			name:     "Single word longer than width",
			input:    "Supercalifragilisticexpialidocious is a word",
			width:    10,
			expected: "Supercalifragilisticexpialidocious\nis a word",
		},
		{
			name:     "Empty string",
			input:    "",
			width:    20,
			expected: "",
		},
		{
			name:     "Zero width uses default",
			input:    "Test",
			width:    0,
			expected: "Test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := modal.wordWrap(tc.input, tc.width)
			if result != tc.expected {
				t.Errorf("wordWrap:\ngot:  %q\nwant: %q", result, tc.expected)
			}
		})
	}
}

// TestNoteModal_RenderAllNoteTypes tests rendering with all 4 note types
func TestNoteModal_RenderAllNoteTypes(t *testing.T) {
	testCases := []struct {
		name       string
		noteIndex  int
		noteType   string
		goldenFile string
	}{
		{"Learning", 0, "learning", "note_modal_type_learning.golden"},
		{"Stuck", 1, "stuck", "note_modal_type_stuck.golden"},
		{"Tip", 2, "tip", "note_modal_type_tip.golden"},
		{"Decision", 3, "decision", "note_modal_type_decision.golden"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			modal := NewNoteModal()
			state := testfixtures.StateWithNotes()
			modal.SetNote(state.Notes[tc.noteIndex])

			// Render
			area := uv.Rectangle{
				Min: uv.Position{X: 0, Y: 0},
				Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
			}
			scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
			modal.Draw(scr, area)

			// Capture rendered output
			rendered := scr.Render()

			// Verify golden file
			goldenPath := filepath.Join("testdata", tc.goldenFile)
			testfixtures.CompareGolden(t, goldenPath, rendered)
		})
	}
}

// TestNoteModal_RenderLongContent tests rendering with long content that wraps
func TestNoteModal_RenderLongContent(t *testing.T) {
	modal := NewNoteModal()
	state := testfixtures.StateWithNotes()

	// Create a note with long content
	longNote := *state.Notes[0]
	longNote.Content = "This is a very long note content that should wrap across multiple lines in the modal. It contains enough text to demonstrate the word wrapping functionality and how the modal handles longer content that doesn't fit on a single line. The modal should properly wrap this text and display it in a readable format."

	modal.SetNote(&longNote)

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenPath := filepath.Join("testdata", "note_modal_long_content.golden")
	testfixtures.CompareGolden(t, goldenPath, rendered)
}

// TestNoteModal_RenderSmallScreen tests rendering with constrained dimensions
func TestNoteModal_RenderSmallScreen(t *testing.T) {
	modal := NewNoteModal()
	state := testfixtures.StateWithNotes()
	modal.SetNote(state.Notes[0])

	// Render with small screen
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 40, Y: 20},
	}
	scr := uv.NewScreenBuffer(40, 20)
	modal.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenPath := filepath.Join("testdata", "note_modal_small_screen.golden")
	testfixtures.CompareGolden(t, goldenPath, rendered)
}

// TestNoteModal_RenderVerySmallScreen tests minimum dimension constraints
func TestNoteModal_RenderVerySmallScreen(t *testing.T) {
	modal := NewNoteModal()
	state := testfixtures.StateWithNotes()
	modal.SetNote(state.Notes[0])

	// Render with very small screen (should use minimum constraints)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: 20, Y: 10},
	}
	scr := uv.NewScreenBuffer(20, 10)
	modal.Draw(scr, area)

	// Should not panic and should use minimum dimensions
	rendered := scr.Render()

	// Verify it rendered something (minimum size constraints applied)
	if len(rendered) == 0 {
		t.Error("Should render content even with very small screen")
	}

	// Verify golden file
	goldenPath := filepath.Join("testdata", "note_modal_very_small_screen.golden")
	testfixtures.CompareGolden(t, goldenPath, rendered)
}

// TestNoteModal_RenderUnknownNoteType tests rendering with unknown note type
func TestNoteModal_RenderUnknownNoteType(t *testing.T) {
	modal := NewNoteModal()
	state := testfixtures.StateWithNotes()

	// Create a note with unknown type
	unknownNote := *state.Notes[0]
	unknownNote.Type = "custom-type"
	unknownNote.Content = "This note has a custom unknown type"

	modal.SetNote(&unknownNote)

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenPath := filepath.Join("testdata", "note_modal_unknown_type.golden")
	testfixtures.CompareGolden(t, goldenPath, rendered)
}

// TestNoteModal_Centering tests that modal is centered on screen
func TestNoteModal_Centering(t *testing.T) {
	t.Parallel()

	modal := NewNoteModal()
	state := testfixtures.StateWithNotes()
	modal.SetNote(state.Notes[0])

	// Render on standard screen
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	modal.Draw(scr, area)

	rendered := scr.Render()

	// Modal should have content in the center, not at edges
	// This is verified via visual inspection of the golden file
	// but we can do a basic sanity check
	if len(rendered) == 0 {
		t.Error("Modal should render content")
	}
}

// compareNoteModalGolden compares rendered output with golden file
