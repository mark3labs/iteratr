package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// --- Initialization Tests ---

func TestApp_Initialization(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)

	require.NotNil(t, app, "app should be initialized")
	require.Equal(t, testfixtures.FixedSessionName, app.sessionName, "session name should match")
	require.NotNil(t, app.dashboard, "dashboard should be initialized")
	require.NotNil(t, app.logs, "logs should be initialized")
	require.NotNil(t, app.agent, "agent should be initialized")
	require.NotNil(t, app.sidebar, "sidebar should be initialized")
	require.NotNil(t, app.status, "status should be initialized")
	require.NotNil(t, app.dialog, "dialog should be initialized")
	require.NotNil(t, app.taskModal, "task modal should be initialized")
	require.NotNil(t, app.noteModal, "note modal should be initialized")
	require.NotNil(t, app.noteInputModal, "note input modal should be initialized")
	require.NotNil(t, app.taskInputModal, "task input modal should be initialized")

	// Verify initial state
	require.False(t, app.logsVisible, "logs should not be visible initially")
	require.True(t, app.sidebarVisible, "sidebar should be visible initially")
	require.False(t, app.sidebarUserHidden, "sidebar should not be user-hidden initially")
	require.False(t, app.awaitingPrefixKey, "should not be in prefix mode initially")
	require.False(t, app.quitting, "should not be quitting initially")
}

// --- Window Size Tests ---

func TestApp_WindowSizeUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{
			name:   "standard_terminal_size",
			width:  testfixtures.TestTermWidth,
			height: testfixtures.TestTermHeight,
		},
		{
			name:   "narrow_terminal",
			width:  80,
			height: 24,
		},
		{
			name:   "wide_terminal",
			width:  200,
			height: 60,
		},
		{
			name:   "tall_terminal",
			width:  120,
			height: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

			msg := tea.WindowSizeMsg{
				Width:  tt.width,
				Height: tt.height,
			}

			updatedModel, cmd := app.Update(msg)
			updatedApp := updatedModel.(*App)

			// Command can be nil - just verify it doesn't panic
			_ = cmd

			require.Equal(t, tt.width, updatedApp.width, "width should be updated")
			require.Equal(t, tt.height, updatedApp.height, "height should be updated")
		})
	}
}

// --- Sidebar Responsive Behavior Tests ---

func TestApp_ResponsiveSidebarBehavior(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()

	tests := []struct {
		name                string
		initialWidth        int
		targetWidth         int
		userHiddenBefore    bool
		sidebarVisibleAfter bool
		userHiddenAfter     bool
	}{
		{
			name:                "narrowing_below_threshold_auto_hides_sidebar",
			initialWidth:        120,
			targetWidth:         80,
			userHiddenBefore:    false,
			sidebarVisibleAfter: false,
			userHiddenAfter:     false, // Auto-hidden, not user-hidden
		},
		{
			name:                "widening_past_threshold_auto_restores_sidebar",
			initialWidth:        80,
			targetWidth:         120,
			userHiddenBefore:    false,
			sidebarVisibleAfter: true,
			userHiddenAfter:     false,
		},
		{
			name:                "user_hidden_sidebar_stays_hidden_when_narrowing",
			initialWidth:        120,
			targetWidth:         80,
			userHiddenBefore:    true,
			sidebarVisibleAfter: false,
			userHiddenAfter:     true, // Remains user-hidden
		},
		{
			name:                "user_hidden_sidebar_stays_hidden_when_widening",
			initialWidth:        80,
			targetWidth:         120,
			userHiddenBefore:    true,
			sidebarVisibleAfter: false,
			userHiddenAfter:     true, // Remains user-hidden
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)

			// Set initial width
			msg := tea.WindowSizeMsg{Width: tt.initialWidth, Height: 30}
			updatedModel, _ := app.Update(msg)
			app = updatedModel.(*App)

			// Set user-hidden state if needed
			if tt.userHiddenBefore {
				app.sidebarVisible = false
				app.sidebarUserHidden = true
			} else {
				app.sidebarVisible = true
				app.sidebarUserHidden = false
			}

			// Resize to target width
			msg = tea.WindowSizeMsg{Width: tt.targetWidth, Height: 30}
			updatedModel, _ = app.Update(msg)
			app = updatedModel.(*App)

			// Check results
			require.Equal(t, tt.sidebarVisibleAfter, app.sidebarVisible, "sidebar visibility should match expected")
			require.Equal(t, tt.userHiddenAfter, app.sidebarUserHidden, "sidebar user-hidden should match expected")
		})
	}
}

func TestApp_ManualTogglePreservedAcrossResizes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", tmpDir, nil, nil, nil)

	// Start with wide terminal
	msg := tea.WindowSizeMsg{Width: 120, Height: 30}
	updatedModel, _ := app.Update(msg)
	app = updatedModel.(*App)

	// User manually hides sidebar (ctrl+x b)
	updatedModel, _ = app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
	app = updatedModel.(*App)
	updatedModel, _ = app.Update(tea.KeyPressMsg{Text: "b"})
	app = updatedModel.(*App)

	require.False(t, app.sidebarVisible, "sidebar should be hidden after manual toggle")
	require.True(t, app.sidebarUserHidden, "sidebarUserHidden should be true after manual hide")

	// Narrow terminal (should stay hidden)
	msg = tea.WindowSizeMsg{Width: 80, Height: 30}
	updatedModel, _ = app.Update(msg)
	app = updatedModel.(*App)

	require.False(t, app.sidebarVisible, "sidebar should remain hidden when narrowing")
	require.True(t, app.sidebarUserHidden, "sidebarUserHidden should remain true")

	// Widen terminal again (should still stay hidden)
	msg = tea.WindowSizeMsg{Width: 120, Height: 30}
	updatedModel, _ = app.Update(msg)
	app = updatedModel.(*App)

	require.False(t, app.sidebarVisible, "sidebar should remain hidden when widening (user preference)")
	require.True(t, app.sidebarUserHidden, "sidebarUserHidden should remain true")

	// User manually shows sidebar (ctrl+x b again)
	updatedModel, _ = app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
	app = updatedModel.(*App)
	updatedModel, _ = app.Update(tea.KeyPressMsg{Text: "b"})
	app = updatedModel.(*App)

	require.True(t, app.sidebarVisible, "sidebar should be visible after manual toggle")
	require.False(t, app.sidebarUserHidden, "sidebarUserHidden should be false after manual show")
}

// --- Message Handling Tests ---

func TestApp_AgentOutputMessage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	msg := AgentOutputMsg{
		Content: "Test output",
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestApp_IterationStartMessage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	msg := IterationStartMsg{
		Number: 5,
	}

	updatedModel, cmd := app.Update(msg)
	app = updatedModel.(*App)

	// Command can be nil - just verify it doesn't panic
	_ = cmd

	require.Equal(t, 5, app.iteration, "iteration number should be updated")
}

func TestApp_StateUpdateMessage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	state := &session.State{
		Session: testfixtures.FixedSessionName,
		Tasks: map[string]*session.Task{
			"t1": {ID: "t1", Content: "Task 1", Status: "remaining"},
		},
		Notes: []*session.Note{
			{ID: "n1", Content: "Note 1", Type: "learning", Iteration: 1},
		},
	}

	msg := StateUpdateMsg{
		State: state,
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

// --- View Tests ---

func TestApp_ViewProperties(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	view := app.View()

	// Verify view properties are set correctly
	require.True(t, view.AltScreen, "AltScreen should be enabled")
	require.Equal(t, tea.MouseModeCellMotion, view.MouseMode, "MouseMode should be CellMotion")
	require.True(t, view.ReportFocus, "ReportFocus should be enabled")
}

func TestApp_ViewQuitting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.quitting = true

	view := app.View()

	// Verify we get a view back (should not panic)
	require.NotNil(t, view, "view should be returned even when quitting")
}

// --- Quit Tests ---

func TestApp_QuitWithCtrlC(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+c"})

	require.True(t, app.quitting, "app should be marked as quitting")
	require.NotNil(t, cmd, "should return quit command")

	msg := cmd()
	require.IsType(t, tea.QuitMsg{}, msg, "command should return QuitMsg")
}

// --- View Type Tests ---

func TestViewType_Constants(t *testing.T) {
	t.Parallel()

	// Verify view type constants are distinct
	views := []ViewType{
		ViewDashboard,
		ViewLogs,
	}

	seen := make(map[ViewType]bool)
	for _, view := range views {
		require.False(t, seen[view], "view type should be unique: %v", view)
		seen[view] = true
	}

	require.Len(t, seen, 2, "should have exactly 2 distinct view types")
}

// --- Modal Priority Tests ---

func TestApp_ModalCloseOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupModals func(app *App)
		keyPress    string
		verifyClose func(t *testing.T, app *App)
	}{
		{
			name: "close_dialog_first",
			setupModals: func(app *App) {
				app.dialog.Show("Test", "Test message", nil)
				app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test", Status: "remaining", Priority: 1})
			},
			keyPress: "esc",
			verifyClose: func(t *testing.T, app *App) {
				require.False(t, app.dialog.IsVisible(), "dialog should be closed")
				require.True(t, app.taskModal.IsVisible(), "task modal should remain open")
			},
		},
		{
			name: "close_task_modal_when_no_dialog",
			setupModals: func(app *App) {
				app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test", Status: "remaining", Priority: 1})
			},
			keyPress: "esc",
			verifyClose: func(t *testing.T, app *App) {
				require.False(t, app.taskModal.IsVisible(), "task modal should be closed")
			},
		},
		{
			name: "close_note_modal",
			setupModals: func(app *App) {
				app.noteModal.SetNote(&session.Note{ID: "note1", Content: "Test", Type: "learning", Iteration: 1})
			},
			keyPress: "esc",
			verifyClose: func(t *testing.T, app *App) {
				require.False(t, app.noteModal.IsVisible(), "note modal should be closed")
			},
		},
		{
			name: "close_subagent_modal",
			setupModals: func(app *App) {
				app.subagentModal = NewSubagentModal(testfixtures.FixedSessionName, "test-agent", "/tmp")
			},
			keyPress: "esc",
			verifyClose: func(t *testing.T, app *App) {
				require.Nil(t, app.subagentModal, "subagent modal should be closed")
			},
		},
		{
			name: "close_task_input_modal",
			setupModals: func(app *App) {
				app.taskInputModal.Show()
			},
			keyPress: "esc",
			verifyClose: func(t *testing.T, app *App) {
				require.False(t, app.taskInputModal.IsVisible(), "task input modal should be closed")
			},
		},
		{
			name: "close_note_input_modal",
			setupModals: func(app *App) {
				app.noteInputModal.Show()
			},
			keyPress: "esc",
			verifyClose: func(t *testing.T, app *App) {
				require.False(t, app.noteInputModal.IsVisible(), "note input modal should be closed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight
			app.iteration = 1 // Enable modal creation

			tt.setupModals(app)

			_, cmd := app.Update(tea.KeyPressMsg{Text: tt.keyPress})
			// Command can be nil - just verify it doesn't panic
			_ = cmd

			tt.verifyClose(t, app)
		})
	}
}

func TestApp_DialogBlocksOtherModals(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight
	app.iteration = 1

	// Open dialog
	app.dialog.Show("Test", "Test message", nil)

	// Try to open task modal (should be blocked)
	_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
	app.Update(tea.KeyPressMsg{Text: "t"})

	// Command can be nil - just verify it doesn't panic
	_ = cmd

	require.True(t, app.dialog.IsVisible(), "dialog should still be visible")
	require.False(t, app.taskInputModal.IsVisible(), "task input modal should not open when dialog is visible")
}

// --- Logs Toggle Tests ---

func TestApp_LogsToggleWithCtrlXL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		initialState  bool
		expectedFinal bool
	}{
		{
			name:          "open_logs_from_closed",
			initialState:  false,
			expectedFinal: true,
		},
		{
			name:          "close_logs_from_open",
			initialState:  true,
			expectedFinal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.logsVisible = tt.initialState

			// Press ctrl+x l
			updatedModel, _ := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			app = updatedModel.(*App)
			updatedModel, _ = app.Update(tea.KeyPressMsg{Text: "l"})
			app = updatedModel.(*App)

			require.Equal(t, tt.expectedFinal, app.logsVisible, "logs visibility should toggle")
		})
	}
}

// --- Sidebar Toggle Tests ---

func TestApp_SidebarToggleWithCtrlXB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		initialState  bool
		expectedFinal bool
	}{
		{
			name:          "show_sidebar_from_hidden",
			initialState:  false,
			expectedFinal: true,
		},
		{
			name:          "hide_sidebar_from_visible",
			initialState:  true,
			expectedFinal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight
			app.sidebarVisible = tt.initialState

			// Press ctrl+x b
			updatedModel, _ := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			app = updatedModel.(*App)
			updatedModel, _ = app.Update(tea.KeyPressMsg{Text: "b"})
			app = updatedModel.(*App)

			require.Equal(t, tt.expectedFinal, app.sidebarVisible, "sidebar visibility should toggle")
			require.Equal(t, !tt.expectedFinal, app.sidebarUserHidden, "user-hidden should be inverse of visibility")
		})
	}
}

// --- Prefix Key Sequence Tests ---

func TestApp_PrefixKeySequenceFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	// Initially not in prefix mode
	require.False(t, app.awaitingPrefixKey, "should not be in prefix mode initially")

	// Press ctrl+x to enter prefix mode
	updatedModel, _ := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
	app = updatedModel.(*App)

	require.True(t, app.awaitingPrefixKey, "should be in prefix mode after ctrl+x")
	require.True(t, app.status.prefixMode, "status bar should show prefix mode")

	// Press 'l' to toggle logs (ctrl+x l)
	updatedModel, _ = app.Update(tea.KeyPressMsg{Text: "l"})
	app = updatedModel.(*App)

	require.False(t, app.awaitingPrefixKey, "should exit prefix mode after completing sequence")
	require.False(t, app.status.prefixMode, "status bar should clear prefix mode")
	require.True(t, app.logsVisible, "logs should be visible after ctrl+x l")
}

func TestApp_PrefixKeySequenceCancel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	// Press ctrl+x to enter prefix mode
	updatedModel, _ := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
	app = updatedModel.(*App)

	require.True(t, app.awaitingPrefixKey, "should be in prefix mode after ctrl+x")

	// Press esc to cancel prefix mode
	updatedModel, _ = app.Update(tea.KeyPressMsg{Text: "esc"})
	app = updatedModel.(*App)

	require.False(t, app.awaitingPrefixKey, "should exit prefix mode after esc")
	require.False(t, app.logsVisible, "logs should remain hidden after canceling prefix mode")
}

// --- View State Tests ---

func TestApp_ViewWithLogsVisible(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Initialize window size
	app.Update(tea.WindowSizeMsg{Width: testfixtures.TestTermWidth, Height: testfixtures.TestTermHeight})

	// Set logs visible
	app.logsVisible = true

	view := app.View()

	// Verify view is rendered without panicking
	require.NotNil(t, view, "view should be returned")
	require.True(t, view.AltScreen, "AltScreen should be enabled")
}

func TestApp_ViewWithSidebarHidden(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Initialize window size
	app.Update(tea.WindowSizeMsg{Width: testfixtures.TestTermWidth, Height: testfixtures.TestTermHeight})

	// Hide sidebar
	app.sidebarVisible = false

	view := app.View()

	// Verify view is rendered without panicking
	require.NotNil(t, view, "view should be returned")
	require.True(t, view.AltScreen, "AltScreen should be enabled")
}

func TestApp_ViewWithPrefixMode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Initialize window size
	app.Update(tea.WindowSizeMsg{Width: testfixtures.TestTermWidth, Height: testfixtures.TestTermHeight})

	// Enter prefix mode
	app.Update(tea.KeyPressMsg{Text: "ctrl+x"})

	view := app.View()

	// Verify view is rendered without panicking
	require.NotNil(t, view, "view should be returned")
	require.True(t, view.AltScreen, "AltScreen should be enabled")

	// Verify prefix mode state
	require.True(t, app.awaitingPrefixKey, "should be in prefix mode")
	require.True(t, app.status.prefixMode, "status bar should show prefix mode")
}
