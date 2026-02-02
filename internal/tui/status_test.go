package tui

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

func TestStatusBar_SpinnerAnimation(t *testing.T) {
	tests := []struct {
		name          string
		hasTasks      bool
		taskStatus    string
		expectSpinner bool
	}{
		{
			name:          "shows spinner when task in_progress",
			hasTasks:      true,
			taskStatus:    "in_progress",
			expectSpinner: true,
		},
		{
			name:          "no spinner when no tasks",
			hasTasks:      false,
			expectSpinner: false,
		},
		{
			name:          "no spinner when task completed",
			hasTasks:      true,
			taskStatus:    "completed",
			expectSpinner: false,
		},
		{
			name:          "no spinner when task remaining",
			hasTasks:      true,
			taskStatus:    "remaining",
			expectSpinner: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBar("test-session")
			sb.SetLayoutMode(LayoutDesktop)
			sb.SetConnectionStatus(true)

			// Create state with tasks if needed
			if tt.hasTasks {
				state := &session.State{
					Tasks: map[string]*session.Task{
						"test-task": {
							ID:      "test-task",
							Content: "Test task",
							Status:  tt.taskStatus,
						},
					},
				}
				sb.SetState(state)
			}

			// Tick() starts the spinner after SetState
			cmd := sb.Tick()

			// Verify Tick returns tick command when working
			if tt.expectSpinner {
				if cmd == nil {
					t.Error("Expected Tick to return command when working, got nil")
				}
			} else {
				if cmd != nil {
					t.Errorf("Expected Tick to return nil when not working, got %T", cmd)
				}
			}

			// Render the status bar
			canvas := uv.NewScreenBuffer(100, 1)
			area := uv.Rect(0, 0, 100, 1)
			sb.Draw(canvas, area)
			content := canvas.Render()

			// Verify content includes session info and keybinding hints
			if !strings.Contains(content, "iteratr") {
				t.Errorf("Expected session title, got: %s", content)
			}
			if !strings.Contains(content, "quit") {
				t.Errorf("Expected keybinding hints, got: %s", content)
			}
		})
	}
}

func TestStatusBar_SpinnerTicking(t *testing.T) {
	sb := NewStatusBar("test-session")
	sb.SetLayoutMode(LayoutDesktop)

	// Create state with in_progress task
	state := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Working task",
				Status:  "in_progress",
			},
		},
	}
	sb.SetState(state)

	// Tick() should return the initial tick command
	cmd1 := sb.Tick()
	if cmd1 == nil {
		t.Fatal("Expected Tick to return command after SetState with in_progress task")
	}

	// Execute the tick command to get a spinner message
	msg := cmd1()

	// Update with spinner message should return another tick (chain continues)
	cmd2 := sb.Update(msg)
	if cmd2 == nil {
		t.Error("Expected Update with spinner message to return next tick")
	}

	// Verify spinner continues ticking
	msg2 := cmd2()
	cmd3 := sb.Update(msg2)
	if cmd3 == nil {
		t.Error("Expected spinner to continue ticking")
	}
}

func TestStatusBar_SpinnerStopsWhenDone(t *testing.T) {
	sb := NewStatusBar("test-session")
	sb.SetLayoutMode(LayoutDesktop)

	// Start with in_progress task
	state := &session.State{
		Tasks: map[string]*session.Task{
			"task1": {
				ID:      "task1",
				Content: "Working task",
				Status:  "in_progress",
			},
		},
	}
	sb.SetState(state)

	// Tick should return command
	cmd := sb.Tick()
	if cmd == nil {
		t.Fatal("Expected Tick to return command when working")
	}

	// Now complete the task
	state.Tasks["task1"].Status = "completed"
	sb.SetState(state)

	// Tick should now return nil (no longer working)
	cmd = sb.Tick()
	if cmd != nil {
		t.Error("Expected Tick to return nil when no longer working")
	}

	// Update should also return nil
	cmd = sb.Update(nil)
	if cmd != nil {
		t.Error("Expected Update to return nil when no longer working")
	}
}

func TestStatusBar_SidebarHint(t *testing.T) {
	tests := []struct {
		name          string
		sidebarHidden bool
		expectHint    bool
	}{
		{
			name:          "shows sidebar hint when hidden",
			sidebarHidden: true,
			expectHint:    true,
		},
		{
			name:          "no sidebar hint when visible",
			sidebarHidden: false,
			expectHint:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBar("test-session")
			sb.SetLayoutMode(LayoutDesktop)
			sb.SetSidebarHidden(tt.sidebarHidden)

			// Render the status bar
			canvas := uv.NewScreenBuffer(150, 1)
			area := uv.Rect(0, 0, 150, 1)
			sb.Draw(canvas, area)
			content := canvas.Render()

			// Check for sidebar hint
			hasSidebarHint := strings.Contains(content, "sidebar")

			if tt.expectHint && !hasSidebarHint {
				t.Errorf("Expected sidebar hint when hidden, got: %s", content)
			}
			if !tt.expectHint && hasSidebarHint {
				t.Errorf("Expected no sidebar hint when visible, got: %s", content)
			}

			// Standard hints should always be present
			if !strings.Contains(content, "quit") {
				t.Errorf("Expected quit hint, got: %s", content)
			}
		})
	}
}
