package tui

import (
	tea "charm.land/bubbletea/v2"
)

// AgentOutput displays streaming agent output with auto-scroll.
type AgentOutput struct {
	content string
	width   int
	height  int
}

// NewAgentOutput creates a new AgentOutput component.
func NewAgentOutput() *AgentOutput {
	return &AgentOutput{}
}

// Init initializes the agent output component.
func (a *AgentOutput) Init() tea.Cmd {
	// TODO: Implement initialization if needed
	return nil
}

// Update handles messages for the agent output.
func (a *AgentOutput) Update(msg tea.Msg) tea.Cmd {
	// TODO: Implement agent output updates (scrolling)
	return nil
}

// Render returns the agent output view as a string.
func (a *AgentOutput) Render() string {
	// TODO: Implement agent output rendering with markdown
	return a.content
}

// UpdateSize updates the agent output dimensions.
func (a *AgentOutput) UpdateSize(width, height int) tea.Cmd {
	a.width = width
	a.height = height
	return nil
}

// Append adds content to the agent output stream.
func (a *AgentOutput) Append(content string) tea.Cmd {
	a.content += content
	return nil
}
