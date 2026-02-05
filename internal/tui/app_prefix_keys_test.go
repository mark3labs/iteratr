package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestPrefixKeys_ToggleLogs tests ctrl+x l sequence to toggle logs visibility
func TestPrefixKeys_ToggleLogs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		initialState  bool // Initial logsVisible state
		expectedFinal bool // Expected logsVisible after sequence
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
			app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight
			app.logsVisible = tt.initialState

			// Send ctrl+x (enter prefix mode)
			_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			require.NotNil(t, cmd, "Should return no-op command")
			require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

			// Send l (toggle logs)
			_, cmd = app.Update(tea.KeyPressMsg{Text: "l"})
			require.Nil(t, cmd)
			require.False(t, app.awaitingPrefixKey, "Should exit prefix mode after second key")
			require.Equal(t, tt.expectedFinal, app.logsVisible, "Logs visibility should toggle")
		})
	}
}

// TestPrefixKeys_ToggleSidebar tests ctrl+x b sequence to toggle sidebar visibility
func TestPrefixKeys_ToggleSidebar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		initialState  bool // Initial sidebarVisible state
		expectedFinal bool // Expected sidebarVisible after sequence
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
			app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight
			app.sidebarVisible = tt.initialState

			// Send ctrl+x (enter prefix mode)
			_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			require.NotNil(t, cmd, "Should return no-op command")
			require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

			// Send b (toggle sidebar)
			_, cmd = app.Update(tea.KeyPressMsg{Text: "b"})
			require.Nil(t, cmd)
			require.False(t, app.awaitingPrefixKey, "Should exit prefix mode after second key")
			require.Equal(t, tt.expectedFinal, app.sidebarVisible, "Sidebar visibility should toggle")
			require.Equal(t, !tt.expectedFinal, app.sidebarUserHidden, "User-hidden should be inverse of visibility")
		})
	}
}

// TestPrefixKeys_CreateNote tests ctrl+x n sequence to open note creation modal
func TestPrefixKeys_CreateNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		iteration          int
		existingModals     func(app *App) // Setup blocking modals
		expectModalOpen    bool
		expectPrefixActive bool // Special case: dialog blocks prefix mode handling
	}{
		{
			name:               "open_note_modal_normal",
			iteration:          1,
			existingModals:     func(app *App) {}, // No blocking modals
			expectModalOpen:    true,
			expectPrefixActive: false,
		},
		{
			name:               "blocked_by_iteration_zero",
			iteration:          0,
			existingModals:     func(app *App) {},
			expectModalOpen:    false,
			expectPrefixActive: false,
		},
		{
			name:      "blocked_by_dialog",
			iteration: 1,
			existingModals: func(app *App) {
				app.dialog.Show("Test", "Test message", nil)
			},
			expectModalOpen:    false,
			expectPrefixActive: true, // Dialog has priority, so prefix mode stays active
		},
		{
			name:      "blocked_by_task_modal",
			iteration: 1,
			existingModals: func(app *App) {
				app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test", Status: "remaining", Priority: 1})
			},
			expectModalOpen:    false,
			expectPrefixActive: false,
		},
		{
			name:      "blocked_by_logs",
			iteration: 1,
			existingModals: func(app *App) {
				app.logsVisible = true
			},
			expectModalOpen:    false,
			expectPrefixActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight
			app.iteration = tt.iteration
			tt.existingModals(app)

			initialModalState := app.noteInputModal.IsVisible()

			// Send ctrl+x (enter prefix mode)
			_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			require.NotNil(t, cmd, "Should return no-op command")
			require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

			// Send n (create note)
			_, cmd = app.Update(tea.KeyPressMsg{Text: "n"})
			require.Equal(t, tt.expectPrefixActive, app.awaitingPrefixKey, "Prefix mode state should match expectation")

			if tt.expectModalOpen {
				require.NotNil(t, cmd, "Should return Show command")
				require.True(t, app.noteInputModal.IsVisible(), "Note modal should be visible")
			} else {
				require.False(t, app.noteInputModal.IsVisible(), "Note modal should not open when blocked")
				require.Equal(t, initialModalState, app.noteInputModal.IsVisible(), "Modal state should not change")
			}
		})
	}
}

// TestPrefixKeys_CreateTask tests ctrl+x t sequence to open task creation modal
func TestPrefixKeys_CreateTask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		iteration       int
		existingModals  func(app *App) // Setup blocking modals
		expectModalOpen bool
	}{
		{
			name:            "open_task_modal_normal",
			iteration:       1,
			existingModals:  func(app *App) {}, // No blocking modals
			expectModalOpen: true,
		},
		{
			name:            "blocked_by_iteration_zero",
			iteration:       0,
			existingModals:  func(app *App) {},
			expectModalOpen: false,
		},
		{
			name:      "blocked_by_note_input_modal",
			iteration: 1,
			existingModals: func(app *App) {
				app.noteInputModal.Show()
			},
			expectModalOpen: false,
		},
		{
			name:      "blocked_by_note_modal",
			iteration: 1,
			existingModals: func(app *App) {
				app.noteModal.SetNote(&session.Note{ID: "note1", Content: "Test", Type: "learning", Iteration: 1})
			},
			expectModalOpen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight
			app.iteration = tt.iteration
			tt.existingModals(app)

			initialModalState := app.taskInputModal.IsVisible()

			// Send ctrl+x (enter prefix mode)
			_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			require.NotNil(t, cmd, "Should return no-op command")
			require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

			// Send t (create task)
			_, cmd = app.Update(tea.KeyPressMsg{Text: "t"})
			require.False(t, app.awaitingPrefixKey, "Should exit prefix mode after second key")

			if tt.expectModalOpen {
				require.NotNil(t, cmd, "Should return Show command")
				require.True(t, app.taskInputModal.IsVisible(), "Task modal should be visible")
			} else {
				require.False(t, app.taskInputModal.IsVisible(), "Task modal should not open when blocked")
				require.Equal(t, initialModalState, app.taskInputModal.IsVisible(), "Modal state should not change")
			}
		})
	}
}

// TestPrefixKeys_TogglePause tests ctrl+x p sequence to toggle pause state
func TestPrefixKeys_TogglePause(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		orchestrator    *mockOrchestrator
		agentBusy       bool
		expectedPaused  bool
		expectedResumed bool
		expectedCancel  bool
	}{
		{
			name: "request_pause_when_not_paused",
			orchestrator: &mockOrchestrator{
				paused: false,
			},
			agentBusy:      false,
			expectedPaused: true,
		},
		{
			name: "cancel_pause_when_paused_and_working",
			orchestrator: &mockOrchestrator{
				paused: true,
			},
			agentBusy:      true,
			expectedCancel: true,
		},
		{
			name: "resume_when_paused_and_blocked",
			orchestrator: &mockOrchestrator{
				paused: true,
			},
			agentBusy:       false,
			expectedResumed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, tt.orchestrator)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight
			app.dashboard.agentBusy = tt.agentBusy

			// Send ctrl+x (enter prefix mode)
			_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			require.NotNil(t, cmd, "Should return no-op command")
			require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

			// Send p (toggle pause)
			_, cmd = app.Update(tea.KeyPressMsg{Text: "p"})
			require.NotNil(t, cmd, "Should return command from togglePause")
			require.False(t, app.awaitingPrefixKey, "Should exit prefix mode after second key")

			// Verify orchestrator calls
			if tt.expectedPaused {
				require.True(t, tt.orchestrator.pauseRequested, "Should request pause")
			}
			if tt.expectedCancel {
				require.True(t, tt.orchestrator.pauseCancelled, "Should cancel pause")
			}
			if tt.expectedResumed {
				require.True(t, tt.orchestrator.resumed, "Should resume")
			}

			// Execute the command to verify it returns PauseStateMsg
			if cmd != nil {
				msg := cmd()
				require.IsType(t, PauseStateMsg{}, msg, "Command should return PauseStateMsg")
			}
		})
	}
}

// TestPrefixKeys_ExitPrefixMode tests escaping prefix mode with esc or ctrl+c
func TestPrefixKeys_ExitPrefixMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		exitKey   string
		expectMsg tea.Msg
	}{
		{
			name:      "exit_with_esc",
			exitKey:   "esc",
			expectMsg: nil,
		},
		{
			name:      "exit_with_ctrl_c",
			exitKey:   "ctrl+c",
			expectMsg: tea.QuitMsg{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight

			// Send ctrl+x (enter prefix mode)
			_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			require.NotNil(t, cmd, "Should return no-op command")
			require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

			// Send exit key
			_, cmd = app.Update(tea.KeyPressMsg{Text: tt.exitKey})

			if tt.expectMsg != nil {
				// ctrl+c should trigger quit even in prefix mode (global key priority)
				require.NotNil(t, cmd, "Should return quit command")
				msg := cmd()
				require.IsType(t, tt.expectMsg, msg, "Should return QuitMsg")
			} else {
				// esc should just exit prefix mode
				require.Nil(t, cmd, "Should not return command")
				require.False(t, app.awaitingPrefixKey, "Should exit prefix mode")
			}
		})
	}
}

// TestPrefixKeys_InvalidKey tests that invalid keys exit prefix mode without action
func TestPrefixKeys_InvalidKey(t *testing.T) {
	t.Parallel()

	invalidKeys := []string{"a", "z", "1", "space", "enter"}

	for _, key := range invalidKeys {
		t.Run("invalid_key_"+key, func(t *testing.T) {
			app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight
			app.iteration = 1 // Enable modal creation

			initialLogsState := app.logsVisible
			initialSidebarState := app.sidebarVisible

			// Send ctrl+x (enter prefix mode)
			_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			require.NotNil(t, cmd, "Should return no-op command")
			require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

			// Send invalid key
			_, cmd = app.Update(tea.KeyPressMsg{Text: key})
			require.Nil(t, cmd, "Should not return command for invalid key")
			require.False(t, app.awaitingPrefixKey, "Should exit prefix mode after invalid key")

			// Verify no state changes occurred
			require.Equal(t, initialLogsState, app.logsVisible, "Logs state should not change")
			require.Equal(t, initialSidebarState, app.sidebarVisible, "Sidebar state should not change")
			require.False(t, app.noteInputModal.IsVisible(), "Note modal should not open")
			require.False(t, app.taskInputModal.IsVisible(), "Task modal should not open")
		})
	}
}

// TestPrefixKeys_PrefixModePriorityOverModals tests that prefix mode works even with modals open
func TestPrefixKeys_PrefixModePriorityOverModals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupModal   func(app *App)
		prefixAction string // Second key after ctrl+x
		verifyResult func(t *testing.T, app *App)
	}{
		{
			name: "toggle_logs_with_task_modal_open",
			setupModal: func(app *App) {
				app.taskModal.SetTask(&session.Task{ID: "task1", Content: "Test", Status: "remaining", Priority: 1})
			},
			prefixAction: "l",
			verifyResult: func(t *testing.T, app *App) {
				require.True(t, app.logsVisible, "Logs should toggle even with task modal open")
				require.True(t, app.taskModal.IsVisible(), "Task modal should remain open")
			},
		},
		{
			name: "toggle_sidebar_with_note_modal_open",
			setupModal: func(app *App) {
				app.noteModal.SetNote(&session.Note{ID: "note1", Content: "Test", Type: "learning", Iteration: 1})
			},
			prefixAction: "b",
			verifyResult: func(t *testing.T, app *App) {
				require.False(t, app.sidebarVisible, "Sidebar should toggle from default true to false")
				require.True(t, app.noteModal.IsVisible(), "Note modal should remain open")
			},
		},
		{
			name: "toggle_logs_with_subagent_modal_open",
			setupModal: func(app *App) {
				app.subagentModal = NewSubagentModal("test-session", "test-agent", "/tmp")
			},
			prefixAction: "l",
			verifyResult: func(t *testing.T, app *App) {
				require.True(t, app.logsVisible, "Logs should toggle even with subagent modal open")
				require.NotNil(t, app.subagentModal, "Subagent modal should remain open")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
			app.width = testfixtures.TestTermWidth
			app.height = testfixtures.TestTermHeight
			app.iteration = 1
			tt.setupModal(app)

			// Send ctrl+x (enter prefix mode)
			_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
			require.NotNil(t, cmd, "Should return no-op command")
			require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

			// Send prefix action key
			_, _ = app.Update(tea.KeyPressMsg{Text: tt.prefixAction})
			require.False(t, app.awaitingPrefixKey, "Should exit prefix mode after second key")

			// Verify result
			tt.verifyResult(t, app)
		})
	}
}

// TestPrefixKeys_StatusBarIndicator tests that status bar shows prefix mode indicator
func TestPrefixKeys_StatusBarIndicator(t *testing.T) {
	t.Parallel()

	app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Send ctrl+x (enter prefix mode)
	_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
	require.NotNil(t, cmd, "Should return no-op command")
	require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")
	require.True(t, app.status.prefixMode, "Status bar should show prefix mode")

	// Send valid key to exit prefix mode
	_, _ = app.Update(tea.KeyPressMsg{Text: "l"})
	require.False(t, app.awaitingPrefixKey, "Should exit prefix mode after second key")
	require.False(t, app.status.prefixMode, "Status bar should clear prefix mode")
}

// TestPrefixKeys_GlobalKeysPriority tests that ctrl+c works even in prefix mode
func TestPrefixKeys_GlobalKeysPriority(t *testing.T) {
	t.Parallel()

	app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Send ctrl+x (enter prefix mode)
	_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
	require.NotNil(t, cmd, "Should return no-op command")
	require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

	// Send ctrl+c (should quit even in prefix mode)
	_, cmd = app.Update(tea.KeyPressMsg{Text: "ctrl+c"})
	require.NotNil(t, cmd, "Should return quit command")
	msg := cmd()
	require.IsType(t, tea.QuitMsg{}, msg, "Should return QuitMsg")
	require.True(t, app.quitting, "Should mark app as quitting")
}

// TestPrefixKeys_SequenceWithWindowResize tests prefix mode survives window resize
func TestPrefixKeys_SequenceWithWindowResize(t *testing.T) {
	t.Parallel()

	app := NewApp(context.Background(), nil, testfixtures.FixedSessionName, "/tmp", t.TempDir(), nil, nil, nil)
	app.width = testfixtures.TestTermWidth
	app.height = testfixtures.TestTermHeight

	// Send ctrl+x (enter prefix mode)
	_, cmd := app.Update(tea.KeyPressMsg{Text: "ctrl+x"})
	require.NotNil(t, cmd, "Should return no-op command")
	require.True(t, app.awaitingPrefixKey, "Should enter prefix mode after ctrl+x")

	// Send window resize event
	_, cmd = app.Update(tea.WindowSizeMsg{Width: 150, Height: 50})
	require.Nil(t, cmd)
	require.True(t, app.awaitingPrefixKey, "Should remain in prefix mode after resize")
	require.Equal(t, 150, app.width, "Width should update")
	require.Equal(t, 50, app.height, "Height should update")

	// Send l to complete sequence
	_, cmd = app.Update(tea.KeyPressMsg{Text: "l"})
	require.Nil(t, cmd)
	require.False(t, app.awaitingPrefixKey, "Should exit prefix mode after second key")
	require.True(t, app.logsVisible, "Logs should toggle")
}

// mockOrchestrator is a mock implementation of the Orchestrator interface for testing
type mockOrchestrator struct {
	paused         bool
	pauseRequested bool
	pauseCancelled bool
	resumed        bool
}

func (m *mockOrchestrator) RequestPause() {
	m.pauseRequested = true
}

func (m *mockOrchestrator) CancelPause() {
	m.pauseCancelled = true
}

func (m *mockOrchestrator) Resume() {
	m.resumed = true
}

func (m *mockOrchestrator) IsPaused() bool {
	return m.paused
}
