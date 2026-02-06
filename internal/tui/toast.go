package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// ToastDismissMsg is sent when the toast should be dismissed.
type ToastDismissMsg struct {
	Generation int
}

// ShowToastMsg is sent to show a toast notification.
type ShowToastMsg struct {
	Text string
}

// Toast is a minimal toast notification component.
// Shows a message in the bottom-right corner that auto-dismisses after 3 seconds.
type Toast struct {
	message    string
	visible    bool
	dismissAt  time.Time
	generation int
}

// NewToast creates a new Toast component.
func NewToast() *Toast {
	return &Toast{}
}

// Show displays a toast with the given message.
// The toast will auto-dismiss after 3 seconds.
func (t *Toast) Show(msg string) tea.Cmd {
	t.message = msg
	t.visible = true
	t.generation++
	t.dismissAt = time.Now().Add(3 * time.Second)
	return t.dismissCmd()
}

// dismissCmd returns a command that will dismiss the toast after the remaining time.
func (t *Toast) dismissCmd() tea.Cmd {
	remaining := time.Until(t.dismissAt)
	if remaining <= 0 {
		remaining = 1 * time.Millisecond
	}
	generation := t.generation
	return tea.Tick(remaining, func(time.Time) tea.Msg {
		return ToastDismissMsg{Generation: generation}
	})
}

// Update handles messages for the toast component.
// Returns a command to re-schedule dismissal if needed.
func (t *Toast) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case ToastDismissMsg:
		// Only dismiss if generation matches (prevents stale dismissals)
		if m.Generation == t.generation {
			t.visible = false
			t.message = ""
		}
		return nil
	}
	return nil
}

// View renders the toast content with styling.
// Returns empty string if toast is not visible.
// Positioning is handled by the caller (app.go Draw method).
func (t *Toast) View(width, height int) string {
	if !t.visible || t.message == "" {
		return ""
	}

	th := theme.Current()

	// Style the toast with warning colors (yellow) to indicate notification
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.FgBase)).
		Background(lipgloss.Color(th.Warning)).
		Padding(0, 1).
		Bold(true)

	content := style.Render(t.message)

	// Clamp width if needed (leave room for padding from edges)
	contentWidth := lipgloss.Width(content)
	if contentWidth > width-2 {
		content = style.Width(width - 2).Render(t.message)
	}

	return content
}

// IsVisible returns whether the toast is currently visible.
func (t *Toast) IsVisible() bool {
	return t.visible
}

// GetMessage returns the current toast message (empty if not visible).
func (t *Toast) GetMessage() string {
	if !t.visible {
		return ""
	}
	return t.message
}
