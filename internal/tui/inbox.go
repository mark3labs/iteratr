package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

// InboxPanel displays unread messages and provides an input field for sending.
type InboxPanel struct {
	state  *session.State
	width  int
	height int
}

// NewInboxPanel creates a new InboxPanel component.
func NewInboxPanel() *InboxPanel {
	return &InboxPanel{}
}

// Update handles messages for the inbox panel.
func (i *InboxPanel) Update(msg tea.Msg) tea.Cmd {
	// TODO: Implement inbox panel updates (input handling)
	return nil
}

// Render returns the inbox panel view as a string.
func (i *InboxPanel) Render() string {
	// TODO: Implement inbox panel rendering with lipgloss
	return "Inbox Panel (TODO)"
}

// UpdateSize updates the inbox panel dimensions.
func (i *InboxPanel) UpdateSize(width, height int) tea.Cmd {
	i.width = width
	i.height = height
	return nil
}

// UpdateState updates the inbox panel with new session state.
func (i *InboxPanel) UpdateState(state *session.State) tea.Cmd {
	i.state = state
	return nil
}
