package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// StatusBar displays session info (left) and keybinding hints (right).
type StatusBar struct {
	width       int
	height      int
	sessionName string
	state       *session.State
	connected   bool
	working     bool
	ticking     bool // Whether the spinner tick chain has been started
	layoutMode  LayoutMode
	spinner     Spinner
}

// NewStatusBar creates a new StatusBar component.
func NewStatusBar(sessionName string) *StatusBar {
	return &StatusBar{
		sessionName: sessionName,
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

	left := title + sep + sessionInfo

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
		parts = append(parts, styleStatusCompleted.Render(fmt.Sprintf("✓%d", completed)))
	}
	if inProgress > 0 {
		parts = append(parts, styleStatusInProgress.Render(fmt.Sprintf("●%d", inProgress)))
	}
	if remaining > 0 {
		parts = append(parts, styleStatusRemaining.Render(fmt.Sprintf("○%d", remaining)))
	}
	if blocked > 0 {
		parts = append(parts, styleStatusBlocked.Render(fmt.Sprintf("✗%d", blocked)))
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
	// Update working state based on in_progress tasks
	s.working = s.hasInProgressTasks()

	// Reset tick chain flag when work stops so it restarts on next work period
	if !s.working {
		s.ticking = false
	}
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
	if !s.working {
		return nil
	}

	// Forward to spinner - it returns a cmd only for its own tick messages
	cmd := s.spinner.Update(msg)
	if cmd != nil {
		return cmd // Spinner handled its tick, returns next tick (self-sustaining chain)
	}

	// Start the tick chain once when working becomes true
	if !s.ticking {
		s.ticking = true
		return s.spinner.Tick()
	}

	return nil
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

// truncateString truncates a string to fit within maxWidth, adding "..." if truncated.
func truncateString(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return "..."
	}

	width := lipgloss.Width(s)
	if width <= maxWidth {
		return s
	}

	// Simple truncation - count runes to handle multi-byte chars
	runes := []rune(s)
	targetLen := maxWidth - 3 // Reserve space for "..."

	if targetLen < 0 {
		targetLen = 0
	}

	if targetLen >= len(runes) {
		return s
	}

	return string(runes[:targetLen]) + "..."
}

// Compile-time interface checks
var _ FullComponent = (*StatusBar)(nil)
