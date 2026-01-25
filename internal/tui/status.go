package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// DurationTickMsg is sent every second to update the session duration display.
type DurationTickMsg struct{}

// StatusBar displays session info (left) and keybinding hints (right).
type StatusBar struct {
	width       int
	height      int
	sessionName string
	startedAt   time.Time
	stoppedAt   time.Time // When the duration timer was stopped (zero if running)
	state       *session.State
	connected   bool
	working     bool
	ticking     bool // Whether the spinner tick chain has been started
	needsTick   bool // Whether a tick needs to be started on next Tick() call
	layoutMode  LayoutMode
	spinner     Spinner
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
	DrawStyled(scr, area, styleStatusBar, content)

	return nil
}

// buildLeft builds the left side of the status bar with session info.
func (s *StatusBar) buildLeft() string {
	title := styleHeaderTitle.Render("iteratr")
	sep := styleHeaderSeparator.Render(" | ")
	sessionInfo := styleHeaderInfo.Render(s.sessionName)

	// Use frozen duration if stopped, otherwise calculate from now
	var elapsed time.Duration
	if !s.stoppedAt.IsZero() {
		elapsed = s.stoppedAt.Sub(s.startedAt)
	} else {
		elapsed = time.Since(s.startedAt)
	}
	duration := s.formatDuration(elapsed)
	left := title + sep + sessionInfo + sep + styleHeaderInfo.Render(duration)

	// Add iteration info if available
	if s.state != nil && len(s.state.Iterations) > 0 {
		currentIter := s.state.Iterations[len(s.state.Iterations)-1]
		iterInfo := fmt.Sprintf("Iteration #%d", currentIter.Number)
		left += sep + styleHeaderInfo.Render(iterInfo)
	}

	// Add task stats if tasks exist
	if stats := s.buildTaskStats(); stats != "" {
		left += sep + stats
	}

	// Add spinner when working
	if s.working {
		left += " " + s.spinner.View()
	}

	return left
}

// buildTaskStats builds a compact task status summary.
// Format: ✓3 ●1 ○5 ✗1 (only non-zero counts shown)
func (s *StatusBar) buildTaskStats() string {
	if s.state == nil || len(s.state.Tasks) == 0 {
		return ""
	}

	var completed, inProgress, remaining, blocked int
	for _, task := range s.state.Tasks {
		switch task.Status {
		case "completed":
			completed++
		case "in_progress":
			inProgress++
		case "blocked":
			blocked++
		default:
			remaining++
		}
	}

	var parts []string
	if completed > 0 {
		parts = append(parts, styleStatusCompleted.Render(fmt.Sprintf("✓ %d", completed)))
	}
	if inProgress > 0 {
		parts = append(parts, styleStatusInProgress.Render(fmt.Sprintf("● %d", inProgress)))
	}
	if remaining > 0 {
		parts = append(parts, styleStatusRemaining.Render(fmt.Sprintf("○ %d", remaining)))
	}
	if blocked > 0 {
		parts = append(parts, styleStatusBlocked.Render(fmt.Sprintf("✗ %d", blocked)))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}

// buildRight builds the right side with keybinding hints.
func (s *StatusBar) buildRight() string {
	hintKey := lipgloss.NewStyle().Foreground(colorSubtext0)
	hintDesc := lipgloss.NewStyle().Foreground(colorOverlay0)
	sep := "  "

	return hintKey.Render("ctrl+l") + hintDesc.Render(" logs") + sep +
		hintKey.Render("ctrl+c") + hintDesc.Render(" quit")
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
	switch msg.(type) {
	case DurationTickMsg:
		// Don't reschedule if the timer has been stopped
		if !s.stoppedAt.IsZero() {
			return nil
		}
		return s.durationTick()
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
