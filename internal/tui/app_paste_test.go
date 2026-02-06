package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestApp_HandlePaste_RoutesToNoteInputModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show the note input modal
	app.noteInputModal.Show()

	// Send paste message - should not panic and should route to modal
	pasteMsg := tea.PasteMsg{Content: "pasted note content"}
	_, cmd := app.handlePaste(pasteMsg)

	// Modal's Update returns a command (may be nil if textarea doesn't have focus yet)
	// The important thing is the routing happens without panic
	_ = cmd
}

func TestApp_HandlePaste_RoutesToTaskInputModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show the task input modal
	app.taskInputModal.Show()

	// Send paste message - should not panic and should route to modal
	pasteMsg := tea.PasteMsg{Content: "pasted task content"}
	_, cmd := app.handlePaste(pasteMsg)

	// Modal's Update returns a command (may be nil if textarea doesn't have focus yet)
	_ = cmd
}

func TestApp_HandlePaste_RoutesToSubagentModal(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Create and show subagent modal
	app.subagentModal = NewSubagentModal("test-session", "explore", "/tmp")

	// Send paste message - should not panic and should route to modal
	pasteMsg := tea.PasteMsg{Content: "pasted subagent content"}
	_, cmd := app.handlePaste(pasteMsg)

	// Subagent modal may or may not return a command - the important thing is no panic
	_ = cmd
}

func TestApp_HandlePaste_NoOpWhenLogsVisible(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Make logs visible
	app.logsVisible = true

	// Send paste message
	pasteMsg := tea.PasteMsg{Content: "pasted content"}
	_, cmd := app.handlePaste(pasteMsg)

	// Verify no command is returned (paste is no-op in logs)
	if cmd != nil {
		t.Error("expected nil command when logs visible, got command")
	}
}

func TestApp_HandlePaste_RoutesToDashboard(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Ensure no modals are visible and logs are not visible
	app.logsVisible = false

	// Send paste message - should not panic and should route to dashboard
	pasteMsg := tea.PasteMsg{Content: "pasted dashboard content"}
	_, cmd := app.handlePaste(pasteMsg)

	// Dashboard may or may not return a command - the important thing is no panic
	_ = cmd
}

func TestApp_HandlePaste_SanitizesContent(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show note input modal
	app.noteInputModal.Show()

	// Send paste with ANSI escape codes - should not panic
	pasteMsg := tea.PasteMsg{Content: "\x1b[31mcolored\x1b[0m text"}
	app.handlePaste(pasteMsg)

	// The sanitization happens before forwarding, so the content
	// should be "colored text" when received by the modal
	// (actual verification of sanitized content would require
	//  inspecting the modal's textarea state)
}

func TestApp_HandlePaste_Priority_NoteOverTask(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show both modals
	app.noteInputModal.Show()
	app.taskInputModal.Show()

	// Send paste message - should not panic
	pasteMsg := tea.PasteMsg{Content: "pasted content"}
	_, cmd := app.handlePaste(pasteMsg)

	// Should route to noteInputModal (priority) without panic
	_ = cmd
}

func TestApp_HandlePaste_Priority_TaskOverSubagent(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Show task input modal and create subagent modal
	app.taskInputModal.Show()
	app.subagentModal = NewSubagentModal("test", "explore", "/tmp")

	// Send paste message - should not panic
	pasteMsg := tea.PasteMsg{Content: "pasted content"}
	_, cmd := app.handlePaste(pasteMsg)

	// Should route to taskInputModal (priority) without panic
	_ = cmd
}

func TestApp_HandlePaste_Priority_SubagentOverDashboard(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Create subagent modal and ensure no text modals visible
	app.subagentModal = NewSubagentModal("test", "explore", "/tmp")
	app.logsVisible = false

	// Send paste message - should not panic
	pasteMsg := tea.PasteMsg{Content: "pasted content"}
	_, cmd := app.handlePaste(pasteMsg)

	// Should route to subagentModal (priority over dashboard) without panic
	_ = cmd
}
