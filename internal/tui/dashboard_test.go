package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

// TestDashboard_Render removed - Dashboard now uses Draw() method with Screen/Draw pattern
// Rendering is tested through integration tests in app_test.go

func TestDashboard_RenderSessionInfo(t *testing.T) {
	d := &Dashboard{
		sessionName: "my-session",
		iteration:   42,
		sidebar:     NewSidebar(),
	}

	output := d.renderSessionInfo()
	if output == "" {
		t.Error("expected non-empty session info")
	}

	// Basic smoke test - should contain session name
	// We don't want to test lipgloss styling details, just that content is present
	// Note: lipgloss styles are stripped in tests, so we just check structure
}

func TestDashboard_GetTaskStats(t *testing.T) {
	tests := []struct {
		name     string
		state    *session.State
		wantZero bool
		expected progressStats
	}{
		{
			name: "counts tasks correctly",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "remaining"},
					"t2": {ID: "t2", Status: "in_progress"},
					"t3": {ID: "t3", Status: "completed"},
					"t4": {ID: "t4", Status: "blocked"},
					"t5": {ID: "t5", Status: "completed"},
				},
			},
			expected: progressStats{
				Total:      5,
				Remaining:  1,
				InProgress: 1,
				Completed:  2,
				Blocked:    1,
			},
		},
		{
			name:     "handles empty task list",
			state:    &session.State{Tasks: map[string]*session.Task{}},
			wantZero: true,
			expected: progressStats{
				Total:      0,
				Remaining:  0,
				InProgress: 0,
				Completed:  0,
				Blocked:    0,
			},
		},
		{
			name: "handles only completed tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "completed"},
					"t2": {ID: "t2", Status: "completed"},
				},
			},
			expected: progressStats{
				Total:      2,
				Remaining:  0,
				InProgress: 0,
				Completed:  2,
				Blocked:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dashboard{state: tt.state}
			stats := d.getTaskStats()

			if stats.Total != tt.expected.Total {
				t.Errorf("Total: got %d, want %d", stats.Total, tt.expected.Total)
			}
			if stats.Remaining != tt.expected.Remaining {
				t.Errorf("Remaining: got %d, want %d", stats.Remaining, tt.expected.Remaining)
			}
			if stats.InProgress != tt.expected.InProgress {
				t.Errorf("InProgress: got %d, want %d", stats.InProgress, tt.expected.InProgress)
			}
			if stats.Completed != tt.expected.Completed {
				t.Errorf("Completed: got %d, want %d", stats.Completed, tt.expected.Completed)
			}
			if stats.Blocked != tt.expected.Blocked {
				t.Errorf("Blocked: got %d, want %d", stats.Blocked, tt.expected.Blocked)
			}
		})
	}
}

func TestDashboard_RenderProgressIndicator(t *testing.T) {
	tests := []struct {
		name  string
		state *session.State
	}{
		{
			name: "renders progress with tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "completed"},
					"t2": {ID: "t2", Status: "remaining"},
				},
			},
		},
		{
			name: "renders progress with no tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{},
			},
		},
		{
			name: "renders progress with all completed",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "completed"},
					"t2": {ID: "t2", Status: "completed"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dashboard{state: tt.state}
			output := d.renderProgressIndicator()
			if output == "" {
				t.Error("expected non-empty progress indicator")
			}
		})
	}
}

// TestDashboard_RenderCurrentTask removed - renderCurrentTask() method no longer exists
// Current task is now shown in StatusBar component

func TestDashboard_UpdateState(t *testing.T) {
	d := NewDashboard(nil, NewSidebar())

	state := &session.State{
		Session: "new-session",
		Tasks:   map[string]*session.Task{},
	}

	d.UpdateState(state)

	if d.state != state {
		t.Error("state was not updated")
	}
	if d.sessionName != "new-session" {
		t.Errorf("session name: got %s, want new-session", d.sessionName)
	}
}

func TestDashboard_SetIteration(t *testing.T) {
	d := NewDashboard(nil, NewSidebar())

	d.SetIteration(10)

	if d.iteration != 10 {
		t.Errorf("iteration: got %d, want 10", d.iteration)
	}
}

func TestDashboard_UpdateSize(t *testing.T) {
	d := NewDashboard(nil, NewSidebar())

	d.UpdateSize(100, 50)

	if d.width != 100 {
		t.Errorf("width: got %d, want 100", d.width)
	}
	if d.height != 50 {
		t.Errorf("height: got %d, want 50", d.height)
	}
}

func TestNewDashboard(t *testing.T) {
	d := NewDashboard(nil, NewSidebar())

	if d == nil {
		t.Fatal("expected non-nil dashboard")
	}
	if d.sessionName != "" {
		t.Errorf("expected empty session name, got %s", d.sessionName)
	}
	if d.iteration != 0 {
		t.Errorf("expected iteration 0, got %d", d.iteration)
	}
}

// TestDashboard_RenderTaskStats removed - renderTaskStats() method no longer exists
// Task stats are shown in Sidebar component and renderProgressIndicator()

func TestDashboard_UserInputMsgOnEnter(t *testing.T) {
	// Create a new agent output and dashboard
	agentOutput := NewAgentOutput()
	d := NewDashboard(agentOutput, NewSidebar())
	d.UpdateSize(100, 50)

	// Set focus to input pane
	d.focusPane = FocusInput
	d.agentOutput.SetInputFocused(true)

	// Set some input text
	d.agentOutput.input.SetValue("test message")

	// Verify input has text
	if d.agentOutput.InputValue() != "test message" {
		t.Errorf("expected input value 'test message', got %q", d.agentOutput.InputValue())
	}

	// Simulate Enter key press
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	cmdFunc := d.Update(enterKey)

	// Verify cmd is not nil (UserInputMsg is returned)
	if cmdFunc == nil {
		t.Fatal("expected cmd to be non-nil (UserInputMsg should be emitted)")
	}

	// Execute the command to get the message
	msg := cmdFunc()

	// Verify the message is UserInputMsg
	userMsg, ok := msg.(UserInputMsg)
	if !ok {
		t.Fatalf("expected UserInputMsg, got %T", msg)
	}

	// Verify the message contains the input text
	if userMsg.Text != "test message" {
		t.Errorf("expected UserInputMsg.Text 'test message', got %q", userMsg.Text)
	}

	// Verify input was reset
	if d.agentOutput.InputValue() != "" {
		t.Errorf("expected input to be reset, got %q", d.agentOutput.InputValue())
	}

	// Verify focus was returned to agent
	if d.focusPane != FocusAgent {
		t.Errorf("expected focusPane to be FocusAgent, got %d", d.focusPane)
	}

	// Verify input is unfocused
	if d.agentOutput.input.Focused() {
		t.Error("expected input to be unfocused after Enter")
	}
}

func TestDashboard_UserInputMsgQueuedWhenBusy(t *testing.T) {
	// Create a new agent output and dashboard
	agentOutput := NewAgentOutput()
	d := NewDashboard(agentOutput, NewSidebar())
	d.UpdateSize(100, 50)

	// Set agent as busy
	d.agentBusy = true

	// Set focus to input pane
	d.focusPane = FocusInput
	d.agentOutput.SetInputFocused(true)

	// Set some input text
	d.agentOutput.input.SetValue("queued message")

	// Simulate Enter key press
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	cmdFunc := d.Update(enterKey)

	// Verify cmd is nil (message is queued, not emitted)
	if cmdFunc != nil {
		t.Error("expected cmd to be nil when agent is busy (message should be queued)")
	}

	// Verify message was queued
	if d.queuedMsg != "queued message" {
		t.Errorf("expected queuedMsg 'queued message', got %q", d.queuedMsg)
	}

	// Verify input was reset
	if d.agentOutput.InputValue() != "" {
		t.Errorf("expected input to be reset, got %q", d.agentOutput.InputValue())
	}

	// Verify focus was returned to agent
	if d.focusPane != FocusAgent {
		t.Errorf("expected focusPane to be FocusAgent, got %d", d.focusPane)
	}
}

func TestDashboard_EmptyInputNoMessage(t *testing.T) {
	// Create a new agent output and dashboard
	agentOutput := NewAgentOutput()
	d := NewDashboard(agentOutput, NewSidebar())
	d.UpdateSize(100, 50)

	// Set focus to input pane
	d.focusPane = FocusInput
	d.agentOutput.SetInputFocused(true)

	// Leave input empty (no text)

	// Simulate Enter key press
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	cmdFunc := d.Update(enterKey)

	// Verify cmd is nil (no message emitted for empty input)
	if cmdFunc != nil {
		t.Error("expected cmd to be nil for empty input")
	}

	// Verify focus remains on input (empty input doesn't exit)
	if d.focusPane != FocusInput {
		t.Errorf("expected focusPane to remain FocusInput, got %d", d.focusPane)
	}
}

func TestDashboard_InputFocusWithIKey(t *testing.T) {
	// Create a new agent output and dashboard
	agentOutput := NewAgentOutput()
	d := NewDashboard(agentOutput, NewSidebar())
	d.UpdateSize(100, 50)

	// Initially focus should be on agent
	if d.focusPane != FocusAgent {
		t.Errorf("expected initial focusPane to be FocusAgent, got %d", d.focusPane)
	}
	if d.agentOutput.input.Focused() {
		t.Error("expected input to be unfocused initially")
	}

	// Press 'i' key to focus input
	iKey := tea.KeyPressMsg{Text: "i"}
	d.Update(iKey)

	// Verify focus moved to input
	if d.focusPane != FocusInput {
		t.Errorf("expected focusPane to be FocusInput after 'i', got %d", d.focusPane)
	}
	if !d.agentOutput.input.Focused() {
		t.Error("expected input to be focused after 'i'")
	}
	if !d.inputFocused {
		t.Error("expected inputFocused to be true after 'i'")
	}
}

func TestDashboard_InputFocusFromTasksPane(t *testing.T) {
	// Create a new agent output and dashboard
	agentOutput := NewAgentOutput()
	d := NewDashboard(agentOutput, NewSidebar())
	d.UpdateSize(100, 50)

	// Set focus to tasks pane
	d.focusPane = FocusTasks

	// Press 'i' key from tasks pane
	iKey := tea.KeyPressMsg{Text: "i"}
	d.Update(iKey)

	// Verify focus moved to input
	if d.focusPane != FocusInput {
		t.Errorf("expected focusPane to be FocusInput after 'i' from tasks, got %d", d.focusPane)
	}
	if !d.agentOutput.input.Focused() {
		t.Error("expected input to be focused after 'i' from tasks")
	}
}

func TestDashboard_InputBlurWithEscape(t *testing.T) {
	// Create a new agent output and dashboard
	agentOutput := NewAgentOutput()
	d := NewDashboard(agentOutput, NewSidebar())
	d.UpdateSize(100, 50)

	// Focus input
	d.focusPane = FocusInput
	d.agentOutput.SetInputFocused(true)
	d.inputFocused = true

	// Add some text to input
	d.agentOutput.input.SetValue("some text")

	// Verify input is focused
	if !d.agentOutput.input.Focused() {
		t.Fatal("expected input to be focused before Escape")
	}

	// Press Escape key
	escKey := tea.KeyPressMsg{Text: "esc"}
	d.Update(escKey)

	// Verify focus returned to agent
	if d.focusPane != FocusAgent {
		t.Errorf("expected focusPane to be FocusAgent after Escape, got %d", d.focusPane)
	}
	if d.agentOutput.input.Focused() {
		t.Error("expected input to be unfocused after Escape")
	}
	if d.inputFocused {
		t.Error("expected inputFocused to be false after Escape")
	}

	// Verify input text is preserved (Escape doesn't clear input)
	if d.agentOutput.InputValue() != "some text" {
		t.Errorf("expected input value 'some text' after Escape, got %q", d.agentOutput.InputValue())
	}
}

func TestDashboard_InputFocusIdempotent(t *testing.T) {
	// Create a new agent output and dashboard
	agentOutput := NewAgentOutput()
	d := NewDashboard(agentOutput, NewSidebar())
	d.UpdateSize(100, 50)

	// Focus input once
	iKey := tea.KeyPressMsg{Text: "i"}
	d.Update(iKey)

	// Verify input is focused
	if d.focusPane != FocusInput {
		t.Fatal("expected focusPane to be FocusInput")
	}

	// Press 'i' again while already focused - should be idempotent
	d.Update(iKey)

	// Verify focus remains on input
	if d.focusPane != FocusInput {
		t.Errorf("expected focusPane to remain FocusInput, got %d", d.focusPane)
	}
	if !d.agentOutput.input.Focused() {
		t.Error("expected input to remain focused")
	}
}
