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
// Format: [indicator] Working | Current Task Name
func (s *StatusBar) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	if area.Dx() <= 0 || area.Dy() <= 0 {
		return nil
	}

	// Build status content based on layout mode
	var content string

	// Add working indicator
	indicator := s.getWorkingIndicator()
	content += indicator + " "

	// Add connection status (condensed in compact mode)
	connStatus := s.getConnectionStatus()
	content += connStatus

	// Add current task if available
	if s.state != nil {
		currentTask := s.getCurrentTask()
		if currentTask != "" {
			content += " │ " + currentTask
		}
	}

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
	wasWorking := s.working
	s.working = s.hasInProgressTasks()

	// Start spinner animation when work begins
	// Note: The tick command will be returned by Update() on next message
	if s.working && !wasWorking {
		// Working state changed to true - spinner will start on next Update
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
	// Update spinner if working
	if s.working {
		cmd := s.spinner.Update(msg)
		if cmd != nil {
			return cmd
		}
		// Ensure spinner keeps ticking
		return s.spinner.Tick()
	}
	return nil
}

// getWorkingIndicator returns the appropriate working indicator.
// Uses animated spinner when working, ○ when idle
func (s *StatusBar) getWorkingIndicator() string {
	if s.working {
		// Show animated spinner
		return s.spinner.View()
	}
	return styleDim.Render("○")
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

// getCurrentTask returns the name of the current in_progress task.
func (s *StatusBar) getCurrentTask() string {
	if s.state == nil || s.state.Tasks == nil {
		return ""
	}

	// Find first in_progress task
	for _, task := range s.state.Tasks {
		if task.Status == "in_progress" {
			// Truncate task content based on layout mode
			content := task.Content
			maxLen := 50
			if s.layoutMode == LayoutCompact {
				maxLen = 30 // More aggressive truncation in compact mode
			}

			if len(content) > maxLen {
				content = content[:maxLen-3] + "..."
			}
			return "task: " + content
		}
	}

	return ""
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
