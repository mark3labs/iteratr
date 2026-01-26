package wizard

import (
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	"github.com/charmbracelet/lipgloss"
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
}

// NewConfigStep creates a new config step with smart defaults.
func NewConfigStep(specPath string) *ConfigStep {
	// Initialize session name input
	sessionInput := textinput.New()
	sessionInput.Placeholder = "Enter session name..."
	sessionInput.Prompt = ""

	// Configure styles for textinput (using lipgloss v2)
	sessionStyles := textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipglossv2.NewStyle().Foreground(lipglossv2.Color("#cdd6f4")),
			Placeholder: lipglossv2.NewStyle().Foreground(lipglossv2.Color("#a6adc8")),
			Prompt:      lipglossv2.NewStyle().Foreground(lipglossv2.Color("#b4befe")),
		},
		Blurred: textinput.StyleState{
			Text:        lipglossv2.NewStyle().Foreground(lipglossv2.Color("#a6adc8")),
			Placeholder: lipglossv2.NewStyle().Foreground(lipglossv2.Color("#a6adc8")),
			Prompt:      lipglossv2.NewStyle().Foreground(lipglossv2.Color("#6c7086")),
		},
		Cursor: textinput.CursorStyle{
			Color: lipglossv2.Color("#cba6f7"),
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
	return c.sessionInput.Focus()
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

	// Handle keyboard input
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "tab":
			// Cycle focus forward
			c.cycleFocusForward()
			return nil

		case "shift+tab":
			// Cycle focus backward
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

// ConfigCompleteMsg is sent when the config is complete and valid.
type ConfigCompleteMsg struct{}
