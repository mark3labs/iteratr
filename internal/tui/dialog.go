package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// Dialog represents a modal dialog overlay
type Dialog struct {
	title      string
	message    string
	button     string
	visible    bool
	width      int
	height     int
	onClose    func() tea.Cmd
	dialogArea uv.Rectangle // Screen area where dialog is drawn (for mouse hit detection)
}

// NewDialog creates a new dialog
func NewDialog() *Dialog {
	return &Dialog{
		visible: false,
		button:  "OK",
	}
}

// Show displays the dialog with the given title and message
func (d *Dialog) Show(title, message string, onClose func() tea.Cmd) {
	d.title = title
	d.message = message
	d.visible = true
	d.onClose = onClose
}

// Hide closes the dialog
func (d *Dialog) Hide() {
	d.visible = false
}

// IsVisible returns whether the dialog is visible
func (d *Dialog) IsVisible() bool {
	return d.visible
}

// SetSize updates the dialog's knowledge of screen size
func (d *Dialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Update handles dialog input
func (d *Dialog) Update(msg tea.Msg) tea.Cmd {
	if !d.visible {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter", "space", "esc":
			d.Hide()
			if d.onClose != nil {
				return d.onClose()
			}
		}
	}
	return nil
}

// Draw renders the dialog centered on screen
func (d *Dialog) Draw(scr uv.Screen, area uv.Rectangle) {
	if !d.visible {
		return
	}

	// Calculate content width first for consistent alignment
	messageWidth := lipgloss.Width(d.message)
	titleWidth := lipgloss.Width(d.title)
	contentWidth := messageWidth
	if titleWidth > contentWidth {
		contentWidth = titleWidth
	}

	t := theme.Current()
	s := t.S()

	// Title styling - just colored text, no background
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center)

	// Message styling
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgBase)).
		Width(contentWidth).
		Align(lipgloss.Center)

	// Button styling - inline padding only
	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.BgBase)).
		Background(lipgloss.Color(t.Primary)).
		Padding(0, 2)

	// Build content parts
	title := titleStyle.Render(d.title)
	message := messageStyle.Render(d.message)
	button := buttonStyle.Render(d.button)

	// Center the button within content width
	buttonLine := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(button)

	// Join vertically
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		message,
		"",
		buttonLine,
	)

	// Dialog box styling - use theme styles
	dialog := s.ModalContainer.
		Width(lipgloss.Width(content)).
		Render(content)

	// Calculate center position
	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := lipgloss.Height(dialog)
	x := (area.Dx() - dialogWidth) / 2
	y := (area.Dy() - dialogHeight) / 2

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// Create centered area for dialog
	d.dialogArea = uv.Rectangle{
		Min: uv.Position{X: area.Min.X + x, Y: area.Min.Y + y},
		Max: uv.Position{X: area.Min.X + x + dialogWidth, Y: area.Min.Y + y + dialogHeight},
	}

	// Draw dialog
	uv.NewStyledString(dialog).Draw(scr, d.dialogArea)
}

// HandleClick processes a mouse click. Clicking anywhere dismisses the dialog.
func (d *Dialog) HandleClick(x, y int) tea.Cmd {
	if !d.visible {
		return nil
	}
	d.Hide()
	if d.onClose != nil {
		return d.onClose()
	}
	return nil
}

// SessionCompleteMsg is sent when the session is marked complete
type SessionCompleteMsg struct{}
