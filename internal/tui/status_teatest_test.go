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

// compareStatusBarGolden compares rendered output against golden file
func compareStatusBarGolden(t *testing.T, canvas uv.ScreenBuffer, name string) {
	t.Helper()
	goldenPath := filepath.Join("testdata", "status_"+name+".golden")
	got := canvas.Render()
	testfixtures.CompareGolden(t, goldenPath, got)
}
