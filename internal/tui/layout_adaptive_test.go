package tui

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// TestStatusBar_AdaptToLayoutMode verifies StatusBar renders in both modes.
func TestStatusBar_AdaptToLayoutMode(t *testing.T) {
	tests := []struct {
		name  string
		mode  LayoutMode
		width int
	}{
		{
			name:  "desktop mode",
			mode:  LayoutDesktop,
			width: 120,
		},
		{
			name:  "compact mode",
			mode:  LayoutCompact,
			width: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create status bar
			s := NewStatusBar("test-session")
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

			// Both modes should show session info and keybinding hints
			if !strings.Contains(content, "test-session") {
				t.Errorf("Should contain session name, got: %q", content)
			}
			if !strings.Contains(content, "quit") {
				t.Errorf("Should contain keybinding hints, got: %q", content)
			}

			// Both modes should show the working indicator (spinner animation)
			hasSpinner := strings.ContainsAny(content, "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
			if !hasSpinner {
				t.Errorf("Should contain spinner when working, got: %q", content)
			}
		})
	}
}
