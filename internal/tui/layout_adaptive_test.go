package tui

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// TestHeader_AdaptToLayoutMode verifies Header adapts to compact mode.
func TestHeader_AdaptToLayoutMode(t *testing.T) {
	tests := []struct {
		name       string
		mode       LayoutMode
		width      int
		expectFull bool // Expect full "connected" text vs just "●"
	}{
		{
			name:       "desktop mode shows full text",
			mode:       LayoutDesktop,
			width:      120,
			expectFull: true,
		},
		{
			name:       "compact mode shows condensed text",
			mode:       LayoutCompact,
			width:      80,
			expectFull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create header
			h := NewHeader("test-session")
			h.SetLayoutMode(tt.mode)
			h.SetSize(tt.width, 1)

			// Create a simple state with iteration
			state := &session.State{
				Iterations: []*session.Iteration{
					{Number: 5},
				},
			}
			h.SetState(state)

			// Create screen buffer and draw
			canvas := uv.NewScreenBuffer(tt.width, 1)
			area := uv.Rect(0, 0, tt.width, 1)
			h.Draw(canvas, area)

			// Get rendered content
			content := canvas.Render()

			// Verify based on mode
			if tt.expectFull {
				// Desktop mode should show "connected"
				if !strings.Contains(content, "connected") {
					t.Errorf("Desktop mode should contain 'connected' text, got: %q", content)
				}
				// Desktop mode should show "Iteration #"
				if !strings.Contains(content, "Iteration") {
					t.Errorf("Desktop mode should contain 'Iteration' text, got: %q", content)
				}
			} else {
				// Compact mode should NOT show "connected" (just dot)
				if strings.Contains(content, "connected") {
					t.Errorf("Compact mode should not contain 'connected' text, got: %q", content)
				}
				// Compact mode should show just "#" prefix
				if !strings.Contains(content, "#5") {
					t.Errorf("Compact mode should contain '#5' for iteration, got: %q", content)
				}
			}

			// Both modes should show the dot indicator
			if !strings.Contains(content, "●") {
				t.Errorf("Both modes should contain '●' indicator, got: %q", content)
			}
		})
	}
}

// TestStatusBar_AdaptToLayoutMode verifies StatusBar adapts to compact mode.
func TestStatusBar_AdaptToLayoutMode(t *testing.T) {
	tests := []struct {
		name       string
		mode       LayoutMode
		width      int
		expectFull bool // Expect full "connected" text vs just "●"
	}{
		{
			name:       "desktop mode shows full text",
			mode:       LayoutDesktop,
			width:      120,
			expectFull: true,
		},
		{
			name:       "compact mode shows condensed text",
			mode:       LayoutCompact,
			width:      80,
			expectFull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create status bar
			s := NewStatusBar()
			s.SetLayoutMode(tt.mode)
			s.SetSize(tt.width, 1)
			s.SetConnectionStatus(true)

			// Create a simple state with an in_progress task
			state := &session.State{
				Tasks: map[string]*session.Task{
					"task1": {
						ID:      "task1",
						Status:  "in_progress",
						Content: "This is a very long task name that should be truncated in compact mode",
					},
				},
			}
			s.SetState(state)

			// Create screen buffer and draw
			canvas := uv.NewScreenBuffer(tt.width, 1)
			area := uv.Rect(0, 0, tt.width, 1)
			s.Draw(canvas, area)

			// Get rendered content
			content := canvas.Render()

			// Verify based on mode
			if tt.expectFull {
				// Desktop mode should show "connected"
				if !strings.Contains(content, "connected") {
					t.Errorf("Desktop mode should contain 'connected' text, got: %q", content)
				}
			} else {
				// Compact mode should NOT show "connected" (just dot)
				if strings.Contains(content, "connected") {
					t.Errorf("Compact mode should not contain 'connected' text, got: %q", content)
				}
			}

			// Both modes should show task content
			if !strings.Contains(content, "task:") {
				t.Errorf("Both modes should contain 'task:' prefix, got: %q", content)
			}

			// Both modes should show the working indicator (spinner animation)
			// The spinner uses braille characters like ⠋, ⠙, ⠹, etc.
			hasSpinner := strings.ContainsAny(content, "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
			if !hasSpinner {
				t.Errorf("Both modes should contain working indicator (spinner), got: %q", content)
			}
		})
	}
}

// TestStatusBar_TruncateTaskInCompactMode verifies more aggressive truncation.
func TestStatusBar_TruncateTaskInCompactMode(t *testing.T) {
	// Create status bar in compact mode
	s := NewStatusBar()
	s.SetLayoutMode(LayoutCompact)
	s.SetSize(80, 1)
	s.SetConnectionStatus(true)

	// Create state with long task name
	longTaskName := "This is an extremely long task name that definitely exceeds the 30 character limit for compact mode"
	state := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Status:  "in_progress",
				Content: longTaskName,
			},
		},
	}
	s.SetState(state)

	// Get current task text
	taskText := s.getCurrentTask()

	// Verify truncation occurred (should be ~33 chars: "task: " + 30 chars max)
	// Allow some buffer for "..." suffix
	if len(taskText) > 40 {
		t.Errorf("Compact mode task text should be truncated to ~33 chars, got %d: %q", len(taskText), taskText)
	}

	// Verify ellipsis was added
	if !strings.HasSuffix(taskText, "...") {
		t.Errorf("Truncated task should end with '...', got: %q", taskText)
	}
}

// TestFooter_AdaptToLayoutMode verifies Footer shows sidebar toggle in compact mode.
func TestFooter_AdaptToLayoutMode(t *testing.T) {
	tests := []struct {
		name             string
		mode             LayoutMode
		expectSidebarKey bool
	}{
		{
			name:             "desktop mode no sidebar toggle",
			mode:             LayoutDesktop,
			expectSidebarKey: false,
		},
		{
			name:             "compact mode shows sidebar toggle",
			mode:             LayoutCompact,
			expectSidebarKey: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create footer
			f := NewFooter()
			f.SetLayoutMode(tt.mode)
			f.SetSize(120, 1)

			// Create screen buffer and draw
			canvas := uv.NewScreenBuffer(120, 1)
			area := uv.Rect(0, 0, 120, 1)
			f.Draw(canvas, area)

			// Get rendered content
			content := canvas.Render()

			// Verify sidebar hint
			hasSidebarKey := strings.Contains(content, "[s]") && strings.Contains(content, "Sidebar")

			if tt.expectSidebarKey && !hasSidebarKey {
				t.Errorf("Compact mode should show '[s]Sidebar' hint, got: %q", content)
			}

			if !tt.expectSidebarKey && hasSidebarKey {
				t.Errorf("Desktop mode should not show '[s]Sidebar' hint, got: %q", content)
			}

			// Both modes should show view shortcuts
			if !strings.Contains(content, "[1]") || !strings.Contains(content, "[2]") {
				t.Errorf("Both modes should show view shortcuts, got: %q", content)
			}
		})
	}
}
