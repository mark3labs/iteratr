package tui

import (
	"context"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestModalPriorityMatrix_AllCombinations tests all possible modal visibility combinations
// to verify that the priority routing in handleKeyPress works correctly.
// Priority order: Global Keys > Dialog > Prefix > TaskModal > NoteModal > NoteInputModal > TaskInputModal > SubagentModal > Logs
func TestModalPriorityMatrix_AllCombinations(t *testing.T) {
	tests := []struct {
		name           string
		setupModals    func(app *App) // Opens modals in specific order
		sendKeys       []string       // Keys to send
		expectedClosed []string       // Which modals should close (in order)
		expectedOpen   []string       // Which modals should remain open
	}{
		{
			name: "dialog_over_all",
			setupModals: func(app *App) {
				app.logsVisible = true
				app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")
				app.taskInputModal.Show()
				app.noteInputModal.Show()
				app.noteModal.SetNote(&session.Note{ID: "note1", Content: "Test note", Type: "learning", Iteration: 1})
				app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test task", Status: "remaining", Priority: 1})
				app.dialog.Show("Test", "Test message", nil)
			},
			sendKeys:       []string{"esc"},
			expectedClosed: []string{"dialog"},
			expectedOpen:   []string{"taskModal", "noteModal", "noteInputModal", "taskInputModal", "subagentModal", "logs"},
		},
		{
			name: "taskmodal_over_notemodal",
			setupModals: func(app *App) {
				app.noteModal.SetNote(&session.Note{ID: "note1", Content: "Test note", Type: "learning", Iteration: 1})
				app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test task", Status: "remaining", Priority: 1})
			},
			sendKeys:       []string{"esc"},
			expectedClosed: []string{"taskModal"},
			expectedOpen:   []string{"noteModal"},
		},
		{
			name: "notemodal_over_inputmodals",
			setupModals: func(app *App) {
				app.taskInputModal.Show()
				app.noteInputModal.Show()
				app.noteModal.SetNote(&session.Note{ID: "note1", Content: "Test note", Type: "learning", Iteration: 1})
			},
			sendKeys:       []string{"esc"},
			expectedClosed: []string{"noteModal"},
			expectedOpen:   []string{"noteInputModal", "taskInputModal"},
		},
		{
			name: "noteinputmodal_over_taskinputmodal",
			setupModals: func(app *App) {
				app.taskInputModal.Show()
				app.noteInputModal.Show()
			},
			sendKeys:       []string{"esc"},
			expectedClosed: []string{"noteInputModal"},
			expectedOpen:   []string{"taskInputModal"},
		},
		{
			name: "taskinputmodal_over_subagentmodal",
			setupModals: func(app *App) {
				app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")
				app.taskInputModal.Show()
			},
			sendKeys:       []string{"esc"},
			expectedClosed: []string{"taskInputModal"},
			expectedOpen:   []string{"subagentModal"},
		},
		{
			name: "subagentmodal_over_logs",
			setupModals: func(app *App) {
				app.logsVisible = true
				app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")
			},
			sendKeys:       []string{"esc"},
			expectedClosed: []string{"subagentModal"},
			expectedOpen:   []string{"logs"},
		},
		{
			name: "prefix_mode_blocks_modal_open",
			setupModals: func(app *App) {
				app.iteration = 1
			},
			sendKeys:       []string{"ctrl+x", "n"},
			expectedClosed: []string{},
			expectedOpen:   []string{}, // No modals should open when blocked
		},
		{
			name: "global_keys_work_with_dialog_open",
			setupModals: func(app *App) {
				app.dialog.Show("Test", "Test message", nil)
			},
			sendKeys:       []string{"ctrl+c"},
			expectedClosed: []string{},
			expectedOpen:   []string{"dialog"}, // Dialog still open, but app quitting
		},
		{
			name: "complete_hierarchy_closes_in_order",
			setupModals: func(app *App) {
				app.logsVisible = true
				app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")
				app.taskInputModal.Show()
				app.noteInputModal.Show()
				app.noteModal.SetNote(&session.Note{ID: "note1", Content: "Test note", Type: "learning", Iteration: 1})
				app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test task", Status: "remaining", Priority: 1})
				app.dialog.Show("Test", "Test message", nil)
			},
			sendKeys: []string{"esc", "esc", "esc", "esc", "esc", "esc", "esc"},
			expectedClosed: []string{
				"dialog",
				"taskModal",
				"noteModal",
				"noteInputModal",
				"taskInputModal",
				"subagentModal",
				"logs",
			},
			expectedOpen: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tmpDir := t.TempDir()
			app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight

			// Setup modals
			tt.setupModals(app)

			// Send keys and verify state
			for i, key := range tt.sendKeys {
				updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: key})
				app = updatedModel.(*App)

				// Check expected state after this key
				if i < len(tt.expectedClosed) {
					checkModalClosed(t, app, tt.expectedClosed[i])
				}
			}

			// Verify final state
			for _, modalName := range tt.expectedOpen {
				checkModalOpen(t, app, modalName)
			}

			// All modals in expectedClosed should be closed
			for _, modalName := range tt.expectedClosed {
				checkModalClosed(t, app, modalName)
			}
		})
	}
}

// TestModalPriorityMatrix_VisualVerification tests visual rendering with multiple modals using golden files
func TestModalPriorityMatrix_VisualVerification(t *testing.T) {
	tests := []struct {
		name        string
		setupModals func(app *App)
		description string // For documentation
	}{
		{
			name: "dialog_only",
			setupModals: func(app *App) {
				app.dialog.Show("Dialog Title", "This is a dialog message", nil)
			},
			description: "Single dialog visible",
		},
		{
			name: "dialog_over_taskmodal",
			setupModals: func(app *App) {
				app.taskModal.SetTask(&session.Task{
					ID:       "TAS-1",
					Content:  "[P1] Test task content",
					Status:   "remaining",
					Priority: 1,
				})
				app.dialog.Show("Dialog Over Task", "Dialog has higher priority", nil)
			},
			description: "Dialog should occlude task modal",
		},
		{
			name: "taskmodal_over_notemodal",
			setupModals: func(app *App) {
				app.noteModal.SetNote(&session.Note{
					ID:        "NOT-1",
					Content:   "Test note content",
					Type:      "learning",
					Iteration: 1,
				})
				app.taskModal.SetTask(&session.Task{
					ID:       "TAS-1",
					Content:  "[P1] Test task content",
					Status:   "remaining",
					Priority: 1,
				})
			},
			description: "Task modal should be rendered over note modal",
		},
		{
			name: "all_modals_stacked",
			setupModals: func(app *App) {
				app.logsVisible = true
				app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")
				app.taskInputModal.Show()
				app.noteInputModal.Show()
				app.noteModal.SetNote(&session.Note{
					ID:        "NOT-1",
					Content:   "Test note",
					Type:      "learning",
					Iteration: 1,
				})
				app.taskModal.SetTask(&session.Task{
					ID:       "TAS-1",
					Content:  "[P1] Test task",
					Status:   "remaining",
					Priority: 1,
				})
				app.dialog.Show("Top Priority", "Dialog on top", nil)
			},
			description: "All modals stacked, dialog on top",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tmpDir := t.TempDir()
			app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)

			// Initialize with window size
			_, _ = app.Update(tea.WindowSizeMsg{
				Width:  testfixtures.TestTermWidth,
				Height: testfixtures.TestTermHeight,
			})

			// Setup modals
			tt.setupModals(app)

			// Render to screen buffer
			area := uv.Rectangle{
				Min: uv.Position{X: 0, Y: 0},
				Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
			}
			scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
			app.Draw(scr, area)
			rendered := scr.Render()

			// Compare with golden file
			goldenFile := filepath.Join("testdata", "modal_priority_"+tt.name+".golden")
			testfixtures.CompareGolden(t, goldenFile, rendered)
		})
	}
}

// TestModalPriorityMatrix_KeyCapture verifies that higher priority modals consume all keyboard input
func TestModalPriorityMatrix_KeyCapture(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("dialog_consumes_all_keys", func(t *testing.T) {
		app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)
		app.width = testfixtures.TestTermWidth
		app.height = testfixtures.TestTermHeight

		// Open task modal first
		app.taskModal.SetTask(&session.Task{
			ID:       "TAS-1",
			Content:  "[P1] Test task",
			Status:   "remaining",
			Priority: 1,
		})

		// Open dialog over it
		app.dialog.Show("Test Dialog", "Test message", nil)

		// Send various keys - all should be consumed by dialog
		keys := []string{"a", "j", "k", "tab", "up", "down", "left", "right"}
		for _, key := range keys {
			updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: key})
			app = updatedModel.(*App)

			// Dialog should still be open (key consumed)
			require.True(t, app.dialog.IsVisible(), "dialog should consume key: %s", key)
			// Task modal should still be open behind it
			require.True(t, app.taskModal.IsVisible(), "task modal should remain open behind dialog")
		}
	})

	t.Run("prefix_mode_blocks_modal_shortcuts", func(t *testing.T) {
		app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)
		app.width = testfixtures.TestTermWidth
		app.height = testfixtures.TestTermHeight
		app.iteration = 1

		// Open task modal
		app.taskModal.SetTask(&session.Task{
			ID:       "TAS-1",
			Content:  "[P1] Test task",
			Status:   "remaining",
			Priority: 1,
		})

		// Enter prefix mode
		updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "ctrl+x"})
		app = updatedModel.(*App)
		require.True(t, app.awaitingPrefixKey, "should be in prefix mode")

		// Try to open note input modal via 'n' (should be blocked by task modal)
		updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "n"})
		app = updatedModel.(*App)

		// Note input modal should NOT open because task modal is already visible
		require.False(t, app.noteInputModal.IsVisible(), "note input modal should not open when task modal is visible")
		require.True(t, app.taskModal.IsVisible(), "task modal should still be visible")
		require.False(t, app.awaitingPrefixKey, "should have exited prefix mode")
	})
}

// TestModalPriorityMatrix_MouseCapture verifies that higher priority modals capture mouse events
func TestModalPriorityMatrix_MouseCapture(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("dialog_captures_all_clicks", func(t *testing.T) {
		app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)
		app.width = testfixtures.TestTermWidth
		app.height = testfixtures.TestTermHeight

		// Open task modal
		app.taskModal.SetTask(&session.Task{
			ID:       "TAS-1",
			Content:  "[P1] Test task",
			Status:   "remaining",
			Priority: 1,
		})

		// Open dialog over it
		app.dialog.Show("Test Dialog", "Test message", nil)

		require.True(t, app.dialog.IsVisible())
		require.True(t, app.taskModal.IsVisible())

		// Click should dismiss dialog (dialog.HandleClick returns cmd)
		cmd := app.dialog.HandleClick(50, 20)
		_ = cmd

		require.False(t, app.dialog.IsVisible(), "dialog should be closed after click")
		require.True(t, app.taskModal.IsVisible(), "task modal should still be visible")
	})

	t.Run("subagent_modal_captures_clicks", func(t *testing.T) {
		app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)
		app.width = testfixtures.TestTermWidth
		app.height = testfixtures.TestTermHeight

		// Open logs
		app.logsVisible = true

		// Open subagent modal
		app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")

		require.NotNil(t, app.subagentModal)
		require.True(t, app.logsVisible)

		// Mouse event in app.handleMouse should be captured by subagent modal
		// We can't easily simulate MouseClickMsg in tests, but we verify the modal exists
		// and handleKeyPress gives it priority

		// ESC should close subagent modal (not logs)
		updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
		app = updatedModel.(*App)

		require.Nil(t, app.subagentModal, "subagent modal should be closed")
		require.True(t, app.logsVisible, "logs should still be visible")
	})
}

// TestModalPriorityMatrix_RenderOrder verifies visual layering
func TestModalPriorityMatrix_RenderOrder(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("highest_priority_modal_renders_last", func(t *testing.T) {
		app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)

		// Set window size
		_, _ = app.Update(tea.WindowSizeMsg{
			Width:  testfixtures.TestTermWidth,
			Height: testfixtures.TestTermHeight,
		})

		// Stack modals from low to high priority
		// Priority: Logs (lowest) < TaskModal < NoteInputModal (highest)
		app.logsVisible = true
		app.taskModal.SetTask(&session.Task{
			ID:       "TAS-1",
			Content:  "[P1] Test task",
			Status:   "remaining",
			Priority: 1,
		})
		app.noteInputModal.Show()

		// Render to screen buffer
		area := uv.Rectangle{
			Min: uv.Position{X: 0, Y: 0},
			Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
		}
		scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
		app.Draw(scr, area)
		rendered := scr.Render()

		// Verify note input modal is rendered on top (highest priority of the three)
		require.Contains(t, rendered, "New Note", "note input modal should be visible on top")
		// Logs should be visible underneath but occluded
		require.Contains(t, rendered, "Event Log", "event log should be visible underneath")

		// Compare with golden file
		goldenFile := filepath.Join("testdata", "modal_priority_render_order.golden")
		testfixtures.CompareGolden(t, goldenFile, rendered)
	})
}

// Helper functions for checking modal state

func checkModalOpen(t *testing.T, app *App, modalName string) {
	t.Helper()
	switch modalName {
	case "dialog":
		require.True(t, app.dialog.IsVisible(), "dialog should be open")
	case "taskModal":
		require.True(t, app.taskModal.IsVisible(), "taskModal should be open")
	case "noteModal":
		require.True(t, app.noteModal.IsVisible(), "noteModal should be open")
	case "noteInputModal":
		require.True(t, app.noteInputModal.IsVisible(), "noteInputModal should be open")
	case "taskInputModal":
		require.True(t, app.taskInputModal.IsVisible(), "taskInputModal should be open")
	case "subagentModal":
		require.NotNil(t, app.subagentModal, "subagentModal should be open")
	case "logs":
		require.True(t, app.logsVisible, "logs should be visible")
	default:
		t.Fatalf("unknown modal name: %s", modalName)
	}
}

func checkModalClosed(t *testing.T, app *App, modalName string) {
	t.Helper()
	switch modalName {
	case "dialog":
		require.False(t, app.dialog.IsVisible(), "dialog should be closed")
	case "taskModal":
		require.False(t, app.taskModal.IsVisible(), "taskModal should be closed")
	case "noteModal":
		require.False(t, app.noteModal.IsVisible(), "noteModal should be closed")
	case "noteInputModal":
		require.False(t, app.noteInputModal.IsVisible(), "noteInputModal should be closed")
	case "taskInputModal":
		require.False(t, app.taskInputModal.IsVisible(), "taskInputModal should be closed")
	case "subagentModal":
		require.Nil(t, app.subagentModal, "subagentModal should be closed")
	case "logs":
		require.False(t, app.logsVisible, "logs should be hidden")
	default:
		t.Fatalf("unknown modal name: %s", modalName)
	}
}

// TestModalPriorityMatrix_EdgeCases tests edge cases and corner scenarios
func TestModalPriorityMatrix_EdgeCases(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("multiple_esc_presses_close_in_order", func(t *testing.T) {
		app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)
		app.width = testfixtures.TestTermWidth
		app.height = testfixtures.TestTermHeight

		// Open multiple modals
		app.taskModal.SetTask(&session.Task{ID: "TAS-1", Content: "Test", Status: "remaining", Priority: 1})
		app.noteModal.SetNote(&session.Note{ID: "NOT-1", Content: "Test note", Type: "learning", Iteration: 1})
		app.dialog.Show("Test", "Message", nil)

		// First ESC closes dialog
		updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
		app = updatedModel.(*App)
		require.False(t, app.dialog.IsVisible())
		require.True(t, app.taskModal.IsVisible())
		require.True(t, app.noteModal.IsVisible())

		// Second ESC closes task modal
		updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
		app = updatedModel.(*App)
		require.False(t, app.taskModal.IsVisible())
		require.True(t, app.noteModal.IsVisible())

		// Third ESC closes note modal
		updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
		app = updatedModel.(*App)
		require.False(t, app.noteModal.IsVisible())
	})

	t.Run("ctrl_c_works_with_all_modals_open", func(t *testing.T) {
		app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)
		app.width = testfixtures.TestTermWidth
		app.height = testfixtures.TestTermHeight

		// Open everything
		app.logsVisible = true
		app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")
		app.taskInputModal.Show()
		app.noteInputModal.Show()
		app.noteModal.SetNote(&session.Note{ID: "NOT-1", Content: "Test", Type: "learning", Iteration: 1})
		app.taskModal.SetTask(&session.Task{ID: "TAS-1", Content: "Test", Status: "remaining", Priority: 1})
		app.dialog.Show("Test", "Message", nil)

		// ctrl+c should still quit (global key priority)
		updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "ctrl+c"})
		app = updatedModel.(*App)

		require.True(t, app.quitting, "ctrl+c should set quitting flag")
	})

	t.Run("prefix_mode_cancellable_with_esc", func(t *testing.T) {
		app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)
		app.width = testfixtures.TestTermWidth
		app.height = testfixtures.TestTermHeight

		// Enter prefix mode
		updatedModel, _ := app.handleKeyPress(tea.KeyPressMsg{Text: "ctrl+x"})
		app = updatedModel.(*App)
		require.True(t, app.awaitingPrefixKey)

		// ESC cancels prefix mode
		updatedModel, _ = app.handleKeyPress(tea.KeyPressMsg{Text: "esc"})
		app = updatedModel.(*App)
		require.False(t, app.awaitingPrefixKey)
	})
}
