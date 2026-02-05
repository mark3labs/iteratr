package tui

import (
	"context"
	"testing"
	"time"

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

// --- Command Execution Tests ---
// These tests verify that App.Update() returns the correct commands for various messages.
// Commands are executed to verify they return the expected message types.

func TestApp_OpenTaskModalMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.iteration = 1 // Required for modal creation

	task := &session.Task{
		ID:       "task1",
		Content:  "Test task",
		Status:   "remaining",
		Priority: 2,
	}

	// Create OpenTaskModalMsg
	msg := OpenTaskModalMsg{
		Task: task,
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify modal is visible with the task
	require.True(t, app.taskModal.IsVisible(), "task modal should be visible after OpenTaskModalMsg")
	require.Equal(t, task, app.taskModal.task, "task modal should contain the correct task")

	// Verify command is nil
	require.Nil(t, cmd, "command should be nil for OpenTaskModalMsg")
}

func TestApp_FileChangeMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Initially no modified files
	require.Equal(t, 0, app.modifiedFileCount, "modifiedFileCount should be 0 initially")

	// Create FileChangeMsg
	msg := FileChangeMsg{
		Path:      "/tmp/test.go",
		IsNew:     false,
		Additions: 5,
		Deletions: 2,
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify modified file count incremented
	require.Equal(t, 1, app.modifiedFileCount, "modifiedFileCount should be incremented")

	// Verify command returned (status tick + git fetch)
	require.NotNil(t, cmd, "command should be returned for FileChangeMsg")
}

func TestApp_OpenSubagentModalMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	// Create OpenSubagentModalMsg
	msg := OpenSubagentModalMsg{
		SessionID:    "subagent-session-123",
		SubagentType: "codebase-analyzer",
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify subagent modal is created
	require.NotNil(t, app.subagentModal, "subagent modal should be created")
	require.Equal(t, "subagent-session-123", app.subagentModal.sessionID, "subagent modal should have correct session ID")
	require.Equal(t, "codebase-analyzer", app.subagentModal.subagentType, "subagent modal should have correct subagent type")

	// Verify command returned (modal.Start())
	require.NotNil(t, cmd, "command should be returned for OpenSubagentModalMsg")
}

func TestApp_UserInputMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sendChan := make(chan string, 10)
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, sendChan, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Initially queue depth is 0
	require.Equal(t, 0, app.queueDepth, "queueDepth should be 0 initially")

	// Create UserInputMsg
	msg := UserInputMsg{
		Text: "test user message",
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify queue depth incremented
	require.Equal(t, 1, app.queueDepth, "queueDepth should be incremented")

	// Verify message sent to channel
	select {
	case sentMsg := <-sendChan:
		require.Equal(t, "test user message", sentMsg, "message should be sent to sendChan")
	default:
		t.Fatal("message should be sent to sendChan")
	}

	// Command may be returned (SetQueueDepth) or nil depending on dashboard state
	_ = cmd
}

func TestApp_QueuedMessageProcessingMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight
	app.queueDepth = 3 // Set initial queue depth

	// Create QueuedMessageProcessingMsg
	msg := QueuedMessageProcessingMsg{
		Text: "processing message",
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify queue depth decremented
	require.Equal(t, 2, app.queueDepth, "queueDepth should be decremented")

	// Command may be returned (batch of AppendUserMessage + SetQueueDepth) or nil
	_ = cmd
}

func TestApp_AgentOutputMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Create AgentOutputMsg
	msg := AgentOutputMsg{
		Content: "Agent output text",
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify command returned (AppendText returns command to add message item)
	// Note: agent.AppendText may return nil depending on state
	_ = cmd
}

func TestApp_AgentThinkingMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Create AgentThinkingMsg
	msg := AgentThinkingMsg{
		Content: "Agent thinking content",
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify command returned (AppendThinking returns command to add thinking item)
	// Note: agent.AppendThinking may return nil depending on state
	_ = cmd
}

func TestApp_AgentFinishMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	// Create AgentFinishMsg
	msg := AgentFinishMsg{
		Reason:   "end_turn",
		Model:    "claude-sonnet-4",
		Provider: "anthropic",
		Duration: 5 * time.Second,
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify command returned (batch of AppendFinish + SetAgentBusy + AgentBusyMsg)
	require.NotNil(t, cmd, "command should be returned for AgentFinishMsg")
}

func TestApp_IterationStartMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	// Create IterationStartMsg
	msg := IterationStartMsg{
		Number: 10,
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify iteration updated
	require.Equal(t, 10, app.iteration, "iteration should be updated")
	require.Equal(t, 0, app.modifiedFileCount, "modifiedFileCount should be reset")

	// Verify command returned (batch of SetIteration + AddIterationDivider + SetAgentBusy + AgentBusyMsg + fetchGitInfo)
	require.NotNil(t, cmd, "command should be returned for IterationStartMsg")
}

func TestApp_StateUpdateMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	state := &session.State{
		Session: testfixtures.FixedSessionName,
		Tasks: map[string]*session.Task{
			"t1": {ID: "t1", Content: "Task 1", Status: "remaining"},
		},
		Notes: []*session.Note{
			{ID: "n1", Content: "Note 1", Type: "learning", Iteration: 1},
		},
	}

	// Create StateUpdateMsg
	msg := StateUpdateMsg{
		State: state,
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify command returned (status.Tick returns DurationTickMsg)
	// Note: status.Tick may return nil depending on ticker state
	_ = cmd
}

func TestApp_ConnectionStatusMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	// Create ConnectionStatusMsg
	msg := ConnectionStatusMsg{
		Connected: true,
	}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify command returned (checkConnectionHealth)
	require.NotNil(t, cmd, "command should be returned for ConnectionStatusMsg")
}

func TestApp_SessionCompleteMsg_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// Use nil store - loadInitialState will handle nil gracefully
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

	// Create SessionCompleteMsg
	msg := SessionCompleteMsg{}

	// Send message to App
	_, cmd := app.Update(msg)

	// Verify dialog is shown
	require.True(t, app.dialog.IsVisible(), "dialog should be visible after SessionCompleteMsg")

	// Verify command returned (loadInitialState)
	require.NotNil(t, cmd, "command should be returned for SessionCompleteMsg")
}

func TestApp_GlobalKeys_Command(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		keyPress       string
		expectQuit     bool
		expectPrefix   bool
		expectCmd      bool
		expectedStatus bool
	}{
		{
			name:           "ctrl+c quits",
			keyPress:       "ctrl+c",
			expectQuit:     true,
			expectPrefix:   false,
			expectCmd:      true,
			expectedStatus: false,
		},
		{
			name:           "ctrl+x enters prefix mode",
			keyPress:       "ctrl+x",
			expectQuit:     false,
			expectPrefix:   true,
			expectCmd:      true,
			expectedStatus: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)

			// Send key press
			_, cmd := app.Update(tea.KeyPressMsg{Text: tt.keyPress})

			// Verify quit state
			require.Equal(t, tt.expectQuit, app.quitting, "quitting state mismatch")

			// Verify prefix state
			require.Equal(t, tt.expectPrefix, app.awaitingPrefixKey, "prefix state mismatch")
			require.Equal(t, tt.expectedStatus, app.status.prefixMode, "status prefix mode mismatch")

			// Verify command presence
			if tt.expectCmd {
				require.NotNil(t, cmd, "command should be returned")
			}

			// For quit, verify command returns QuitMsg
			if tt.expectQuit && cmd != nil {
				msg := cmd()
				_, ok := msg.(tea.QuitMsg)
				require.True(t, ok, "command should return QuitMsg for ctrl+c")
			}
		})
	}
}

func TestApp_TogglePause_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockOrch := testfixtures.NewMockOrchestrator()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, mockOrch)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Enter prefix mode
	app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
	require.True(t, app.awaitingPrefixKey, "should be in prefix mode")

	// Press 'p' to toggle pause
	_, cmd := app.Update(tea.KeyPressMsg{Text: "p"})

	// Verify prefix mode exited
	require.False(t, app.awaitingPrefixKey, "should exit prefix mode")

	// Verify pause was requested
	require.True(t, mockOrch.WasPauseRequested(), "pause should be requested")

	// Verify command returned (PauseStateMsg)
	require.NotNil(t, cmd, "command should be returned")

	// Execute command to verify PauseStateMsg
	if cmd != nil {
		msg := cmd()
		pauseMsg, ok := msg.(PauseStateMsg)
		require.True(t, ok, "command should return PauseStateMsg")
		require.True(t, pauseMsg.Paused, "PauseStateMsg.Paused should be true")
	}
}

func TestApp_HandleSidebarToggle_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight
	app.sidebarVisible = true

	// Enter prefix mode
	app.Update(tea.KeyPressMsg{Text: "ctrl+x"})

	// Press 'b' to toggle sidebar
	updatedModel, cmd := app.Update(tea.KeyPressMsg{Text: "b"})
	app = updatedModel.(*App)

	// Verify sidebar toggled
	require.False(t, app.sidebarVisible, "sidebar should be hidden")
	require.True(t, app.sidebarUserHidden, "sidebarUserHidden should be true")

	// Verify layout marked dirty
	require.True(t, app.layoutDirty, "layoutDirty should be true")

	// Verify command is nil (handleSidebarToggle returns nil)
	require.Nil(t, cmd, "command should be nil for sidebar toggle")
}

func TestApp_ToggleLogs_Command(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	app := NewApp(ctx, nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.logsVisible = false

	// Enter prefix mode
	app.Update(tea.KeyPressMsg{Text: "ctrl+x"})

	// Press 'l' to toggle logs
	updatedModel, cmd := app.Update(tea.KeyPressMsg{Text: "l"})
	app = updatedModel.(*App)

	// Verify logs toggled
	require.True(t, app.logsVisible, "logs should be visible")

	// Verify prefix mode exited
	require.False(t, app.awaitingPrefixKey, "prefix mode should be exited")

	// Verify command is nil (logs toggle returns nil)
	require.Nil(t, cmd, "command should be nil for logs toggle")
}
