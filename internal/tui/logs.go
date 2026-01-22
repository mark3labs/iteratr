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

// Draw renders the log viewer to the screen buffer.
func (l *LogViewer) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// Draw panel border with title
	inner := DrawPanel(scr, area, "Event Log", l.focused)

	// Draw viewport content
	content := l.viewport.View()
	DrawText(scr, inner, content)

	// Draw scroll indicator if content overflows
	if l.viewport.TotalLineCount() > l.viewport.Height() {
		percent := l.viewport.ScrollPercent()
		DrawScrollIndicator(scr, area, percent)
	}

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
	case "inbox":
		typeStyle = styleLogInbox
		typeLabel = "INBOX"
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
