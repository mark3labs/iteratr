package specwizard

import (
	"os"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/editor"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui/theme"
	"github.com/mark3labs/iteratr/internal/tui/wizard"
)

// ReviewStep handles the spec review and editing step with markdown rendering.
type ReviewStep struct {
	viewport             viewport.Model // Scrollable viewport for spec display
	content              string         // Raw spec markdown content
	cfg                  *config.Config
	width                int    // Available width
	height               int    // Available height
	tmpFile              string // Path to temp file for editing
	edited               bool   // True if user edited via external editor
	buttonBar            *wizard.ButtonBar
	buttonFocused        bool // True if buttons have focus
	showConfirmRestart   bool // True if restart confirmation modal is visible
	showConfirmOverwrite bool // True if overwrite confirmation modal is visible
}

// NewReviewStep creates a new review step.
func NewReviewStep(content string, cfg *config.Config) *ReviewStep {
	// Create viewport with initial dimensions
	vp := viewport.New(
		viewport.WithWidth(60),
		viewport.WithHeight(10),
	)

	// Enable mouse wheel scrolling
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	// Set rendered markdown content
	vp.SetContent(renderMarkdown(content, 60))

	// Create button bar with Back and Save buttons
	buttons := []wizard.Button{
		{Label: "← Restart", State: wizard.ButtonNormal},
		{Label: "Save", State: wizard.ButtonNormal},
	}
	buttonBar := wizard.NewButtonBar(buttons)

	return &ReviewStep{
		viewport:           vp,
		content:            content,
		cfg:                cfg,
		width:              60,
		height:             20,
		buttonBar:          buttonBar,
		buttonFocused:      false,
		showConfirmRestart: false,
	}
}

// renderMarkdown renders markdown content with syntax highlighting using glamour.
// Falls back to plain text if rendering fails.
func renderMarkdown(content string, width int) string {
	// Cap width to 120 for readability
	if width > 120 {
		width = 120
	}

	// Create glamour renderer with dark theme and word wrap
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		// Fallback to plain text
		return content
	}

	// Render the markdown
	rendered, err := r.Render(content)
	if err != nil {
		// Fallback to plain text
		return content
	}

	// Remove trailing newline that glamour adds
	return strings.TrimSuffix(rendered, "\n")
}

// Init initializes the review step.
func (s *ReviewStep) Init() tea.Cmd {
	return nil
}

// SetSize updates the dimensions for the review step.
func (s *ReviewStep) SetSize(width, height int) {
	s.width = width
	s.height = height

	// Update viewport dimensions (full width)
	s.viewport.SetWidth(width)

	// Reserve space for hint bar (1 line) + button bar (3 lines)
	viewportHeight := height - 4
	if viewportHeight < 5 {
		viewportHeight = 5
	}
	s.viewport.SetHeight(viewportHeight)

	// Re-render markdown with new width
	s.viewport.SetContent(renderMarkdown(s.content, width))

	// Update button bar width
	if s.buttonBar != nil {
		s.buttonBar.SetWidth(width)
	}
}

// Update handles messages for the review step.
func (s *ReviewStep) Update(msg tea.Msg) tea.Cmd {
	// Handle overwrite confirmation modal
	if s.showConfirmOverwrite {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "y", "Y":
				// Confirm overwrite - proceed with save
				s.showConfirmOverwrite = false
				return func() tea.Msg {
					return SaveSpecMsg{}
				}
			case "n", "N", "esc":
				// Cancel overwrite
				s.showConfirmOverwrite = false
				return nil
			}
		}
		return nil
	}

	// Handle restart confirmation modal
	if s.showConfirmRestart {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "y", "Y":
				// Confirm restart - return RestartWizardMsg
				s.showConfirmRestart = false
				return func() tea.Msg {
					return RestartWizardMsg{}
				}
			case "n", "N", "esc":
				// Cancel restart
				s.showConfirmRestart = false
				return nil
			}
		}
		return nil
	}

	// Handle button-focused keyboard input
	if s.buttonFocused && s.buttonBar != nil {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "esc":
				// ESC returns focus to viewport
				s.buttonFocused = false
				s.buttonBar.Blur()
				return nil
			case "tab", "right":
				if !s.buttonBar.FocusNext() {
					// Wrapped around - move focus back to viewport
					s.buttonFocused = false
					s.buttonBar.Blur()
				}
				return nil
			case "shift+tab", "left":
				if !s.buttonBar.FocusPrev() {
					// Wrapped around - move focus back to viewport
					s.buttonFocused = false
					s.buttonBar.Blur()
				}
				return nil
			case "enter", " ":
				return s.activateButton(s.buttonBar.FocusedButton())
			}
		}
		return nil
	}

	// Handle viewport-focused input
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			// ESC key shows restart confirmation modal
			s.showConfirmRestart = true
			return nil
		case "e":
			// Open external editor if $EDITOR is set
			if os.Getenv("EDITOR") != "" {
				return s.openEditor()
			}
		case "tab":
			// Move focus to buttons
			s.buttonFocused = true
			if s.buttonBar != nil {
				s.buttonBar.FocusFirst()
			}
			return nil
		case "shift+tab":
			// Move focus to buttons from end
			s.buttonFocused = true
			if s.buttonBar != nil {
				s.buttonBar.FocusLast()
			}
			return nil
		}
	case SpecEditedMsg:
		// Editor returned with new content
		s.content = msg.Content
		s.edited = true
		s.viewport.SetContent(renderMarkdown(s.content, s.width))
		s.viewport.GotoTop()
		// Clear temp file path (file already cleaned up in callback)
		s.tmpFile = ""
		return nil
	}

	// Forward viewport messages (scrolling, etc.)
	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)
	return cmd
}

// openEditor launches the user's $EDITOR with the spec content.
func (s *ReviewStep) openEditor() tea.Cmd {
	// Create temp file with spec content
	tmpfile, err := os.CreateTemp("", "iteratr_spec_*.md")
	if err != nil {
		return nil // Silently fail - editor not available
	}

	// Write current content to temp file
	if _, err := tmpfile.WriteString(s.content); err != nil {
		_ = tmpfile.Close()
		_ = os.Remove(tmpfile.Name())
		return nil
	}
	_ = tmpfile.Close()

	// Store temp file path for cleanup
	s.tmpFile = tmpfile.Name()

	// Create editor command
	cmd, err := editor.Command("iteratr", tmpfile.Name())
	if err != nil {
		_ = os.Remove(tmpfile.Name())
		return nil
	}

	// Execute editor and read result
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			// Clean up temp file on editor error/cancel
			_ = os.Remove(tmpfile.Name())
			return nil
		}

		// Read modified content
		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			// Clean up temp file on read error
			_ = os.Remove(tmpfile.Name())
			return nil
		}

		// Clean up temp file after successful read
		_ = os.Remove(tmpfile.Name())

		return SpecEditedMsg{
			Content: string(content),
		}
	})
}

// View renders the review step.
func (s *ReviewStep) View() string {
	// If showing overwrite confirmation modal, render it as overlay
	if s.showConfirmOverwrite {
		return s.renderOverwriteConfirmationModal()
	}

	// If showing restart confirmation modal, render it as overlay
	if s.showConfirmRestart {
		return s.renderConfirmationModal()
	}

	var b strings.Builder

	// Render viewport with markdown content
	b.WriteString(s.viewport.View())
	b.WriteString("\n")

	// Button bar
	if s.buttonBar != nil {
		b.WriteString(s.buttonBar.Render())
		b.WriteString("\n")
	}

	// Hint bar - show edit option if $EDITOR is set
	var hintBar string
	if os.Getenv("EDITOR") != "" {
		hintBar = renderHintBar(
			"↑↓", "scroll",
			"e", "edit",
			"tab", "buttons",
			"esc", "back",
		)
	} else {
		hintBar = renderHintBar(
			"↑↓", "scroll",
			"tab", "buttons",
			"esc", "back",
		)
	}
	b.WriteString(hintBar)

	return b.String()
}

// renderOverwriteConfirmationModal renders the file overwrite confirmation modal.
func (s *ReviewStep) renderOverwriteConfirmationModal() string {
	return RenderConfirmationModal(
		"Overwrite Existing Spec?",
		"A spec file with this name already exists. Overwrite it?",
	)
}

// renderConfirmationModal renders the restart confirmation modal.
func (s *ReviewStep) renderConfirmationModal() string {
	t := theme.Current()

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.Warning)).
		MarginBottom(1)
	title := titleStyle.Render("⚠ Restart Wizard?")

	// Message
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgBase)).
		MarginBottom(1)
	message := messageStyle.Render("This will discard the current spec and restart from the beginning.")

	// Buttons
	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgMuted))
	buttons := buttonStyle.Render("Press Y to restart, N or ESC to cancel")

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		message,
		"",
		buttons,
	)

	// Modal styling
	modalStyle := lipgloss.NewStyle().
		Width(50).
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Warning))

	return modalStyle.Render(content)
}

// Content returns the current spec content (possibly edited).
func (s *ReviewStep) Content() string {
	return s.content
}

// WasEdited returns true if the user edited the spec via external editor.
func (s *ReviewStep) WasEdited() bool {
	return s.edited
}

// activateButton handles button activation.
func (s *ReviewStep) activateButton(btnID wizard.ButtonID) tea.Cmd {
	switch btnID {
	case wizard.ButtonBack:
		// Show restart confirmation modal
		s.showConfirmRestart = true
		return nil
	case wizard.ButtonNext:
		// Save button - check if file exists first
		return func() tea.Msg {
			return CheckFileExistsMsg{}
		}
	}
	return nil
}

// SpecEditedMsg is sent when the external editor returns with new content.
type SpecEditedMsg struct {
	Content string
}

// RestartWizardMsg is sent when the user confirms restarting the wizard.
type RestartWizardMsg struct{}

// CheckFileExistsMsg is sent when the user clicks the Save button to check if file exists.
type CheckFileExistsMsg struct{}

// SaveSpecMsg is sent when the user clicks the Save button.
type SaveSpecMsg struct{}
