package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

// NotesPanel displays notes grouped by type with color-coding.
type NotesPanel struct {
	state  *session.State
	width  int
	height int
}

// NewNotesPanel creates a new NotesPanel component.
func NewNotesPanel() *NotesPanel {
	return &NotesPanel{}
}

// Update handles messages for the notes panel.
func (n *NotesPanel) Update(msg tea.Msg) tea.Cmd {
	// TODO: Implement notes panel updates
	return nil
}

// Render returns the notes panel view as a string.
func (n *NotesPanel) Render() string {
	// TODO: Implement notes panel rendering with lipgloss
	return "Notes Panel (TODO)"
}

// UpdateSize updates the notes panel dimensions.
func (n *NotesPanel) UpdateSize(width, height int) tea.Cmd {
	n.width = width
	n.height = height
	return nil
}

// UpdateState updates the notes panel with new session state.
func (n *NotesPanel) UpdateState(state *session.State) tea.Cmd {
	n.state = state
	return nil
}
