package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

func TestNewApp(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)

	if app == nil {
		t.Fatal("expected non-nil app")
	}
	if app.sessionName != "test-session" {
		t.Errorf("session name: got %s, want test-session", app.sessionName)
	}
	if app.activeView != ViewDashboard {
		t.Errorf("active view: got %v, want ViewDashboard", app.activeView)
	}
	if app.dashboard == nil {
		t.Error("expected non-nil dashboard")
	}
	if app.tasks == nil {
		t.Error("expected non-nil tasks")
	}
	if app.logs == nil {
		t.Error("expected non-nil logs")
	}
	if app.notes == nil {
		t.Error("expected non-nil notes")
	}
	if app.inbox == nil {
		t.Error("expected non-nil inbox")
	}
	if app.agent == nil {
		t.Error("expected non-nil agent")
	}
}

func TestApp_HandleKeyPress_ViewSwitching(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		expectedView ViewType
	}{
		{
			name:         "switch to dashboard",
			key:          "1",
			expectedView: ViewDashboard,
		},
		{
			name:         "switch to tasks",
			key:          "2",
			expectedView: ViewTasks,
		},
		{
			name:         "switch to logs",
			key:          "3",
			expectedView: ViewLogs,
		},
		{
			name:         "switch to notes",
			key:          "4",
			expectedView: ViewNotes,
		},
		{
			name:         "switch to inbox",
			key:          "5",
			expectedView: ViewInbox,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			app := NewApp(ctx, nil, "test-session", nil)
			app.activeView = ViewDashboard // Start at dashboard

			msg := tea.KeyPressMsg{Code: rune(tt.key[0]), Text: tt.key}
			updatedModel, _ := app.handleKeyPress(msg)
			app = updatedModel.(*App)

			if app.activeView != tt.expectedView {
				t.Errorf("active view: got %v, want %v", app.activeView, tt.expectedView)
			}
		})
	}
}

func TestApp_HandleKeyPress_Quit(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)

	// Test 'q' key
	msg := tea.KeyPressMsg{Code: 'q', Text: "q"}
	updatedModel, cmd := app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if !app.quitting {
		t.Error("expected quitting to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestApp_RenderActiveView(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)

	views := []ViewType{
		ViewDashboard,
		ViewTasks,
		ViewLogs,
		ViewNotes,
		ViewInbox,
	}

	for _, view := range views {
		t.Run(string(rune(view)), func(t *testing.T) {
			app.activeView = view
			output := app.renderActiveView()
			if output == "" {
				t.Errorf("expected non-empty output for view %v", view)
			}
		})
	}
}

func TestApp_Update_WindowSize(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 50,
	}

	updatedModel, _ := app.Update(msg)
	updatedApp := updatedModel.(*App)

	if updatedApp.width != 100 {
		t.Errorf("width: got %d, want 100", updatedApp.width)
	}
	if updatedApp.height != 50 {
		t.Errorf("height: got %d, want 50", updatedApp.height)
	}
}

func TestApp_Update_AgentOutput(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)

	msg := AgentOutputMsg{
		Content: "Test output",
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestApp_Update_IterationStart(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)

	msg := IterationStartMsg{
		Number: 5,
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestApp_Update_StateUpdate(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)

	state := &session.State{
		Session: "test-session",
		Tasks: map[string]*session.Task{
			"t1": {ID: "t1", Content: "Task 1", Status: "remaining"},
		},
	}

	msg := StateUpdateMsg{
		State: state,
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestApp_View(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)
	app.width = 100
	app.height = 50

	view := app.View()

	// Verify view properties are set correctly
	if !view.AltScreen {
		t.Error("expected AltScreen to be enabled")
	}

	if view.MouseMode != tea.MouseModeCellMotion {
		t.Errorf("mouse mode: got %v, want MouseModeCellMotion", view.MouseMode)
	}

	if !view.ReportFocus {
		t.Error("expected ReportFocus to be enabled")
	}
}

func TestApp_View_Quitting(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)
	app.quitting = true

	view := app.View()

	// Just verify we get a view back
	_ = view
}

func TestApp_RenderViewTabs(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)
	app.width = 100

	// Test with different active views
	views := []ViewType{
		ViewDashboard,
		ViewTasks,
		ViewLogs,
		ViewNotes,
		ViewInbox,
	}

	for _, view := range views {
		t.Run(string(rune(view)), func(t *testing.T) {
			app.activeView = view
			output := app.renderViewTabs()
			if output == "" {
				t.Error("expected non-empty view tabs")
			}
		})
	}
}

func TestApp_RenderHeader(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)
	app.width = 100

	output := app.renderHeader()
	if output == "" {
		t.Error("expected non-empty header")
	}
}

func TestApp_RenderFooter(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", nil)
	app.width = 100

	output := app.renderFooter()
	if output == "" {
		t.Error("expected non-empty footer")
	}
}

func TestViewType_Constants(t *testing.T) {
	// Verify view type constants are distinct
	views := []ViewType{
		ViewDashboard,
		ViewTasks,
		ViewLogs,
		ViewNotes,
		ViewInbox,
	}

	seen := make(map[ViewType]bool)
	for _, view := range views {
		if seen[view] {
			t.Errorf("duplicate view type: %v", view)
		}
		seen[view] = true
	}

	if len(seen) != 5 {
		t.Errorf("expected 5 distinct view types, got %d", len(seen))
	}
}
