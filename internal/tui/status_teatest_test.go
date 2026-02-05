package tui

import (
	"path/filepath"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
)

// TestStatusBar_InitialState verifies the initial state of a new StatusBar
func TestStatusBar_InitialState(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	if sb.sessionName != testfixtures.FixedSessionName {
		t.Errorf("sessionName: got %q, want %q", sb.sessionName, testfixtures.FixedSessionName)
	}
	if sb.connected {
		t.Error("connected should be false initially")
	}
	if sb.working {
		t.Error("working should be false initially")
	}
	// Spinner is initialized by NewStatusBar via NewDefaultSpinner()
}

// TestStatusBar_SetState_WithInProgressTask tests SetState with an in_progress task
func TestStatusBar_SetState_WithInProgressTask(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)

	state := testfixtures.StateWithTasks()
	sb.SetState(state)

	if !sb.working {
		t.Error("working should be true with in_progress task")
	}
	if !sb.needsTick {
		t.Error("needsTick should be true after SetState with in_progress task")
	}

	// Tick should return command
	cmd := sb.Tick()
	if cmd == nil {
		t.Error("Tick should return command when working")
	}
	if !sb.ticking {
		t.Error("ticking should be true after Tick")
	}
}

// TestStatusBar_SetState_NoInProgressTask tests SetState with no in_progress tasks
func TestStatusBar_SetState_NoInProgressTask(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)

	state := testfixtures.StateWithCompletedSession()
	sb.SetState(state)

	if sb.working {
		t.Error("working should be false with no in_progress tasks")
	}
	if sb.needsTick {
		t.Error("needsTick should be false with no in_progress tasks")
	}

	// Tick should return nil
	cmd := sb.Tick()
	if cmd != nil {
		t.Error("Tick should return nil when not working")
	}
}

// TestStatusBar_SpinnerStopsWhenWorkCompletes tests spinner stops when task completes
func TestStatusBar_SpinnerStopsWhenWorkCompletes(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)

	// Start with in_progress task
	state := testfixtures.StateWithTasks()
	sb.SetState(state)

	// Verify working
	if !sb.working {
		t.Fatal("working should be true initially")
	}

	// Start ticking
	cmd := sb.Tick()
	if cmd == nil {
		t.Fatal("Tick should return command when working")
	}

	// Now complete all tasks by changing status
	for _, task := range state.Tasks {
		task.Status = "completed"
	}
	sb.SetState(state)

	// Verify no longer working
	if sb.working {
		t.Error("working should be false after all tasks completed")
	}
	if sb.ticking {
		t.Error("ticking should be false after work stops")
	}

	// Tick should return nil
	cmd = sb.Tick()
	if cmd != nil {
		t.Error("Tick should return nil when no longer working")
	}
}

// TestStatusBar_SetSize verifies SetSize updates dimensions
func TestStatusBar_SetSize(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetSize(100, 1)

	if sb.width != 100 {
		t.Errorf("width: got %d, want 100", sb.width)
	}
	if sb.height != 1 {
		t.Errorf("height: got %d, want 1", sb.height)
	}
}

// TestStatusBar_SetConnectionStatus verifies connection status changes
func TestStatusBar_SetConnectionStatus(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	sb.SetConnectionStatus(true)
	if !sb.connected {
		t.Error("connected should be true after SetConnectionStatus(true)")
	}

	sb.SetConnectionStatus(false)
	if sb.connected {
		t.Error("connected should be false after SetConnectionStatus(false)")
	}
}

// TestStatusBar_SetLayoutMode verifies layout mode changes
func TestStatusBar_SetLayoutMode(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	sb.SetLayoutMode(LayoutDesktop)
	if sb.layoutMode != LayoutDesktop {
		t.Errorf("layoutMode: got %v, want LayoutDesktop", sb.layoutMode)
	}

	sb.SetLayoutMode(LayoutCompact)
	if sb.layoutMode != LayoutCompact {
		t.Errorf("layoutMode: got %v, want LayoutCompact", sb.layoutMode)
	}
}

// TestStatusBar_SetModifiedFileCount verifies file count updates
func TestStatusBar_SetModifiedFileCount(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	sb.SetModifiedFileCount(5)
	if sb.modifiedFileCount != 5 {
		t.Errorf("modifiedFileCount: got %d, want 5", sb.modifiedFileCount)
	}
}

// TestStatusBar_SetPrefixMode verifies prefix mode changes
func TestStatusBar_SetPrefixMode(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	sb.SetPrefixMode(true)
	if !sb.prefixMode {
		t.Error("prefixMode should be true after SetPrefixMode(true)")
	}

	sb.SetPrefixMode(false)
	if sb.prefixMode {
		t.Error("prefixMode should be false after SetPrefixMode(false)")
	}
}

// TestStatusBar_SetSidebarHidden verifies sidebar hidden state changes
func TestStatusBar_SetSidebarHidden(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	sb.SetSidebarHidden(true)
	if !sb.sidebarHidden {
		t.Error("sidebarHidden should be true after SetSidebarHidden(true)")
	}

	sb.SetSidebarHidden(false)
	if sb.sidebarHidden {
		t.Error("sidebarHidden should be false after SetSidebarHidden(false)")
	}
}

// TestStatusBar_SetGitInfo verifies git info updates
func TestStatusBar_SetGitInfo(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	msg := GitInfoMsg{
		Branch: "main",
		Hash:   "abc1234",
		Dirty:  true,
		Ahead:  2,
		Behind: 1,
		Valid:  true,
	}

	sb.SetGitInfo(msg)

	if sb.gitBranch != "main" {
		t.Errorf("gitBranch: got %q, want %q", sb.gitBranch, "main")
	}
	if sb.gitHash != "abc1234" {
		t.Errorf("gitHash: got %q, want %q", sb.gitHash, "abc1234")
	}
	if !sb.gitDirty {
		t.Error("gitDirty should be true")
	}
	if sb.gitAhead != 2 {
		t.Errorf("gitAhead: got %d, want 2", sb.gitAhead)
	}
	if sb.gitBehind != 1 {
		t.Errorf("gitBehind: got %d, want 1", sb.gitBehind)
	}
	if !sb.gitValid {
		t.Error("gitValid should be true")
	}
}

// TestStatusBar_UpdatePauseStateMsg verifies pause state updates
func TestStatusBar_UpdatePauseStateMsg(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	cmd := sb.Update(PauseStateMsg{Paused: true})
	if cmd != nil {
		t.Error("Update should return nil for PauseStateMsg")
	}
	if !sb.paused {
		t.Error("paused should be true after PauseStateMsg{Paused: true}")
	}

	cmd = sb.Update(PauseStateMsg{Paused: false})
	if cmd != nil {
		t.Error("Update should return nil for PauseStateMsg")
	}
	if sb.paused {
		t.Error("paused should be false after PauseStateMsg{Paused: false}")
	}
}

// TestStatusBar_UpdateAgentBusyMsg verifies agent busy state updates
func TestStatusBar_UpdateAgentBusyMsg(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	cmd := sb.Update(AgentBusyMsg{Busy: true})
	if cmd != nil {
		t.Error("Update should return nil for AgentBusyMsg")
	}
	if !sb.agentBusy {
		t.Error("agentBusy should be true after AgentBusyMsg{Busy: true}")
	}

	cmd = sb.Update(AgentBusyMsg{Busy: false})
	if cmd != nil {
		t.Error("Update should return nil for AgentBusyMsg")
	}
	if sb.agentBusy {
		t.Error("agentBusy should be false after AgentBusyMsg{Busy: false}")
	}
}

// TestStatusBar_UpdateGitInfoMsg verifies git info message handling
func TestStatusBar_UpdateGitInfoMsg(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	msg := GitInfoMsg{
		Branch: "develop",
		Hash:   "xyz9876",
		Dirty:  false,
		Ahead:  0,
		Behind: 3,
		Valid:  true,
	}

	cmd := sb.Update(msg)
	if cmd != nil {
		t.Error("Update should return nil for GitInfoMsg")
	}

	if sb.gitBranch != "develop" {
		t.Errorf("gitBranch: got %q, want %q", sb.gitBranch, "develop")
	}
	if sb.gitHash != "xyz9876" {
		t.Errorf("gitHash: got %q, want %q", sb.gitHash, "xyz9876")
	}
	if sb.gitDirty {
		t.Error("gitDirty should be false")
	}
	if sb.gitAhead != 0 {
		t.Errorf("gitAhead: got %d, want 0", sb.gitAhead)
	}
	if sb.gitBehind != 3 {
		t.Errorf("gitBehind: got %d, want 3", sb.gitBehind)
	}
}

// TestStatusBar_DurationTick verifies duration tick behavior
func TestStatusBar_DurationTick(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	// Start duration tick
	cmd := sb.StartDurationTick()
	if cmd == nil {
		t.Fatal("StartDurationTick should return command")
	}

	// Update with duration tick message
	cmd = sb.Update(DurationTickMsg{})
	if cmd == nil {
		t.Error("Update with DurationTickMsg should return next tick command")
	}

	// Stop duration tick
	sb.StopDurationTick()

	// Update with duration tick after stop should return nil
	cmd = sb.Update(DurationTickMsg{})
	if cmd != nil {
		t.Error("Update with DurationTickMsg after stop should return nil")
	}
}

// TestStatusBar_FormatDuration verifies duration formatting
func TestStatusBar_FormatDuration(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			want:     "0:45",
		},
		{
			name:     "minutes and seconds",
			duration: 5*time.Minute + 30*time.Second,
			want:     "5:30",
		},
		{
			name:     "hours, minutes, seconds",
			duration: 2*time.Hour + 15*time.Minute + 42*time.Second,
			want:     "2:15:42",
		},
		{
			name:     "zero duration",
			duration: 0,
			want:     "0:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sb.formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v): got %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

// TestStatusBar_HasInProgressTasks verifies in_progress task detection
func TestStatusBar_HasInProgressTasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state *session.State
		want  bool
	}{
		{
			name:  "nil state",
			state: nil,
			want:  false,
		},
		{
			name:  "empty state",
			state: testfixtures.EmptyState(),
			want:  false,
		},
		{
			name:  "with in_progress task",
			state: testfixtures.StateWithTasks(),
			want:  true,
		},
		{
			name:  "all tasks completed",
			state: testfixtures.StateWithCompletedSession(),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBar(testfixtures.FixedSessionName)
			sb.state = tt.state
			got := sb.hasInProgressTasks()
			if got != tt.want {
				t.Errorf("hasInProgressTasks(): got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatusBar_Render_EmptyState tests rendering with empty state
func TestStatusBar_Render_EmptyState(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time

	canvas := uv.NewScreenBuffer(150, 1)
	area := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "empty_state")
}

// TestStatusBar_Render_WithTasks tests rendering with tasks
func TestStatusBar_Render_WithTasks(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time
	sb.SetState(testfixtures.StateWithTasks())

	canvas := uv.NewScreenBuffer(150, 1)
	area := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "with_tasks")
}

// TestStatusBar_Render_WithGitInfo tests rendering with git information
func TestStatusBar_Render_WithGitInfo(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time

	sb.SetGitInfo(GitInfoMsg{
		Branch: "main",
		Hash:   testfixtures.FixedGitHash,
		Dirty:  true,
		Ahead:  2,
		Behind: 1,
		Valid:  true,
	})

	canvas := uv.NewScreenBuffer(150, 1)
	area := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "with_git_info")
}

// TestStatusBar_Render_WithGitClean tests rendering with clean git state
func TestStatusBar_Render_WithGitClean(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time

	sb.SetGitInfo(GitInfoMsg{
		Branch: "develop",
		Hash:   testfixtures.FixedGitHash,
		Dirty:  false,
		Ahead:  0,
		Behind: 0,
		Valid:  true,
	})

	canvas := uv.NewScreenBuffer(150, 1)
	area := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "with_git_clean")
}

// TestStatusBar_Render_PrefixMode tests rendering in prefix mode
func TestStatusBar_Render_PrefixMode(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time
	sb.SetPrefixMode(true)

	canvas := uv.NewScreenBuffer(150, 1)
	area := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "prefix_mode")
}

// TestStatusBar_Render_SidebarHidden tests rendering with sidebar hidden
func TestStatusBar_Render_SidebarHidden(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time
	sb.SetSidebarHidden(true)

	canvas := uv.NewScreenBuffer(150, 1)
	area := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "sidebar_hidden")
}

// TestStatusBar_Render_Paused tests rendering in paused state
func TestStatusBar_Render_Paused(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time
	sb.paused = true
	sb.agentBusy = false // PAUSED state

	canvas := uv.NewScreenBuffer(150, 1)
	area := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "paused")
}

// TestStatusBar_Render_Pausing tests rendering in pausing state
func TestStatusBar_Render_Pausing(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time
	sb.paused = true
	sb.agentBusy = true // PAUSING state (agent still processing)
	sb.SetState(testfixtures.StateWithTasks())

	canvas := uv.NewScreenBuffer(150, 1)
	area := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "pausing")
}

// TestStatusBar_Render_WithModifiedFiles tests rendering with modified files count
func TestStatusBar_Render_WithModifiedFiles(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time
	sb.SetState(testfixtures.StateWithTasks())
	sb.SetModifiedFileCount(5)

	canvas := uv.NewScreenBuffer(150, 1)
	area := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "with_modified_files")
}

// TestStatusBar_Render_FullState tests rendering with all features enabled
func TestStatusBar_Render_FullState(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time
	sb.SetState(testfixtures.FullState())
	sb.SetModifiedFileCount(3)
	sb.SetGitInfo(GitInfoMsg{
		Branch: "feature/teatest",
		Hash:   testfixtures.FixedGitHash,
		Dirty:  true,
		Ahead:  1,
		Behind: 0,
		Valid:  true,
	})

	canvas := uv.NewScreenBuffer(200, 1)
	area := uv.Rect(0, 0, 200, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "full_state")
}

// TestStatusBar_Render_SmallWidth tests rendering on narrow terminal
func TestStatusBar_Render_SmallWidth(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime // Freeze duration at fixed time
	sb.SetState(testfixtures.StateWithTasks())

	canvas := uv.NewScreenBuffer(80, 1)
	area := uv.Rect(0, 0, 80, 1)
	sb.Draw(canvas, area)

	compareStatusBarGolden(t, canvas, "small_width")
}

// TestStatusBar_PauseTransition_PausingToPaused tests PAUSING→PAUSED state transition
func TestStatusBar_PauseTransition_PausingToPaused(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.SetState(testfixtures.StateWithTasks())

	// Step 1: Request pause while agent is busy (PAUSING state)
	cmd := sb.Update(PauseStateMsg{Paused: true})
	if cmd != nil {
		t.Error("Update should return nil for PauseStateMsg")
	}
	if !sb.paused {
		t.Error("paused should be true after pause request")
	}

	// Verify working due to in_progress task
	if !sb.working {
		t.Fatal("working should be true with in_progress task")
	}

	// Initially agent is busy
	sb.agentBusy = true
	if !sb.paused || !sb.agentBusy {
		t.Fatal("should be in PAUSING state (paused=true, agentBusy=true)")
	}

	// Step 2: Agent finishes work (PAUSED state)
	cmd = sb.Update(AgentBusyMsg{Busy: false})
	if cmd != nil {
		t.Error("Update should return nil for AgentBusyMsg")
	}
	if sb.agentBusy {
		t.Error("agentBusy should be false after agent finishes")
	}

	// Verify now in PAUSED state
	if !sb.paused || sb.agentBusy {
		t.Error("should be in PAUSED state (paused=true, agentBusy=false)")
	}
}

// TestStatusBar_PauseTransition_CancelWhilePausing tests canceling pause during PAUSING
func TestStatusBar_PauseTransition_CancelWhilePausing(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.SetState(testfixtures.StateWithTasks())

	// Step 1: Request pause while agent is busy (PAUSING state)
	sb.paused = true
	sb.agentBusy = true

	if !sb.paused || !sb.agentBusy {
		t.Fatal("should be in PAUSING state")
	}

	// Step 2: Cancel pause request
	cmd := sb.Update(PauseStateMsg{Paused: false})
	if cmd != nil {
		t.Error("Update should return nil for PauseStateMsg")
	}
	if sb.paused {
		t.Error("paused should be false after cancel")
	}

	// Agent still working
	if !sb.agentBusy {
		t.Error("agentBusy should still be true")
	}

	// Verify back to working state (not paused, agent busy)
	if sb.paused {
		t.Error("should not be paused after cancel")
	}
}

// TestStatusBar_PauseTransition_ResumeFromPaused tests resuming from PAUSED state
func TestStatusBar_PauseTransition_ResumeFromPaused(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)

	// Start in PAUSED state (agent idle, orchestrator blocked)
	sb.paused = true
	sb.agentBusy = false

	if !sb.paused || sb.agentBusy {
		t.Fatal("should be in PAUSED state")
	}

	// Resume
	cmd := sb.Update(PauseStateMsg{Paused: false})
	if cmd != nil {
		t.Error("Update should return nil for PauseStateMsg")
	}
	if sb.paused {
		t.Error("paused should be false after resume")
	}
	if sb.agentBusy {
		t.Error("agentBusy should still be false")
	}
}

// TestStatusBar_PauseTransition_MultipleToggleCycles tests rapid pause/resume cycles
func TestStatusBar_PauseTransition_MultipleToggleCycles(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)

	// Cycle 1: Pause → Resume
	sb.Update(PauseStateMsg{Paused: true})
	if !sb.paused {
		t.Error("cycle 1: should be paused")
	}

	sb.Update(PauseStateMsg{Paused: false})
	if sb.paused {
		t.Error("cycle 1: should not be paused")
	}

	// Cycle 2: Pause → Resume
	sb.Update(PauseStateMsg{Paused: true})
	if !sb.paused {
		t.Error("cycle 2: should be paused")
	}

	sb.Update(PauseStateMsg{Paused: false})
	if sb.paused {
		t.Error("cycle 2: should not be paused")
	}

	// Cycle 3: Pause → Resume
	sb.Update(PauseStateMsg{Paused: true})
	if !sb.paused {
		t.Error("cycle 3: should be paused")
	}

	sb.Update(PauseStateMsg{Paused: false})
	if sb.paused {
		t.Error("cycle 3: should not be paused")
	}
}

// TestStatusBar_PauseTransition_WithNoTasks tests pause state when no tasks exist
func TestStatusBar_PauseTransition_WithNoTasks(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.SetState(testfixtures.EmptyState()) // No tasks

	// Verify not working
	if sb.working {
		t.Fatal("should not be working with no tasks")
	}

	// Request pause (should work even with no tasks)
	cmd := sb.Update(PauseStateMsg{Paused: true})
	if cmd != nil {
		t.Error("Update should return nil for PauseStateMsg")
	}
	if !sb.paused {
		t.Error("should be paused even with no tasks")
	}

	// Since no work, should immediately be in PAUSED state (not PAUSING)
	sb.agentBusy = false
	if sb.working || sb.agentBusy {
		t.Error("should not be working or agent busy with no tasks")
	}

	// Resume
	sb.Update(PauseStateMsg{Paused: false})
	if sb.paused {
		t.Error("should not be paused after resume")
	}
}

// TestStatusBar_PauseTransition_SpinnerStopsOnPaused tests spinner stops when transitioning to PAUSED
func TestStatusBar_PauseTransition_SpinnerStopsOnPaused(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.SetState(testfixtures.StateWithTasks())

	// Start working (spinner should be active)
	if !sb.working {
		t.Fatal("should be working with in_progress task")
	}

	// Start spinner tick
	cmd := sb.Tick()
	if cmd == nil {
		t.Fatal("Tick should return command when working")
	}
	if !sb.ticking {
		t.Fatal("ticking should be true after Tick")
	}

	// Request pause while working (PAUSING state)
	sb.paused = true
	sb.agentBusy = true

	// Verify still working (spinner should continue during PAUSING)
	if !sb.working {
		t.Error("should still be working during PAUSING")
	}

	// Transition to PAUSED (agent finishes)
	sb.Update(AgentBusyMsg{Busy: false})

	// Now complete all tasks to stop working
	state := testfixtures.StateWithTasks()
	for _, task := range state.Tasks {
		task.Status = "completed"
	}
	sb.SetState(state)

	// Verify working stopped
	if sb.working {
		t.Error("working should stop when tasks complete")
	}
	if sb.ticking {
		t.Error("ticking should stop when work stops")
	}

	// Tick should return nil
	cmd = sb.Tick()
	if cmd != nil {
		t.Error("Tick should return nil when not working")
	}
}

// TestStatusBar_PauseTransition_StateChangesDuringPause tests state changes while paused
func TestStatusBar_PauseTransition_StateChangesDuringPause(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.SetState(testfixtures.StateWithTasks())

	// Enter PAUSED state
	sb.paused = true
	sb.agentBusy = false

	// Verify in PAUSED state
	if !sb.paused || sb.agentBusy {
		t.Fatal("should be in PAUSED state")
	}

	// Modify state while paused (complete a task)
	state := testfixtures.StateWithTasks()
	if task, ok := state.Tasks["TAS-1"]; ok {
		task.Status = "completed" // Complete one task
	}
	sb.SetState(state)

	// Should still have in_progress tasks
	if !sb.working {
		t.Error("should still be working with remaining in_progress task")
	}

	// Pause state should persist through state updates
	if !sb.paused {
		t.Error("paused should remain true during state update")
	}
}

// TestStatusBar_PauseTransition_AgentBusyCyclesDuringPause tests agentBusy cycling during pause
func TestStatusBar_PauseTransition_AgentBusyCyclesDuringPause(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)

	// Start in PAUSING state
	sb.paused = true
	sb.agentBusy = true

	// Agent finishes (PAUSED)
	sb.Update(AgentBusyMsg{Busy: false})
	if sb.agentBusy {
		t.Error("agentBusy should be false")
	}
	if !sb.paused {
		t.Error("paused should remain true")
	}

	// Agent starts again (hypothetical - shouldn't happen but test robustness)
	sb.Update(AgentBusyMsg{Busy: true})
	if !sb.agentBusy {
		t.Error("agentBusy should be true")
	}
	if !sb.paused {
		t.Error("paused should remain true")
	}

	// Agent finishes again (back to PAUSED)
	sb.Update(AgentBusyMsg{Busy: false})
	if sb.agentBusy {
		t.Error("agentBusy should be false again")
	}
	if !sb.paused {
		t.Error("paused should still be true")
	}
}

// TestStatusBar_Render_PauseTransitionSequence tests visual output during pause transition
func TestStatusBar_Render_PauseTransitionSequence(t *testing.T) {
	t.Parallel()

	sb := NewStatusBar(testfixtures.FixedSessionName)
	sb.SetLayoutMode(LayoutDesktop)
	sb.startedAt = testfixtures.FixedTime
	sb.stoppedAt = testfixtures.FixedTime
	sb.SetState(testfixtures.StateWithTasks())

	// Test sequence: Working → PAUSING → PAUSED → Resumed

	// State 1: Working (not paused)
	sb.paused = false
	sb.agentBusy = true
	canvas1 := uv.NewScreenBuffer(150, 1)
	area1 := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas1, area1)

	render1 := canvas1.Render()
	if !testfixtures.Contains(render1, "● 1") {
		t.Error("working state should show in_progress task indicator")
	}
	if testfixtures.Contains(render1, "PAUSING") || testfixtures.Contains(render1, "PAUSED") {
		t.Error("working state should not show pause indicators")
	}

	// State 2: PAUSING (pause requested, agent still busy)
	sb.paused = true
	sb.agentBusy = true
	canvas2 := uv.NewScreenBuffer(150, 1)
	area2 := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas2, area2)

	render2 := canvas2.Render()
	if !testfixtures.Contains(render2, "PAUSING") {
		t.Error("PAUSING state should show PAUSING indicator")
	}
	if testfixtures.Contains(render2, "⏸ PAUSED") {
		t.Error("PAUSING state should not show static PAUSED icon")
	}

	// State 3: PAUSED (agent idle, orchestrator blocked)
	sb.paused = true
	sb.agentBusy = false
	canvas3 := uv.NewScreenBuffer(150, 1)
	area3 := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas3, area3)

	render3 := canvas3.Render()
	if !testfixtures.Contains(render3, "⏸ PAUSED") {
		t.Error("PAUSED state should show PAUSED icon")
	}
	if testfixtures.Contains(render3, "PAUSING") {
		t.Error("PAUSED state should not show PAUSING indicator")
	}

	// State 4: Resumed (not paused, agent busy again)
	sb.paused = false
	sb.agentBusy = true
	canvas4 := uv.NewScreenBuffer(150, 1)
	area4 := uv.Rect(0, 0, 150, 1)
	sb.Draw(canvas4, area4)

	render4 := canvas4.Render()
	if testfixtures.Contains(render4, "PAUSING") || testfixtures.Contains(render4, "PAUSED") {
		t.Error("resumed state should not show pause indicators")
	}
}

// contains checks if a string contains a substring
func compareStatusBarGolden(t *testing.T, canvas uv.ScreenBuffer, name string) {
	t.Helper()
	goldenPath := filepath.Join("testdata", "status_"+name+".golden")
	got := canvas.Render()
	testfixtures.CompareGolden(t, goldenPath, got)
}
