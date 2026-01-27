package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

func TestNewApp(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)

	if app == nil {
		t.Fatal("expected non-nil app")
		return // Explicit return to help static analysis
	}
	if app.sessionName != "test-session" {
		t.Errorf("session name: got %s, want test-session", app.sessionName)
	}
	if app.dashboard == nil {
		t.Error("expected non-nil dashboard")
	}
	if app.logs == nil {
		t.Error("expected non-nil logs")
	}
	if app.agent == nil {
		t.Error("expected non-nil agent")
	}
}

func TestApp_HandleKeyPress_LogsToggle(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)

	// Initially logs not visible
	if app.logsVisible {
		t.Error("logs should not be visible initially")
	}

	// ctrl+l toggles logs
	msg := tea.KeyPressMsg{Text: "ctrl+l"}
	app.handleKeyPress(msg)
	if !app.logsVisible {
		t.Error("logs should be visible after ctrl+l")
	}

	// ctrl+l again hides
	app.handleKeyPress(msg)
	if app.logsVisible {
		t.Error("logs should be hidden after second ctrl+l")
	}
}

func TestApp_HandleKeyPress_Quit(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)

	// Test ctrl+c
	msg := tea.KeyPressMsg{Text: "ctrl+c"}
	_, cmd := app.handleKeyPress(msg)

	if !app.quitting {
		t.Error("expected quitting to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

// TestApp_RenderActiveView removed - renderActiveView() method was removed in Phase 12.4
// View rendering now uses Draw pattern with Ultraviolet Screen buffer

func TestApp_Update_WindowSize(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)

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
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)

	msg := AgentOutputMsg{
		Content: "Test output",
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestApp_Update_IterationStart(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)

	msg := IterationStartMsg{
		Number: 5,
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestApp_Update_StateUpdate(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)

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
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)
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
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)
	app.quitting = true

	view := app.View()

	// Just verify we get a view back
	_ = view
}

// TestApp_RenderViewTabs removed - renderViewTabs() method was removed in Phase 12.4
// View navigation now handled by Footer component

// TestApp_RenderHeader removed - renderHeader() method was removed in Phase 12.4
// Header now handled by Header component with Draw pattern

// TestApp_RenderFooter removed - renderFooter() method was removed in Phase 12.4
// Footer now handled by Footer component with Draw pattern

func TestViewType_Constants(t *testing.T) {
	// Verify view type constants are distinct
	views := []ViewType{
		ViewDashboard,
		ViewLogs,
		ViewNotes,
	}

	seen := make(map[ViewType]bool)
	for _, view := range views {
		if seen[view] {
			t.Errorf("duplicate view type: %v", view)
		}
		seen[view] = true
	}

	if len(seen) != 3 {
		t.Errorf("expected 3 distinct view types, got %d", len(seen))
	}
}

func TestApp_HandleKeyPress_SidebarToggle(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", nil, nil)

	// Initially sidebar should be hidden
	if app.sidebarVisible {
		t.Error("expected sidebar to be hidden initially")
	}

	// Press ctrl+s to toggle sidebar visible
	msg := tea.KeyPressMsg{Text: "ctrl+s"}
	updatedModel, _ := app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if !app.sidebarVisible {
		t.Error("expected sidebar to be visible after ctrl+s")
	}

	// Press ctrl+s again to toggle sidebar hidden
	msg = tea.KeyPressMsg{Text: "ctrl+s"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if app.sidebarVisible {
		t.Error("expected sidebar to be hidden after second ctrl+s")
	}
}
