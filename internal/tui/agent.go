package tui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// AgentOutput displays streaming agent output with auto-scroll.
type AgentOutput struct {
	viewport   viewport.Model
	content    strings.Builder
	width      int
	height     int
	autoScroll bool // Whether to auto-scroll to bottom on new content
	ready      bool // Whether viewport is initialized
}

// NewAgentOutput creates a new AgentOutput component.
func NewAgentOutput() *AgentOutput {
	return &AgentOutput{
		autoScroll: true, // Start with auto-scroll enabled
	}
}

// Init initializes the agent output component.
func (a *AgentOutput) Init() tea.Cmd {
	// Viewport will be initialized when we receive size
	return nil
}

// Update handles messages for the agent output.
func (a *AgentOutput) Update(msg tea.Msg) tea.Cmd {
	if !a.ready {
		return nil
	}

	var cmd tea.Cmd
	a.viewport, cmd = a.viewport.Update(msg)

	// Check if user manually scrolled - disable auto-scroll
	switch msg.(type) {
	case tea.KeyPressMsg, tea.MouseMsg:
		// User interaction detected - check if they scrolled away from bottom
		if !a.viewport.AtBottom() {
			a.autoScroll = false
		} else {
			// User scrolled back to bottom - re-enable auto-scroll
			a.autoScroll = true
		}
	}

	return cmd
}

// Render returns the agent output view as a string.
func (a *AgentOutput) Render() string {
	if !a.ready {
		return styleAgentOutput.Render("Waiting for agent output...")
	}
	return styleAgentOutput.Render(a.viewport.View())
}

// UpdateSize updates the agent output dimensions.
func (a *AgentOutput) UpdateSize(width, height int) tea.Cmd {
	a.width = width
	a.height = height

	// Initialize or update viewport
	if !a.ready {
		a.viewport = viewport.New(
			viewport.WithWidth(width),
			viewport.WithHeight(height),
		)
		a.viewport.MouseWheelEnabled = true
		a.viewport.MouseWheelDelta = 3
		a.viewport.SetContent(a.content.String())
		a.ready = true
	} else {
		a.viewport.SetWidth(width)
		a.viewport.SetHeight(height)
	}

	return nil
}

// Append adds content to the agent output stream.
// This is called when AgentOutputMsg is received.
func (a *AgentOutput) Append(content string) tea.Cmd {
	// Append to content buffer
	a.content.WriteString(content)

	// Update viewport content
	if a.ready {
		a.viewport.SetContent(a.content.String())

		// Auto-scroll to bottom if enabled
		if a.autoScroll {
			a.viewport.GotoBottom()
		}
	}

	return nil
}

// Clear resets the agent output content.
func (a *AgentOutput) Clear() tea.Cmd {
	a.content.Reset()
	if a.ready {
		a.viewport.SetContent("")
		a.viewport.GotoTop()
	}
	a.autoScroll = true
	return nil
}
