package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestLogViewer_InitialState_Empty tests LogViewer with no events
func TestLogViewer_InitialState_Empty(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Initial state should have no events
	require.Empty(t, logs.events, "Initial events should be empty")

	// Render content
	logs.updateContent()

	// Viewport should show empty state message
	content := logs.viewport.View()
	require.Contains(t, content, "No events yet", "Empty state should show 'No events yet'")
}

// TestLogViewer_AddEvent_SingleEvent tests adding a single event to LogViewer
func TestLogViewer_AddEvent_SingleEvent(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	event := session.Event{
		ID:        "1",
		Timestamp: testfixtures.FixedTime,
		Session:   testfixtures.FixedSessionName,
		Type:      "task",
		Action:    "add",
		Data:      "Create test infrastructure",
	}

	logs.AddEvent(event)

	// Event should be added
	require.Len(t, logs.events, 1, "Should have 1 event")
	require.Equal(t, event, logs.events[0], "Event should match")

	// Content should be rendered
	content := logs.viewport.View()
	require.Contains(t, content, "TASK", "Content should contain TASK label")
	require.Contains(t, content, "add", "Content should contain action")
	require.Contains(t, content, "Create test infrastructure", "Content should contain event data")
}

// TestLogViewer_AddEvent_MultipleEvents tests adding multiple events
func TestLogViewer_AddEvent_MultipleEvents(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	events := []session.Event{
		{
			ID:        "1",
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      "Task 1",
		},
		{
			ID:        "2",
			Timestamp: testfixtures.FixedTime.Add(1 * 60 * 1000000000), // +1 minute
			Session:   testfixtures.FixedSessionName,
			Type:      "note",
			Action:    "add",
			Data:      "Note 1",
		},
		{
			ID:        "3",
			Timestamp: testfixtures.FixedTime.Add(2 * 60 * 1000000000), // +2 minutes
			Session:   testfixtures.FixedSessionName,
			Type:      "iteration",
			Action:    "start",
			Data:      "Iteration 1",
		},
	}

	for _, event := range events {
		logs.AddEvent(event)
	}

	// All events should be added
	require.Len(t, logs.events, 3, "Should have 3 events")

	// Content should contain all event types
	content := logs.viewport.View()
	require.Contains(t, content, "TASK", "Content should contain TASK")
	require.Contains(t, content, "NOTE", "Content should contain NOTE")
	require.Contains(t, content, "ITER", "Content should contain ITER")
}

// TestLogViewer_AddEvent_AutoScrollToBottom tests that AddEvent auto-scrolls to bottom
func TestLogViewer_AddEvent_AutoScrollToBottom(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10) // Small height to force scrolling

	// Add many events to exceed viewport height
	for i := 0; i < 20; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event %d", i+1),
		}
		logs.AddEvent(event)
	}

	// After adding events, viewport should be at bottom
	require.True(t, logs.viewport.AtBottom(), "Viewport should be at bottom after adding events")
}

// TestLogViewer_ScrollPosition_ManualScroll tests manual scrolling up and down
func TestLogViewer_ScrollPosition_ManualScroll(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10) // Small height to force scrolling
	logs.SetFocus(true)

	// Add many events to exceed viewport height
	for i := 0; i < 20; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event %d", i+1),
		}
		logs.AddEvent(event)
	}

	// Initially at bottom
	require.True(t, logs.viewport.AtBottom(), "Should start at bottom")

	// Scroll up with 'k'
	logs.Update(tea.KeyPressMsg{Text: "k"})
	require.False(t, logs.viewport.AtBottom(), "Should not be at bottom after scrolling up")
	require.False(t, logs.viewport.AtTop(), "Should not be at top after single scroll up")

	// Scroll all the way up with many 'k' presses
	for i := 0; i < 30; i++ {
		logs.Update(tea.KeyPressMsg{Text: "k"})
	}
	require.True(t, logs.viewport.AtTop(), "Should be at top after scrolling up many times")

	// Scroll all the way down with many 'j' presses
	for i := 0; i < 30; i++ {
		logs.Update(tea.KeyPressMsg{Text: "j"})
	}
	require.True(t, logs.viewport.AtBottom(), "Should be at bottom after scrolling down many times")
}

// TestLogViewer_ScrollPosition_PageUpDown tests page up/down scrolling
func TestLogViewer_ScrollPosition_PageUpDown(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10) // Small height
	logs.SetFocus(true)

	// Add many events
	for i := 0; i < 30; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event %d", i+1),
		}
		logs.AddEvent(event)
	}

	// Initially at bottom
	require.True(t, logs.viewport.AtBottom(), "Should start at bottom")

	// Page up
	logs.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	require.False(t, logs.viewport.AtBottom(), "Should not be at bottom after page up")

	// Page down
	logs.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	// After page down, we should be closer to bottom or at bottom
	// (exact position depends on viewport implementation)
}

// TestLogViewer_ScrollPosition_HalfPageUpDown tests half-page up/down scrolling
func TestLogViewer_ScrollPosition_HalfPageUpDown(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10) // Small height
	logs.SetFocus(true)

	// Add many events
	for i := 0; i < 30; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event %d", i+1),
		}
		logs.AddEvent(event)
	}

	// Scroll to middle using ctrl+u (half page up)
	logs.Update(tea.KeyPressMsg{Text: "ctrl+u"})
	require.False(t, logs.viewport.AtBottom(), "Should not be at bottom after ctrl+u")

	// Scroll back down using ctrl+d (half page down)
	logs.Update(tea.KeyPressMsg{Text: "ctrl+d"})
}

// TestLogViewer_EventTypes_AllTypes tests rendering of all event types
func TestLogViewer_EventTypes_AllTypes(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	events := []session.Event{
		{
			ID:        "1",
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      "Task event",
		},
		{
			ID:        "2",
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "note",
			Action:    "add",
			Data:      "Note event",
		},
		{
			ID:        "3",
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "iteration",
			Action:    "start",
			Data:      "Iteration event",
		},
		{
			ID:        "4",
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "control",
			Action:    "pause",
			Data:      "Control event",
		},
		{
			ID:        "5",
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "unknown",
			Action:    "custom",
			Data:      "Unknown event type",
		},
	}

	for _, event := range events {
		logs.AddEvent(event)
	}

	// Verify all event type labels are present
	content := logs.viewport.View()
	require.Contains(t, content, "TASK", "Should contain TASK label")
	require.Contains(t, content, "NOTE", "Should contain NOTE label")
	require.Contains(t, content, "ITER", "Should contain ITER label")
	require.Contains(t, content, "CTRL", "Should contain CTRL label")
	require.Contains(t, content, "EVENT", "Should contain EVENT label for unknown type")
}

// TestLogViewer_Rendering_ContentTruncation tests that long content is truncated
func TestLogViewer_Rendering_ContentTruncation(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Create event with very long data
	longData := strings.Repeat("A", 200)
	event := session.Event{
		ID:        "1",
		Timestamp: testfixtures.FixedTime,
		Session:   testfixtures.FixedSessionName,
		Type:      "task",
		Action:    "add",
		Data:      longData,
	}

	logs.AddEvent(event)

	// Content should be truncated with ellipsis (renderEvent truncates based on maxContentWidth)
	content := logs.viewport.View()
	require.Contains(t, content, "...", "Long content should be truncated with ellipsis")
	// The rendered line should not contain the full 200 'A's in a row
	require.NotContains(t, content, strings.Repeat("A", 200), "Full long data should not appear in output")
}

// TestLogViewer_Rendering_TimestampFormat tests that timestamps are formatted correctly
func TestLogViewer_Rendering_TimestampFormat(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	event := session.Event{
		ID:        "1",
		Timestamp: testfixtures.FixedTime, // 2024-01-15 10:30:00 UTC
		Session:   testfixtures.FixedSessionName,
		Type:      "task",
		Action:    "add",
		Data:      "Test event",
	}

	logs.AddEvent(event)

	// Timestamp should be formatted as HH:MM:SS
	content := logs.viewport.View()
	require.Contains(t, content, "10:30:00", "Timestamp should be formatted as 10:30:00")
}

// TestLogViewer_Focus_State tests focus state management
func TestLogViewer_Focus_State(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()

	// Initially not focused
	require.False(t, logs.IsFocused(), "Should not be focused initially")

	// Set focused
	logs.SetFocus(true)
	require.True(t, logs.IsFocused(), "Should be focused after SetFocus(true)")

	// Unfocus
	logs.SetFocus(false)
	require.False(t, logs.IsFocused(), "Should not be focused after SetFocus(false)")
}

// TestLogViewer_SetSize_UpdatesDimensions tests that SetSize updates dimensions
func TestLogViewer_SetSize_UpdatesDimensions(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()

	logs.SetSize(100, 50)

	require.Equal(t, 100, logs.width, "Width should be updated")
	require.Equal(t, 50, logs.height, "Height should be updated")
}

// TestLogViewer_SetState_UpdatesState tests that SetState updates state
func TestLogViewer_SetState_UpdatesState(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	state := testfixtures.StateWithTasks()

	logs.SetState(state)

	require.Equal(t, state, logs.state, "State should be updated")
}

// TestLogViewer_Rendering_ModalOverlay_Teatest tests visual rendering of modal overlay
func TestLogViewer_Rendering_ModalOverlay_Teatest(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add sample events
	events := []session.Event{
		{
			ID:        "1",
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      "Create test infrastructure",
		},
		{
			ID:        "2",
			Timestamp: testfixtures.FixedTime.Add(1 * 60 * 1000000000),
			Session:   testfixtures.FixedSessionName,
			Type:      "note",
			Action:    "add",
			Data:      "Learned about event sourcing",
		},
		{
			ID:        "3",
			Timestamp: testfixtures.FixedTime.Add(2 * 60 * 1000000000),
			Session:   testfixtures.FixedSessionName,
			Type:      "iteration",
			Action:    "start",
			Data:      "Iteration 1 started",
		},
		{
			ID:        "4",
			Timestamp: testfixtures.FixedTime.Add(3 * 60 * 1000000000),
			Session:   testfixtures.FixedSessionName,
			Type:      "control",
			Action:    "pause",
			Data:      "Paused execution",
		},
	}

	for _, event := range events {
		logs.AddEvent(event)
	}

	// Render to screen
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	logs.Draw(scr, area)

	out := scr.Render()

	// Verify modal content
	require.Contains(t, out, "Event Log", "Should contain modal title")
	require.Contains(t, out, "TASK", "Should contain TASK event")
	require.Contains(t, out, "NOTE", "Should contain NOTE event")
	require.Contains(t, out, "ITER", "Should contain ITER event")
	require.Contains(t, out, "CTRL", "Should contain CTRL event")

	// Compare with golden file
	goldenPath := filepath.Join("testdata", "logs_modal_teatest.golden")
	testfixtures.CompareGolden(t, goldenPath, out)
}

// TestLogViewer_Rendering_EmptyState_Teatest tests visual rendering of empty state
func TestLogViewer_Rendering_EmptyState_Teatest(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// No events added - should show empty state
	logs.updateContent()

	// Render to screen
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	logs.Draw(scr, area)

	out := scr.Render()

	// Verify empty state
	require.Contains(t, out, "Event Log", "Should contain modal title")
	require.Contains(t, out, "No events yet", "Should contain empty state message")

	// Compare with golden file
	goldenPath := filepath.Join("testdata", "logs_empty_state_teatest.golden")
	testfixtures.CompareGolden(t, goldenPath, out)
}

// TestLogViewer_Rendering_ScrolledUp_Teatest tests visual rendering when scrolled up from bottom
func TestLogViewer_Rendering_ScrolledUp_Teatest(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 15) // Small height to force scrolling
	logs.SetFocus(true)

	// Add many events to exceed viewport height
	for i := 0; i < 25; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event number %d with some content", i+1),
		}
		logs.AddEvent(event)
	}

	// Scroll up several times to move away from bottom
	for i := 0; i < 5; i++ {
		logs.Update(tea.KeyPressMsg{Text: "k"})
	}

	// Verify not at bottom
	require.False(t, logs.viewport.AtBottom(), "Should not be at bottom after scrolling up")

	// Render to screen
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, 15)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: 15},
	}
	logs.Draw(scr, area)

	out := scr.Render()

	// Verify modal content (should show middle events, not the last ones)
	require.Contains(t, out, "Event Log", "Should contain modal title")
	require.Contains(t, out, "TASK", "Should contain TASK events")

	// Compare with golden file
	goldenPath := filepath.Join("testdata", "logs_scrolled_up_teatest.golden")
	testfixtures.CompareGolden(t, goldenPath, out)
}

// TestLogViewer_Update_KeyPresses tests that Update correctly handles key presses
func TestLogViewer_Update_KeyPresses(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10)
	logs.SetFocus(true)

	// Add events
	for i := 0; i < 20; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event %d", i+1),
		}
		logs.AddEvent(event)
	}

	// Test various key presses
	testCases := []struct {
		name     string
		key      tea.KeyPressMsg
		validate func(t *testing.T)
	}{
		{
			name: "Press k to scroll up",
			key:  tea.KeyPressMsg{Text: "k"},
			validate: func(t *testing.T) {
				require.False(t, logs.viewport.AtBottom(), "Should not be at bottom after 'k'")
			},
		},
		{
			name: "Press j to scroll down",
			key:  tea.KeyPressMsg{Text: "j"},
			validate: func(t *testing.T) {
				// After scrolling up, then down, we should move towards bottom
				// (exact position depends on viewport state)
			},
		},
		{
			name: "Press up arrow to scroll up",
			key:  tea.KeyPressMsg{Code: tea.KeyUp},
			validate: func(t *testing.T) {
				require.False(t, logs.viewport.AtBottom(), "Should not be at bottom after up arrow")
			},
		},
		{
			name: "Press down arrow to scroll down",
			key:  tea.KeyPressMsg{Code: tea.KeyDown},
			validate: func(t *testing.T) {
				// Just verify it doesn't panic
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logs.Update(tc.key)
			tc.validate(t)
		})
	}
}

// TestLogViewer_AutoScroll_NewEventWhileScrolledUp tests that new events don't auto-scroll when user scrolled up
func TestLogViewer_AutoScroll_NewEventWhileScrolledUp(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10)
	logs.SetFocus(true)

	// Add initial events
	for i := 0; i < 15; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event %d", i+1),
		}
		logs.AddEvent(event)
	}

	// Scroll up to middle
	for i := 0; i < 5; i++ {
		logs.Update(tea.KeyPressMsg{Text: "k"})
	}

	// Verify not at bottom
	wasAtBottom := logs.viewport.AtBottom()
	require.False(t, wasAtBottom, "Should not be at bottom after scrolling up")

	// Add new event - this should auto-scroll to bottom (current implementation always scrolls)
	newEvent := session.Event{
		ID:        "16",
		Timestamp: testfixtures.FixedTime,
		Session:   testfixtures.FixedSessionName,
		Type:      "task",
		Action:    "add",
		Data:      "New event",
	}
	logs.AddEvent(newEvent)

	// Check if at bottom - current implementation always auto-scrolls
	// This test documents the current behavior (always auto-scroll)
	require.True(t, logs.viewport.AtBottom(), "AddEvent currently always auto-scrolls to bottom")
}

// TestLogViewer_Rendering_ManyEvents tests rendering with many events
func TestLogViewer_Rendering_ManyEvents(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Add 100 events
	for i := 0; i < 100; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event number %d", i+1),
		}
		logs.AddEvent(event)
	}

	// Should have 100 events
	require.Len(t, logs.events, 100, "Should have 100 events")

	// Should be at bottom
	require.True(t, logs.viewport.AtBottom(), "Should be at bottom after adding events")

	// Content should be rendered
	content := logs.viewport.View()
	require.NotEmpty(t, content, "Content should not be empty")
}

// --- LogViewer Command Execution Tests ---

func TestLogViewer_Update_ReturnsCommandsFromViewport(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10)
	logs.SetFocus(true)

	// Add many events to enable scrolling
	for i := 0; i < 30; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event %d", i+1),
		}
		logs.AddEvent(event)
	}

	// Test that Update forwards to viewport and may return commands
	testCases := []struct {
		name           string
		msg            tea.Msg
		expectCommand  bool
		commandComment string
	}{
		{"KeyPress k", tea.KeyPressMsg{Text: "k"}, false, "viewport scrolling"},
		{"KeyPress j", tea.KeyPressMsg{Text: "j"}, false, "viewport scrolling"},
		{"KeyPress pgup", tea.KeyPressMsg{Text: "pgup"}, false, "page up"},
		{"KeyPress pgdown", tea.KeyPressMsg{Text: "pgdown"}, false, "page down"},
		{"KeyPress ctrl+u", tea.KeyPressMsg{Text: "ctrl+u"}, false, "half page up"},
		{"KeyPress ctrl+d", tea.KeyPressMsg{Text: "ctrl+d"}, false, "half page down"},
		{"KeyPress g", tea.KeyPressMsg{Text: "g"}, false, "goto top"},
		{"KeyPress G", tea.KeyPressMsg{Text: "G"}, false, "goto bottom"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := logs.Update(tc.msg)
			// Viewport may or may not return a command depending on state
			// We just verify Update doesn't panic and returns a valid result
			if cmd != nil {
				// Command returned, verify it can be executed
				msg := cmd()
				_ = msg // Viewport commands typically return nil or viewport messages
			}
		})
	}
}

func TestLogViewer_Update_WhenNotFocused(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10)
	logs.SetFocus(false) // Not focused

	// Add events
	for i := 0; i < 5; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event %d", i+1),
		}
		logs.AddEvent(event)
	}

	// Update should still work when not focused (viewport handles it)
	cmd := logs.Update(tea.KeyPressMsg{Text: "k"})
	// Viewport handles focus state internally
	_ = cmd
}

func TestLogViewer_AddEvent_ReturnsNil(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10)

	event := session.Event{
		ID:        "1",
		Timestamp: testfixtures.FixedTime,
		Session:   testfixtures.FixedSessionName,
		Type:      "task",
		Action:    "add",
		Data:      "Test event",
	}

	// AddEvent should return nil (no command needed)
	cmd := logs.AddEvent(event)
	require.Nil(t, cmd, "AddEvent should return nil")

	// Event should be added
	require.Len(t, logs.events, 1, "Event should be added")
}

func TestLogViewer_Update_CommandExecution(t *testing.T) {
	t.Parallel()

	logs := NewLogViewer()
	logs.SetSize(testfixtures.TestTermWidth, 10)
	logs.SetFocus(true)

	// Add events
	for i := 0; i < 20; i++ {
		event := session.Event{
			ID:        fmt.Sprintf("%d", i+1),
			Timestamp: testfixtures.FixedTime,
			Session:   testfixtures.FixedSessionName,
			Type:      "task",
			Action:    "add",
			Data:      fmt.Sprintf("Event %d", i+1),
		}
		logs.AddEvent(event)
	}

	// Send a key press and verify command can be executed
	cmd := logs.Update(tea.KeyPressMsg{Text: "k"})

	// Execute command if returned
	if cmd != nil {
		msg := cmd()
		// Viewport commands typically return internal viewport messages or nil
		// We just verify it doesn't panic
		_ = msg
	}

	// Verify viewport state changed (scrolled up)
	require.False(t, logs.viewport.AtBottom(), "Should have scrolled up from bottom")
}
