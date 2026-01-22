package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
)

// MessageType indicates the type of agent message.
type MessageType int

const (
	MessageTypeText MessageType = iota
	MessageTypeTool
	MessageTypeDivider
)

// AgentMessage represents a single message from the agent.
type AgentMessage struct {
	Type      MessageType
	Content   string
	Tool      string // Tool name for tool messages
	Iteration int    // Iteration number for dividers
}

// AgentOutput displays streaming agent output with auto-scroll.
type AgentOutput struct {
	viewport   viewport.Model
	messages   []AgentMessage
	width      int
	height     int
	autoScroll bool // Whether to auto-scroll to bottom on new content
	ready      bool // Whether viewport is initialized
}

// Compile-time interface checks
var _ Drawable = (*AgentOutput)(nil)
var _ Updateable = (*AgentOutput)(nil)
var _ Component = (*AgentOutput)(nil)

// NewAgentOutput creates a new AgentOutput component.
func NewAgentOutput() *AgentOutput {
	return &AgentOutput{
		messages:   make([]AgentMessage, 0),
		autoScroll: true,
	}
}

// Init initializes the agent output component.
func (a *AgentOutput) Init() tea.Cmd {
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
		if !a.viewport.AtBottom() {
			a.autoScroll = false
		} else {
			a.autoScroll = true
		}
	}

	return cmd
}

// Render returns the agent output view as a string.
func (a *AgentOutput) Render() string {
	if !a.ready {
		return styleDim.Render("Waiting for agent output...")
	}
	return a.viewport.View()
}

// Draw renders the agent output to a screen buffer.
func (a *AgentOutput) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	if !a.ready {
		// Show waiting message
		waitMsg := styleDim.Render("Waiting for agent output...")
		uv.NewStyledString(waitMsg).Draw(scr, area)
		return nil
	}

	// Render viewport content
	content := a.viewport.View()
	uv.NewStyledString(content).Draw(scr, area)

	// Draw scroll indicator if there's overflow
	if a.viewport.TotalLineCount() > a.viewport.Height() {
		pct := a.viewport.ScrollPercent()
		indicator := fmt.Sprintf(" %d%% ", int(pct*100))

		// Position indicator at bottom-right of area
		indicatorArea := uv.Rect(
			area.Max.X-len(indicator),
			area.Max.Y-1,
			len(indicator),
			1,
		)

		styledIndicator := styleScrollIndicator.Render(indicator)
		uv.NewStyledString(styledIndicator).Draw(scr, indicatorArea)
	}

	return nil
}

// UpdateSize updates the agent output dimensions.
func (a *AgentOutput) UpdateSize(width, height int) tea.Cmd {
	a.width = width
	a.height = height

	if !a.ready {
		a.viewport = viewport.New(
			viewport.WithWidth(width),
			viewport.WithHeight(height),
		)
		a.viewport.MouseWheelEnabled = true
		a.viewport.MouseWheelDelta = 3
		a.ready = true
	} else {
		a.viewport.SetWidth(width)
		a.viewport.SetHeight(height)
	}

	a.refreshContent()
	return nil
}

// AppendText adds a text message to the output.
func (a *AgentOutput) AppendText(content string) tea.Cmd {
	// If last message is text, append to it
	if len(a.messages) > 0 && a.messages[len(a.messages)-1].Type == MessageTypeText {
		a.messages[len(a.messages)-1].Content += content
	} else {
		a.messages = append(a.messages, AgentMessage{
			Type:    MessageTypeText,
			Content: content,
		})
	}
	a.refreshContent()
	return nil
}

// AppendTool adds a tool use message to the output.
func (a *AgentOutput) AppendTool(tool string, input map[string]any) tea.Cmd {
	// Format tool info
	content := formatToolInput(input)
	a.messages = append(a.messages, AgentMessage{
		Type:    MessageTypeTool,
		Content: content,
		Tool:    tool,
	})
	a.refreshContent()
	return nil
}

// AddIterationDivider adds a horizontal divider for a new iteration.
func (a *AgentOutput) AddIterationDivider(iteration int) tea.Cmd {
	a.messages = append(a.messages, AgentMessage{
		Type:      MessageTypeDivider,
		Iteration: iteration,
	})
	a.refreshContent()
	return nil
}

// formatToolInput formats the tool input for display.
func formatToolInput(input map[string]any) string {
	if input == nil {
		return ""
	}
	var parts []string
	for k, v := range input {
		parts = append(parts, fmt.Sprintf("%s: %v", k, v))
	}
	return strings.Join(parts, ", ")
}

// refreshContent rebuilds the viewport content from messages.
func (a *AgentOutput) refreshContent() {
	if !a.ready {
		return
	}

	var rendered strings.Builder
	contentWidth := a.width - 4 // Account for border and padding

	for _, msg := range a.messages {
		block := a.renderMessage(msg, contentWidth)
		rendered.WriteString(block)
		rendered.WriteString("\n")
	}

	a.viewport.SetContent(rendered.String())

	if a.autoScroll {
		a.viewport.GotoBottom()
	}
}

// renderMessage renders a single message with appropriate styling.
func (a *AgentOutput) renderMessage(msg AgentMessage, width int) string {
	switch msg.Type {
	case MessageTypeTool:
		return a.renderToolMessage(msg, width)
	case MessageTypeDivider:
		return a.renderDivider(msg, width)
	default:
		return a.renderTextMessage(msg, width)
	}
}

// renderTextMessage renders a text message with left border.
func (a *AgentOutput) renderTextMessage(msg AgentMessage, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(colorPrimary).
		PaddingLeft(1).
		MarginBottom(1).
		Width(width)

	// Word wrap the content
	wrapped := wrapText(msg.Content, width-3)
	return style.Render(wrapped)
}

// renderToolMessage renders a tool use message with different styling.
func (a *AgentOutput) renderToolMessage(msg AgentMessage, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(colorSecondary).
		PaddingLeft(1).
		MarginBottom(1).
		Width(width)

	// Tool header
	header := lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true).
		Render(fmt.Sprintf(" %s", msg.Tool))

	content := header
	if msg.Content != "" {
		content += "\n" + styleDim.Render(msg.Content)
	}

	return style.Render(content)
}

// renderDivider renders an iteration divider with a horizontal rule.
func (a *AgentOutput) renderDivider(msg AgentMessage, width int) string {
	// Create the iteration label
	label := fmt.Sprintf(" Iteration #%d ", msg.Iteration)
	labelWidth := len(label)

	// Calculate line widths on each side
	lineWidth := (width - labelWidth) / 2
	if lineWidth < 3 {
		lineWidth = 3
	}

	// Build the horizontal rule with centered label
	line := strings.Repeat("─", lineWidth)
	divider := line + label + line

	// Style the divider
	style := lipgloss.NewStyle().
		Foreground(colorMuted).
		Bold(true).
		MarginTop(1).
		MarginBottom(1)

	return style.Render(divider)
}

// wrapText wraps text to the given width.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Wrap long lines
		for len(line) > width {
			// Find last space before width
			breakPoint := width
			for j := width; j > 0; j-- {
				if line[j] == ' ' {
					breakPoint = j
					break
				}
			}
			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakPoint:], " ")
		}
		result.WriteString(line)
	}

	return result.String()
}

// Clear resets the agent output content.
func (a *AgentOutput) Clear() tea.Cmd {
	a.messages = make([]AgentMessage, 0)
	if a.ready {
		a.viewport.SetContent("")
		a.viewport.GotoTop()
	}
	a.autoScroll = true
	return nil
}

// Append adds content to the agent output stream (legacy - calls AppendText).
func (a *AgentOutput) Append(content string) tea.Cmd {
	return a.AppendText(content)
}
