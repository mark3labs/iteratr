package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// DurationTickMsg is sent every second to update the session duration display.
type DurationTickMsg struct{}

// StatusBar displays session info (left) and keybinding hints (right).
type StatusBar struct {
	width             int
	height            int
	sessionName       string
	startedAt         time.Time
	stoppedAt         time.Time // When the duration timer was stopped (zero if running)
	state             *session.State
	connected         bool
	working           bool
	agentBusy         bool // Whether agent is processing (set via AgentBusyMsg, used for pause display)
	paused            bool // Whether orchestrator is paused (set via PauseStateMsg)
	ticking           bool // Whether the spinner tick chain has been started
	needsTick         bool // Whether a tick needs to be started on next Tick() call
	layoutMode        LayoutMode
	spinner           Spinner
	modifiedFileCount int  // Number of files modified in current iteration
	prefixMode        bool // Whether waiting for second key after ctrl+x

	// Git status fields
	gitBranch string // Branch name or "HEAD" if detached
	gitHash   string // Short commit hash (7 chars)
	gitDirty  bool   // Uncommitted changes exist
	gitAhead  int    // Commits ahead of remote
	gitBehind int    // Commits behind remote
	gitValid  bool   // false if not a git repo
}

// NewStatusBar creates a new StatusBar component.
func NewStatusBar(sessionName string) *StatusBar {
	return &StatusBar{
		sessionName: sessionName,
		startedAt:   time.Now(),
		connected:   false,
		working:     false,
		spinner:     NewDefaultSpinner(),
	}
}

// Draw renders the status bar to the screen.
// Format: iteratr | session | Iteration #N [spinner]     ^c quit
func (s *StatusBar) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	if area.Dx() <= 0 || area.Dy() <= 0 {
		return nil
	}

	// Build left side: session info
	left := s.buildLeft()

	// Build right side: keybinding hints
	right := s.buildRight()

	// Calculate spacing to fill width
	totalWidth := area.Dx() - 2 // Account for padding
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	padding := totalWidth - leftWidth - rightWidth
	if padding < 1 {
		padding = 1
	}

	content := left + strings.Repeat(" ", padding) + right

	// Render with style
	DrawStyled(scr, area, theme.Current().S().StatusBar, content)

	return nil
}

// buildLeft builds the left side of the status bar with session info.
func (s *StatusBar) buildLeft() string {
	title := theme.Current().S().HeaderTitle.Render("iteratr")
	sep := theme.Current().S().HeaderSeparator.Render(" | ")
	sessionInfo := theme.Current().S().HeaderInfo.Render(s.sessionName)

	left := title + sep + sessionInfo

	// Add git info if valid (after session name)
	if s.gitValid {
		left += sep + s.buildGitInfo()
	}

	// Use frozen duration if stopped, otherwise calculate from now
	var elapsed time.Duration
	if !s.stoppedAt.IsZero() {
		elapsed = s.stoppedAt.Sub(s.startedAt)
	} else {
		elapsed = time.Since(s.startedAt)
	}
	duration := s.formatDuration(elapsed)
	left += sep + theme.Current().S().HeaderInfo.Render(duration)

	// Add iteration info if available
	if s.state != nil && len(s.state.Iterations) > 0 {
		currentIter := s.state.Iterations[len(s.state.Iterations)-1]
		iterInfo := fmt.Sprintf("Iteration #%d", currentIter.Number)
		left += sep + theme.Current().S().HeaderInfo.Render(iterInfo)
	}

	// Add task stats if tasks exist
	if stats := s.buildTaskStats(); stats != "" {
		left += sep + stats
	}

	// Add modified file count if any files modified
	if s.modifiedFileCount > 0 {
		fileInfo := fmt.Sprintf("%d file", s.modifiedFileCount)
		if s.modifiedFileCount > 1 {
			fileInfo += "s"
		}
		fileInfo += " modified"
		left += sep + theme.Current().S().HeaderInfo.Render(fileInfo)
	}

	// Add spinner when working
	if s.working {
		left += " " + s.spinner.View()
	}

	// Add pause indicator (uses agentBusy, not working, to determine PAUSING vs PAUSED)
	if s.paused {
		if s.agentBusy {
			// Agent still processing: show "PAUSING..." with spinner already visible
			left += " " + theme.Current().S().StatusPausing.Render("PAUSING...")
		} else {
			// Agent idle, orchestrator blocked: show static "⏸ PAUSED" icon
			left += " " + theme.Current().S().StatusPaused.Render("⏸ PAUSED")
		}
	}

	return left
}

// buildGitInfo builds the git status segment: "branch* hash ↑N↓M"
// Asterisk shown only if dirty, arrows shown only if ahead/behind > 0.
func (s *StatusBar) buildGitInfo() string {
	th := theme.Current().S()

	// Branch name with dirty indicator
	branch := s.gitBranch
	if s.gitDirty {
		branch += th.Warning.Render("*")
	}
	result := th.HeaderTitle.Render(branch)

	// Short commit hash
	result += " " + th.HeaderInfo.Render(s.gitHash)

	// Ahead/behind counts (only if non-zero)
	if s.gitAhead > 0 {
		result += " " + th.HeaderInfo.Render(fmt.Sprintf("↑%d", s.gitAhead))
	}
	if s.gitBehind > 0 {
		result += " " + th.HeaderInfo.Render(fmt.Sprintf("↓%d", s.gitBehind))
	}

	return result
}

// buildTaskStats builds a compact task status summary.
// Format: ✓3 ●1 ○5 ✗1 (only non-zero counts shown)
func (s *StatusBar) buildTaskStats() string {
	if s.state == nil || len(s.state.Tasks) == 0 {
		return ""
	}

	var completed, inProgress, remaining, blocked, cancelled int
	for _, task := range s.state.Tasks {
		switch task.Status {
		case "completed":
			completed++
		case "in_progress":
			inProgress++
		case "blocked":
			blocked++
		case "cancelled":
			cancelled++
		default:
			remaining++
		}
	}

	var parts []string
	if completed > 0 {
		parts = append(parts, theme.Current().S().StatusCompleted.Render(fmt.Sprintf("✓ %d", completed)))
	}
	if inProgress > 0 {
		parts = append(parts, theme.Current().S().StatusInProgress.Render(fmt.Sprintf("● %d", inProgress)))
	}
	if remaining > 0 {
		parts = append(parts, theme.Current().S().StatusRemaining.Render(fmt.Sprintf("○ %d", remaining)))
	}
	if blocked > 0 {
		parts = append(parts, theme.Current().S().StatusBlocked.Render(fmt.Sprintf("✗ %d", blocked)))
	}
	if cancelled > 0 {
		parts = append(parts, theme.Current().S().StatusBlocked.Render(fmt.Sprintf("⊘ %d", cancelled)))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}

// buildRight builds the right side with keybinding hints.
func (s *StatusBar) buildRight() string {
	// Show prefix mode indicator when waiting for second key
	if s.prefixMode {
		return theme.Current().S().HintKey.Render("ctrl+x") + " " +
			theme.Current().S().HintDesc.Render("(awaiting key...)")
	}
	return HintStatus()
}

// SetSize updates the component dimensions.
func (s *StatusBar) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetState updates the session state.
func (s *StatusBar) SetState(state *session.State) {
	s.state = state
	wasWorking := s.working
	s.working = s.hasInProgressTasks()

	// Track whether a tick needs to be started
	s.needsTick = s.working && (!wasWorking || !s.ticking)

	// Reset tick chain flag when work stops so it restarts on next work period
	if !s.working {
		s.ticking = false
	}
}

// SetModifiedFileCount updates the count of files modified in the current iteration.
func (s *StatusBar) SetModifiedFileCount(count int) {
	s.modifiedFileCount = count
}

// SetPrefixMode updates whether the app is waiting for a second key after ctrl+x.
func (s *StatusBar) SetPrefixMode(prefixMode bool) {
	s.prefixMode = prefixMode
}

// SetGitInfo updates the git repository status fields.
func (s *StatusBar) SetGitInfo(msg GitInfoMsg) {
	s.gitBranch = msg.Branch
	s.gitHash = msg.Hash
	s.gitDirty = msg.Dirty
	s.gitAhead = msg.Ahead
	s.gitBehind = msg.Behind
	s.gitValid = msg.Valid
}

// Tick returns a command to start the spinner animation if needed.
// Call this after SetState to ensure the spinner starts immediately.
func (s *StatusBar) Tick() tea.Cmd {
	if !s.needsTick {
		return nil
	}
	s.needsTick = false
	s.ticking = true
	return s.spinner.Tick()
}

// SetConnectionStatus updates the connection status.
func (s *StatusBar) SetConnectionStatus(connected bool) {
	s.connected = connected
}

// SetLayoutMode updates the layout mode (desktop/compact).
func (s *StatusBar) SetLayoutMode(mode LayoutMode) {
	s.layoutMode = mode
}

// Update handles messages and spinner animation.
func (s *StatusBar) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case DurationTickMsg:
		// Don't reschedule if the timer has been stopped
		if !s.stoppedAt.IsZero() {
			return nil
		}
		return s.durationTick()
	case PauseStateMsg:
		s.paused = m.Paused
		return nil
	case AgentBusyMsg:
		s.agentBusy = m.Busy
		return nil
	}

	if !s.working {
		return nil
	}

	// Forward to spinner - it returns a cmd only for its own tick messages.
	// The tick chain is started by Tick() after SetState(), and sustained
	// here by returning the spinner's next tick command.
	cmd := s.spinner.Update(msg)
	return cmd
}

// StartDurationTick starts the 1-second duration tick loop.
func (s *StatusBar) StartDurationTick() tea.Cmd {
	return s.durationTick()
}

// StopDurationTick stops the duration timer, freezing the displayed time.
func (s *StatusBar) StopDurationTick() {
	s.stoppedAt = time.Now()
}

func (s *StatusBar) durationTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return DurationTickMsg{}
	})
}

// formatDuration formats a duration as H:MM:SS or M:SS.
func (s *StatusBar) formatDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%d:%02d", m, sec)
}

// hasInProgressTasks checks if there are any in_progress tasks.
func (s *StatusBar) hasInProgressTasks() bool {
	if s.state == nil || s.state.Tasks == nil {
		return false
	}

	for _, task := range s.state.Tasks {
		if task.Status == "in_progress" {
			return true
		}
	}

	return false
}

// Compile-time interface checks
var _ FullComponent = (*StatusBar)(nil)
