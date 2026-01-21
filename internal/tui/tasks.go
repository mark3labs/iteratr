package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

// TaskList displays tasks grouped by status with filtering and navigation.
type TaskList struct {
	state  *session.State
	width  int
	height int
}

// NewTaskList creates a new TaskList component.
func NewTaskList() *TaskList {
	return &TaskList{}
}

// Update handles messages for the task list.
func (t *TaskList) Update(msg tea.Msg) tea.Cmd {
	// TODO: Implement task list updates (j/k navigation, filtering)
	return nil
}

// Render returns the task list view as a string.
func (t *TaskList) Render() string {
	// TODO: Implement task list rendering with lipgloss
	return "Task List view (TODO)"
}

// UpdateSize updates the task list dimensions.
func (t *TaskList) UpdateSize(width, height int) tea.Cmd {
	t.width = width
	t.height = height
	return nil
}

// UpdateState updates the task list with new session state.
func (t *TaskList) UpdateState(state *session.State) tea.Cmd {
	t.state = state
	return nil
}
