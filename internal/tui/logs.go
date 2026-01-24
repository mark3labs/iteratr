package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/session"
)

// LogViewer displays scrollable event history with color-coding.
type LogViewer struct {
	viewport viewport.Model
	state    *session.State
	events   []session.Event // Live event stream
	width    int
	height   int
	focused  bool
}

// Compile-time interface check
var _ FocusableComponent = (*LogViewer)(nil)

// NewLogViewer creates a new LogViewer component.
func NewLogViewer() *LogViewer {
	vp := viewport.New()
	return &LogViewer{
		viewport: vp,
	}
}

// Draw renders the log viewer as a modal overlay.
func (l *LogViewer) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// Calculate modal dimensions (80% of screen, with margins)
	modalWidth := area.Dx() - 4
	modalHeight := area.Dy() - 4
	if modalWidth < 40 {
		modalWidth = area.Dx()
	}
	if modalHeight < 10 {
		modalHeight = area.Dy()
	}

	// Update viewport size to match modal
	contentWidth := modalWidth - 4   // Account for border + padding
	contentHeight := modalHeight - 4 // Account for border + title + padding
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}
	l.viewport.SetWidth(contentWidth)
	l.viewport.SetHeight(contentHeight)

	// Build modal content: title + separator + viewport
	title := renderModalTitle("Event Log", contentWidth)
	separator := styleModalSeparator.Render(strings.Repeat("─", contentWidth))
	vpContent := l.viewport.View()

	// Hint at bottom
	hint := styleModalHint.Render("esc close • ↑/↓ scroll • ctrl+l close")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		separator,
		vpContent,
		hint,
	)

	// Style the modal
	modalStyle := styleModalContainer.Copy().
		Width(modalWidth).
		Height(modalHeight)

	modalContent := modalStyle.Render(content)

	// Center on screen
	renderedWidth := lipgloss.Width(modalContent)
	renderedHeight := lipgloss.Height(modalContent)
	x := (area.Dx() - renderedWidth) / 2
	y := (area.Dy() - renderedHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	modalArea := uv.Rectangle{
		Min: uv.Position{X: area.Min.X + x, Y: area.Min.Y + y},
		Max: uv.Position{X: area.Min.X + x + renderedWidth, Y: area.Min.Y + y + renderedHeight},
	}
	uv.NewStyledString(modalContent).Draw(scr, modalArea)

	return nil
}

// Update handles messages for the log viewer.
func (l *LogViewer) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	return cmd
}

// Render returns the log viewer view as a string.
func (l *LogViewer) Render() string {
	if len(l.events) == 0 {
		l.viewport.SetContent(styleEmptyState.Render("No events yet"))
	}
	return l.viewport.View()
}

// renderEvent renders a single event with appropriate styling.
func (l *LogViewer) renderEvent(event session.Event) string {
	// Format timestamp
	timestamp := event.Timestamp.Format("15:04:05")
	timestampStr := styleLogTimestamp.Render(timestamp)

	// Choose style based on event type
	var typeStyle lipgloss.Style
	var typeLabel string

	switch event.Type {
	case "task":
		typeStyle = styleLogTask
		typeLabel = "TASK"
	case "note":
		typeStyle = styleLogNote
		typeLabel = "NOTE"
	case "iteration":
		typeStyle = styleLogIteration
		typeLabel = "ITER"
	case "control":
		typeStyle = styleLogControl
		typeLabel = "CTRL"
	default:
		typeStyle = styleLogContent
		typeLabel = "EVENT"
	}

	typeStr := typeStyle.Render(fmt.Sprintf("[%s]", typeLabel))

	// Format action
	actionStr := styleDim.Render(event.Action)

	// Format content (truncate if too long)
	content := event.Data
	maxContentWidth := l.width - 30 // Reserve space for timestamp, type, action
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-3] + "..."
	}
	contentStr := styleLogContent.Render(content)

	return fmt.Sprintf("%s %s %-10s %s", timestampStr, typeStr, actionStr, contentStr)
}

// SetSize updates the log viewer dimensions.
func (l *LogViewer) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.viewport.SetWidth(width - 2) // Account for border
	l.viewport.SetHeight(height - 2)
}

// SetState updates the log viewer with new session state.
func (l *LogViewer) SetState(state *session.State) {
	l.state = state
}

// SetFocus updates the focus state.
func (l *LogViewer) SetFocus(focused bool) {
	l.focused = focused
}

// IsFocused returns whether the log viewer is focused.
func (l *LogViewer) IsFocused() bool {
	return l.focused
}

// AddEvent adds a new event to the log viewer.
// This is called when real-time events are received from NATS.
func (l *LogViewer) AddEvent(event session.Event) tea.Cmd {
	l.events = append(l.events, event)
	l.updateContent()
	// Auto-scroll to bottom when new event arrives
	l.viewport.GotoBottom()
	return nil
}

// updateContent rebuilds the viewport content from current events.
func (l *LogViewer) updateContent() {
	if len(l.events) == 0 {
		l.viewport.SetContent(styleEmptyState.Render("No events yet"))
		return
	}

	var b strings.Builder
	for _, event := range l.events {
		b.WriteString(l.renderEvent(event))
		b.WriteString("\n")
	}
	l.viewport.SetContent(b.String())
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
