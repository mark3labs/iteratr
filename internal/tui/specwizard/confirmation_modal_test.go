package specwizard

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfirmationModal(t *testing.T) {
	modal := NewConfirmationModal("Test Title", "Test message")

	assert.NotNil(t, modal)
	assert.Equal(t, "Test Title", modal.title)
	assert.Equal(t, "Test message", modal.message)
	assert.False(t, modal.visible, "modal should start hidden")
}

func TestConfirmationModal_ShowHide(t *testing.T) {
	modal := NewConfirmationModal("Test", "Message")

	// Initially hidden
	assert.False(t, modal.IsVisible())

	// Show
	modal.Show()
	assert.True(t, modal.IsVisible())

	// Hide
	modal.Hide()
	assert.False(t, modal.IsVisible())

	// Show again
	modal.Show()
	assert.True(t, modal.IsVisible())
}

func TestConfirmationModal_Render(t *testing.T) {
	modal := NewConfirmationModal("Cancel Operation?", "This will discard all changes.")

	// Render modal
	rendered := modal.Render()

	// Verify content
	assert.Contains(t, rendered, "Cancel Operation", "should contain title")
	assert.Contains(t, rendered, "This will discard all changes", "should contain message")
	assert.Contains(t, rendered, "Press Y to confirm", "should contain instructions")
	assert.Contains(t, rendered, "⚠", "should contain warning icon")
}

func TestRenderConfirmationModal_Standalone(t *testing.T) {
	rendered := RenderConfirmationModal("Test Title", "Test message content")

	// Verify all required elements are present
	assert.Contains(t, rendered, "Test Title", "should contain title")
	assert.Contains(t, rendered, "Test message content", "should contain message")
	assert.Contains(t, rendered, "Press Y to confirm, N or ESC to cancel", "should contain button instructions")
	assert.Contains(t, rendered, "⚠", "should contain warning icon")
}

func TestRenderConfirmationModal_MultilineMessage(t *testing.T) {
	message := "Line 1\nLine 2\nLine 3"
	rendered := RenderConfirmationModal("Title", message)

	// Should preserve multiline message
	assert.Contains(t, rendered, "Line 1")
	assert.Contains(t, rendered, "Line 2")
	assert.Contains(t, rendered, "Line 3")
}

func TestRenderConfirmationModal_LongTitle(t *testing.T) {
	longTitle := "This is a very long title that should still render correctly"
	rendered := RenderConfirmationModal(longTitle, "Message")

	// Long titles may be wrapped, so check for partial content
	assert.Contains(t, rendered, "This is a very long title", "should handle long titles")
}

func TestRenderConfirmationModal_EmptyStrings(t *testing.T) {
	// Should handle empty title and message gracefully
	rendered := RenderConfirmationModal("", "")

	// Should still render modal structure with buttons
	assert.Contains(t, rendered, "Press Y to confirm", "should contain button instructions")
	assert.Contains(t, rendered, "⚠", "should contain warning icon")
}

func TestRenderConfirmationModal_SpecialCharacters(t *testing.T) {
	title := "Cancel? (Yes/No)"
	message := "This will delete files"
	rendered := RenderConfirmationModal(title, message)

	assert.Contains(t, rendered, title, "should handle special characters in title")
	assert.Contains(t, rendered, message, "should handle special characters in message")
}

func TestRenderConfirmationModal_Structure(t *testing.T) {
	rendered := RenderConfirmationModal("Title", "Message")

	// Should have border characters
	assert.True(t, strings.Contains(rendered, "─") || strings.Contains(rendered, "│"),
		"should contain border characters")

	// Should be non-empty
	assert.NotEmpty(t, rendered, "should render non-empty modal")

	// Should be multi-line (title, message, blank line, buttons, plus borders/padding)
	lines := strings.Split(rendered, "\n")
	assert.Greater(t, len(lines), 5, "should have multiple lines for structure")
}

func TestConfirmationModal_MultipleInstances(t *testing.T) {
	modal1 := NewConfirmationModal("Title 1", "Message 1")
	modal2 := NewConfirmationModal("Title 2", "Message 2")

	// Show first modal
	modal1.Show()
	assert.True(t, modal1.IsVisible())
	assert.False(t, modal2.IsVisible())

	// Show second modal
	modal2.Show()
	assert.True(t, modal1.IsVisible())
	assert.True(t, modal2.IsVisible())

	// Hide first modal
	modal1.Hide()
	assert.False(t, modal1.IsVisible())
	assert.True(t, modal2.IsVisible())

	// Verify content is independent
	rendered1 := modal1.Render()
	rendered2 := modal2.Render()
	assert.Contains(t, rendered1, "Title 1")
	assert.Contains(t, rendered2, "Title 2")
	assert.NotContains(t, rendered1, "Title 2")
	assert.NotContains(t, rendered2, "Title 1")
}
