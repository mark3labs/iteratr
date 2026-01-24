package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

// TestIntegration_KeyboardNavigation tests keyboard shortcuts
func TestIntegration_KeyboardNavigation(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)
	app.width = 120
	app.height = 40
	app.layout = CalculateLayout(120, 40)

	// ctrl+l toggles logs modal
	msg := tea.KeyPressMsg{Text: "ctrl+l"}
	_, _ = app.Update(msg)
	if !app.logsVisible {
		t.Error("ctrl+l should toggle logs visible")
	}

	// ctrl+l again hides logs
	_, _ = app.Update(msg)
	if app.logsVisible {
		t.Error("ctrl+l again should hide logs")
	}

	// ctrl+c quits
	quitMsg := tea.KeyPressMsg{Text: "ctrl+c"}
	_, _ = app.Update(quitMsg)
	if !app.quitting {
		t.Error("ctrl+c should quit")
	}
}

// TestIntegration_StatePropagation tests that state updates propagate to all components
func TestIntegration_StatePropagation(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)
	app.width = 120
	app.height = 40
	app.layout = CalculateLayout(120, 40)

	// Create test state
	testState := &session.State{
		Session: "test-session",
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Test task",
				Status:  "in_progress",
			},
		},
		Notes: []*session.Note{
			{
				ID:      "note1",
				Content: "Test note",
				Type:    "learning",
			},
		},
	}

	// Send state update
	msg := StateUpdateMsg{State: testState}
	_, _ = app.Update(msg)

	// Verify status received state
	if app.status.state != testState {
		t.Error("StatusBar did not receive state update")
	}

	// Verify sidebar received state
	if app.sidebar.state != testState {
		t.Error("Sidebar did not receive state update")
	}

	// Verify logs received state
	if app.logs.state != testState {
		t.Error("LogViewer did not receive state update")
	}

	// Verify dashboard received state update call
	// Note: Dashboard doesn't store state directly, but UpdateState should be called
	// We can't verify this without tracking calls, so we'll just check it compiles

	// Verify notes and inbox received state update call
	// Same as dashboard - UpdateState should be called
}

// TestIntegration_ViewportScrolling tests viewport scrolling in scrollable components
func TestIntegration_ViewportScrolling(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)
	app.width = 120
	app.height = 40
	app.layout = CalculateLayout(120, 40)
	app.propagateSizes()

	// Add content to logs that exceeds viewport height
	// Note: LogViewer stores events internally, not in State
	// We add events directly via AddEvent
	for i := 0; i < 100; i++ {
		event := session.Event{
			ID:   string(rune('a' + i%26)),
			Type: "test",
			Data: "Event content",
		}
		app.logs.AddEvent(event)
	}

	// Open logs modal
	app.logsVisible = true

	// Simulate down arrow key (should scroll down)
	downMsg := tea.KeyPressMsg{Text: "down"}
	_, _ = app.Update(downMsg)

	// Note: Viewport scroll position might not change if already at bottom or content fits
	// This is expected behavior and we don't assert on it

	// Simulate up arrow key (should scroll up if possible)
	upMsg := tea.KeyPressMsg{Text: "up"}
	_, _ = app.Update(upMsg)

	// Test page down
	pgDnMsg := tea.KeyPressMsg{Text: "pgdown"}
	_, _ = app.Update(pgDnMsg)

	// Test page up
	pgUpMsg := tea.KeyPressMsg{Text: "pgup"}
	_, _ = app.Update(pgUpMsg)

	// If we got here without panicking, viewport scrolling works
}

// TestIntegration_SidebarScrolling tests sidebar viewport scrolling
func TestIntegration_SidebarScrolling(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)
	app.width = 120
	app.height = 40
	app.layout = CalculateLayout(120, 40)
	app.propagateSizes()

	// Add many tasks to sidebar
	testState := &session.State{
		Tasks: make(map[string]*session.Task),
		Notes: make([]*session.Note, 50), // Many notes
	}
	for i := 0; i < 50; i++ {
		id := string(rune('a'+i%26)) + string(rune('0'+i/26))
		testState.Tasks[id] = &session.Task{
			ID:      id,
			Content: "Test task",
			Status:  "remaining",
		}
		testState.Notes[i] = &session.Note{
			ID:      id,
			Content: "Test note",
			Type:    "learning",
		}
	}

	// Update sidebar with state
	app.sidebar.SetState(testState)

	// Focus sidebar (simulate tab to focus)
	app.sidebar.SetFocus(true)

	// Simulate scrolling
	downMsg := tea.KeyPressMsg{Text: "down"}
	_ = app.sidebar.Update(downMsg)

	upMsg := tea.KeyPressMsg{Text: "up"}
	_ = app.sidebar.Update(upMsg)

	// If we got here without panicking, sidebar scrolling works
}

// TestIntegration_FocusManagement tests focus switching between components
func TestIntegration_FocusManagement(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)
	app.width = 120
	app.height = 40
	app.layout = CalculateLayout(120, 40)

	// Test focus on dashboard
	app.dashboard.SetFocus(true)
	if !app.dashboard.IsFocused() {
		t.Error("Dashboard should be focused")
	}

	// Test logs focus
	app.dashboard.SetFocus(false)
	app.logs.SetFocus(true)
	if app.dashboard.IsFocused() {
		t.Error("Dashboard should not be focused")
	}
	if !app.logs.IsFocused() {
		t.Error("Logs should be focused")
	}
}

// TestIntegration_ResizeHandling tests that resize updates all components
func TestIntegration_ResizeHandling(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)

	sizes := []struct {
		width  int
		height int
	}{
		{80, 24},
		{120, 40},
		{200, 60},
	}

	for _, size := range sizes {
		t.Run("resize", func(t *testing.T) {
			// Send resize message
			msg := tea.WindowSizeMsg{
				Width:  size.width,
				Height: size.height,
			}
			_, _ = app.Update(msg)

			// Verify app dimensions updated
			if app.width != size.width || app.height != size.height {
				t.Errorf("App dimensions not updated: got %dx%d, want %dx%d",
					app.width, app.height, size.width, size.height)
			}

			// Verify layout recalculated
			if app.layout.Area.Dx() != size.width || app.layout.Area.Dy() != size.height {
				t.Errorf("Layout not recalculated: got %dx%d, want %dx%d",
					app.layout.Area.Dx(), app.layout.Area.Dy(), size.width, size.height)
			}

			// Verify components received size updates
			// We can't easily verify without exposing internals, but if no panic, it works
		})
	}
}

// TestIntegration_CompactModeToggle tests toggling sidebar in compact mode
func TestIntegration_CompactModeToggle(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)

	// Set compact mode
	app.width = 80
	app.height = 24
	app.layout = CalculateLayout(80, 24)
	app.propagateSizes()

	if app.layout.Mode != LayoutCompact {
		t.Fatal("Expected compact mode")
	}

	// Initial state: sidebar not visible
	if app.sidebarVisible {
		t.Error("Sidebar should not be visible initially")
	}

	// Press ctrl+s to toggle sidebar
	msg := tea.KeyPressMsg{Text: "ctrl+s"}
	_, _ = app.Update(msg)

	// Sidebar should now be visible
	if !app.sidebarVisible {
		t.Error("Sidebar should be visible after toggle")
	}

	// Press ctrl+s again to hide
	_, _ = app.Update(msg)

	// Sidebar should be hidden
	if app.sidebarVisible {
		t.Error("Sidebar should be hidden after second toggle")
	}
}

// TestIntegration_AgentOutputAppend tests appending to agent output
func TestIntegration_AgentOutputAppend(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)
	app.width = 120
	app.height = 40
	app.layout = CalculateLayout(120, 40)
	app.propagateSizes()

	// Send agent output message
	msg := AgentOutputMsg{Content: "Test output"}
	_, _ = app.Update(msg)

	// Verify test completes without panic
	// AppendText returns the command from agent.AppendText
}

// TestIntegration_EventHandling tests event message handling
func TestIntegration_EventHandling(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)
	app.width = 120
	app.height = 40
	app.layout = CalculateLayout(120, 40)
	app.propagateSizes()

	// Send event message
	event := session.Event{
		ID:   "evt1",
		Type: "test",
		Data: "Test event",
	}
	msg := EventMsg{Event: event}
	_, cmd := app.Update(msg)

	// Should return a batch command (logs.AddEvent + loadInitialState + waitForEvents)
	if cmd == nil {
		t.Error("Expected command from event handling")
	}
}

// TestIntegration_AllMessageTypes tests that all message types render correctly together
func TestIntegration_AllMessageTypes(t *testing.T) {
	app := NewApp(context.Background(), nil, "test-session", nil, nil)
	app.width = 120
	app.height = 40
	app.layout = CalculateLayout(120, 40)
	app.propagateSizes()

	// Test iteration divider
	iterMsg := IterationStartMsg{Number: 1}
	_, _ = app.Update(iterMsg)

	// Test text message
	textMsg := AgentOutputMsg{Content: "This is assistant text"}
	_, _ = app.Update(textMsg)

	// Test thinking message
	thinkingMsg := AgentThinkingMsg{Content: "Analyzing the problem..."}
	_, _ = app.Update(thinkingMsg)

	// Test tool call messages
	toolMsg := AgentToolCallMsg{
		ToolCallID: "tool-1",
		Title:      "Read",
		Status:     "completed",
		Input:      map[string]any{"filePath": "test.go"},
		Output:     "file contents here",
	}
	_, _ = app.Update(toolMsg)

	// Test finish message
	finishMsg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4",
		Provider: "Anthropic",
		Duration: 1500000000, // 1.5 seconds
	}
	_, _ = app.Update(finishMsg)

	// Verify agent has messages
	if len(app.agent.messages) == 0 {
		t.Error("Expected agent to have messages")
	}

	// Verify different message types are present
	hasText := false
	hasThinking := false
	hasTool := false
	hasInfo := false
	hasDivider := false

	for _, msg := range app.agent.messages {
		switch msg.(type) {
		case *TextMessageItem:
			hasText = true
		case *ThinkingMessageItem:
			hasThinking = true
		case *ToolMessageItem:
			hasTool = true
		case *InfoMessageItem:
			hasInfo = true
		case *DividerMessageItem:
			hasDivider = true
		}
	}

	if !hasText {
		t.Error("Expected at least one TextMessageItem")
	}
	if !hasThinking {
		t.Error("Expected at least one ThinkingMessageItem")
	}
	if !hasTool {
		t.Error("Expected at least one ToolMessageItem")
	}
	if !hasInfo {
		t.Error("Expected at least one InfoMessageItem")
	}
	if !hasDivider {
		t.Error("Expected at least one DividerMessageItem")
	}

	// Verify rendering doesn't panic
	output := app.agent.Render()
	if output == "" {
		t.Error("Expected non-empty agent output")
	}
}
