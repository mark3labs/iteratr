package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// StatusBar displays connection status and current task information.
type StatusBar struct {
	width      int
	height     int
	state      *session.State
	connected  bool
	working    bool
	ticking    bool // Whether the spinner tick chain has been started
	layoutMode LayoutMode
	spinner    Spinner
}

// NewStatusBar creates a new StatusBar component.
func NewStatusBar() *StatusBar {
	return &StatusBar{
		connected: false,
		working:   false,
		spinner:   NewDefaultSpinner(),
	}
}

// Draw renders the status bar to the screen.
// Format: [spinner] ● connected
// When idle (not working), spinner is hidden. When working, shows animated spinner.
func (s *StatusBar) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	if area.Dx() <= 0 || area.Dy() <= 0 {
		return nil
	}

	// Build status content based on layout mode
	var content string

	// Add spinner only when working (not when idle)
	if s.working {
		content += s.spinner.View() + " "
	}

	// Add connection status (condensed in compact mode)
	connStatus := s.getConnectionStatus()
	content += connStatus

	// Truncate if too long (more aggressive in compact mode)
	maxWidth := area.Dx() - 2 // Account for padding
	if lipgloss.Width(content) > maxWidth {
		content = truncateString(content, maxWidth)
	}

	// Render with style
	DrawStyled(scr, area, styleStatusBar, content)

	return nil
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

// getConnectionStatus returns the connection status indicator.
// ● = connected, ○ = disconnected
func (s *StatusBar) getConnectionStatus() string {
	if s.layoutMode == LayoutCompact {
		// Compact mode: just show the dot
		if s.connected {
			return lipgloss.NewStyle().Foreground(colorSuccess).Render("●")
		}
		return lipgloss.NewStyle().Foreground(colorError).Render("○")
	}

	// Desktop mode: show full text
	if s.connected {
		return lipgloss.NewStyle().Foreground(colorSuccess).Render("●") + " connected"
	}
	return lipgloss.NewStyle().Foreground(colorError).Render("○") + " disconnected"
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
