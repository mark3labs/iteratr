package wizard

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/iteratr/internal/tui"
)

// ConfigStep manages the session configuration UI step.
type ConfigStep struct {
	sessionInput    textinput.Model // Session name input
	iterationsInput textinput.Model // Max iterations input
	focusIndex      int             // 0=session, 1=iterations
	sessionError    string          // Validation error for session name
	iterationsError string          // Validation error for iterations
	width           int             // Available width
	height          int             // Available height
	sessionStore    *session.Store  // Session store for uniqueness check
}

// NewConfigStep creates a new config step with smart defaults.
func NewConfigStep(specPath string, sessionStore *session.Store) *ConfigStep {
	// Initialize session name input
	sessionInput := textinput.New()
	sessionInput.Placeholder = "Enter session name..."
	sessionInput.Prompt = ""

	// Configure styles for textinput (using lipgloss v2)
	sessionStyles := textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("#b4befe")),
		},
		Blurred: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")),
		},
		Cursor: textinput.CursorStyle{
			Color: lipgloss.Color("#cba6f7"),
			Shape: tea.CursorBar,
			Blink: true,
		},
	}
	sessionInput.SetStyles(sessionStyles)
	sessionInput.SetWidth(50)

	// Smart default: derive from spec filename
	defaultName := deriveSessionName(specPath)
	sessionInput.SetValue(defaultName)

	// Initialize iterations input
	iterationsInput := textinput.New()
	iterationsInput.Placeholder = "0"
	iterationsInput.Prompt = ""
	iterationsInput.SetStyles(sessionStyles)
	iterationsInput.SetWidth(50)
	iterationsInput.SetValue("0") // Default: infinite

	return &ConfigStep{
		sessionInput:    sessionInput,
		iterationsInput: iterationsInput,
		focusIndex:      0, // Start with session input focused
		width:           60,
		height:          10,
		sessionStore:    sessionStore,
	}
}

// deriveSessionName creates a sanitized session name from the spec file path.
// Matches the logic in build.go:70-76
func deriveSessionName(specPath string) string {
	if specPath == "" {
		return ""
	}

	// Extract filename without extension
	base := filepath.Base(specPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Replace dots with hyphens (NATS subject constraint)
	name = strings.ReplaceAll(name, ".", "-")

	// Sanitize: only keep alphanumeric, hyphens, underscores
	var sanitized strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			sanitized.WriteRune(r)
		}
	}

	return sanitized.String()
}

// Init initializes the config step and focuses the session input.
func (c *ConfigStep) Init() tea.Cmd {
	c.focusIndex = 0
	return c.sessionInput.Focus()
}

// Focus gives focus to the config step (first input).
func (c *ConfigStep) Focus() tea.Cmd {
	c.focusIndex = 0
	c.iterationsInput.Blur()
	return c.sessionInput.Focus()
}

// FocusLast gives focus to the config step's last input.
func (c *ConfigStep) FocusLast() tea.Cmd {
	c.focusIndex = 1
	c.sessionInput.Blur()
	return c.iterationsInput.Focus()
}

// Blur removes focus from all inputs in the config step.
func (c *ConfigStep) Blur() {
	c.sessionInput.Blur()
	c.iterationsInput.Blur()
}

// SetSize updates the dimensions for the config step.
func (c *ConfigStep) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.sessionInput.SetWidth(width - 10)
	c.iterationsInput.SetWidth(width - 10)
}

// Update handles messages for the config step.
func (c *ConfigStep) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	// Handle paste messages - intercept and collapse newlines for single-line inputs
	if pasteMsg, ok := msg.(tea.PasteMsg); ok {
		// Sanitize and collapse newlines
		sanitized := tui.SanitizePaste(pasteMsg.Content)
		collapsed := collapseNewlines(sanitized)

		// Create modified paste message with collapsed content
		modifiedMsg := tea.PasteMsg{Content: collapsed}

		// Forward to focused input
		var cmd tea.Cmd
		if c.focusIndex == 0 {
			c.sessionInput, cmd = c.sessionInput.Update(modifiedMsg)
			cmds = append(cmds, cmd)
		} else {
			c.iterationsInput, cmd = c.iterationsInput.Update(modifiedMsg)
			cmds = append(cmds, cmd)
		}
		return tea.Batch(cmds...)
	}

	// Handle keyboard input
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "tab":
			// Cycle focus forward, or signal exit if at last input
			if c.focusIndex == 1 {
				// At last input - signal to move to buttons
				return func() tea.Msg {
					return TabExitForwardMsg{}
				}
			}
			c.cycleFocusForward()
			return nil

		case "shift+tab":
			// Cycle focus backward, or signal exit if at first input
			if c.focusIndex == 0 {
				// At first input - signal to move to buttons from end
				return func() tea.Msg {
					return TabExitBackwardMsg{}
				}
			}
			c.cycleFocusBackward()
			return nil

		case "enter":
			// Validate and advance
			if c.validate() {
				// Config is valid - this will be handled by parent wizard
				return func() tea.Msg {
					return ConfigCompleteMsg{}
				}
			}
			return nil
		}
	}

	// Forward messages to focused input
	var cmd tea.Cmd
	if c.focusIndex == 0 {
		c.sessionInput, cmd = c.sessionInput.Update(msg)
		cmds = append(cmds, cmd)
		// Clear error on input change
		if _, ok := msg.(tea.KeyPressMsg); ok {
			c.sessionError = ""
		}
	} else {
		c.iterationsInput, cmd = c.iterationsInput.Update(msg)
		cmds = append(cmds, cmd)
		// Clear error on input change
		if _, ok := msg.(tea.KeyPressMsg); ok {
			c.iterationsError = ""
		}
	}

	return tea.Batch(cmds...)
}

// cycleFocusForward moves focus to the next input (wraps around).
func (c *ConfigStep) cycleFocusForward() {
	c.focusIndex = (c.focusIndex + 1) % 2
	c.updateFocus()
}

// cycleFocusBackward moves focus to the previous input (wraps around).
func (c *ConfigStep) cycleFocusBackward() {
	c.focusIndex = (c.focusIndex - 1 + 2) % 2
	c.updateFocus()
}

// updateFocus updates the focus state of both inputs.
func (c *ConfigStep) updateFocus() {
	if c.focusIndex == 0 {
		c.sessionInput.Focus()
		c.iterationsInput.Blur()
	} else {
		c.sessionInput.Blur()
		c.iterationsInput.Focus()
	}
}

// View renders the config step.
func (c *ConfigStep) View() string {
	var b strings.Builder

	// Session name section
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
	b.WriteString(labelStyle.Render("Session Name"))
	b.WriteString("\n")
	b.WriteString(c.sessionInput.View())
	b.WriteString("\n")

	// Show session validation error
	if c.sessionError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
		b.WriteString(errorStyle.Render("✗ " + c.sessionError))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Iterations section
	b.WriteString(labelStyle.Render("Max Iterations (0 = infinite)"))
	b.WriteString("\n")
	b.WriteString(c.iterationsInput.View())
	b.WriteString("\n")

	// Show iterations validation error
	if c.iterationsError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
		b.WriteString(errorStyle.Render("✗ " + c.iterationsError))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Hint bar
	hintBar := renderHintBar(
		"tab", "next/buttons",
		"enter", "finish",
		"esc", "back",
	)
	b.WriteString(hintBar)

	return b.String()
}

// validate validates both inputs and sets error messages.
// Returns true if all inputs are valid.
func (c *ConfigStep) validate() bool {
	valid := true

	// Validate session name
	sessionName := strings.TrimSpace(c.sessionInput.Value())
	if sessionName == "" {
		c.sessionError = "Session name cannot be empty"
		valid = false
	} else if len(sessionName) > 64 {
		c.sessionError = "Session name too long (max 64 characters)"
		valid = false
	} else {
		// Check for invalid characters
		for _, r := range sessionName {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' {
				c.sessionError = "Use only alphanumeric, hyphens, underscores"
				valid = false
				break
			}
		}

		// Check for uniqueness if store is available
		if valid && c.sessionStore != nil {
			ctx := context.Background()
			sessions, err := c.sessionStore.ListSessions(ctx)
			if err == nil {
				for _, s := range sessions {
					if s.Name == sessionName {
						c.sessionError = "Session name already exists"
						valid = false
						break
					}
				}
			}
			// Silently ignore errors loading sessions (e.g., store not connected)
		}
	}

	// Validate iterations
	iterationsStr := strings.TrimSpace(c.iterationsInput.Value())
	if iterationsStr == "" {
		c.iterationsError = "Iterations cannot be empty (use 0 for infinite)"
		valid = false
	} else {
		iterations, err := strconv.Atoi(iterationsStr)
		if err != nil {
			c.iterationsError = "Must be a valid number"
			valid = false
		} else if iterations < 0 {
			c.iterationsError = "Must be >= 0 (0 means infinite)"
			valid = false
		}
	}

	return valid
}

// IsValid returns true if the current config is valid.
func (c *ConfigStep) IsValid() bool {
	return c.validate()
}

// SessionName returns the validated session name.
func (c *ConfigStep) SessionName() string {
	return strings.TrimSpace(c.sessionInput.Value())
}

// Iterations returns the validated iterations count.
func (c *ConfigStep) Iterations() int {
	iterationsStr := strings.TrimSpace(c.iterationsInput.Value())
	iterations, err := strconv.Atoi(iterationsStr)
	if err != nil {
		return 0
	}
	return iterations
}

// Cursor returns the cursor from the currently focused input, with Y offset
// adjusted for lines above the input within this step's View output.
func (c *ConfigStep) Cursor() *tea.Cursor {
	var cur *tea.Cursor
	switch c.focusIndex {
	case 0:
		cur = c.sessionInput.Cursor()
		if cur != nil {
			// Session input is below the "Session Name" label (1 line)
			cur.Y += 1
		}
	case 1:
		cur = c.iterationsInput.Cursor()
		if cur != nil {
			// Lines above iterations input:
			// "Session Name" label: 1
			// session input: 1
			// session error (if present): 0 or 1
			// blank line: 1
			// "Max Iterations" label: 1
			yOffset := 4 // label + input + blank + label
			if c.sessionError != "" {
				yOffset++ // error line
			}
			cur.Y += yOffset
		}
	}
	return cur
}

// PreferredHeight returns the preferred height for this step's content.
// This allows the modal to size dynamically based on content.
func (c *ConfigStep) PreferredHeight() int {
	// Fixed content lines:
	// - "Session Name" label: 1
	// - session input: 1
	// - potential error line: 1 (reserve even if not showing)
	// - blank line: 1
	// - "Max Iterations" label: 1
	// - iterations input: 1
	// - potential error line: 1 (reserve even if not showing)
	// - blank line: 1
	// - hint bar: 1
	// Total: 9 lines
	return 9
}

// ConfigCompleteMsg is sent when the config is complete and valid.
type ConfigCompleteMsg struct{}

// TabExitForwardMsg is sent when Tab is pressed on the last input.
// Parent should move focus to buttons.
type TabExitForwardMsg struct{}

// TabExitBackwardMsg is sent when Shift+Tab is pressed on the first input.
// Parent should move focus to buttons (from end).
type TabExitBackwardMsg struct{}
