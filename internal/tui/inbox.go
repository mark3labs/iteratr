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

// InboxPanel displays unread messages and provides an input field for sending.
type InboxPanel struct {
	viewport     viewport.Model
	state        *session.State
	width        int
	height       int
	inputValue   string
	inputFocused bool
	cursorPos    int
	focused      bool
}

// NewInboxPanel creates a new InboxPanel component.
func NewInboxPanel() *InboxPanel {
	vp := viewport.New()
	return &InboxPanel{
		viewport: vp,
	}
}

// Update handles messages for the inbox panel.
func (i *InboxPanel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		k := msg.String()

		// Handle focus toggle
		if k == "i" && !i.inputFocused {
			i.inputFocused = true
			i.cursorPos = len(i.inputValue)
			return nil
		}

		if k == "esc" && i.inputFocused {
			i.inputFocused = false
			return nil
		}

		// Only handle input when input field is focused
		if i.inputFocused {
			switch k {
			case "enter":
				// Send message
				if i.inputValue != "" {
					return i.sendMessage()
				}
			case "backspace":
				if i.cursorPos > 0 && len(i.inputValue) > 0 {
					// Remove character before cursor
					i.inputValue = i.inputValue[:i.cursorPos-1] + i.inputValue[i.cursorPos:]
					i.cursorPos--
				}
			case "left":
				if i.cursorPos > 0 {
					i.cursorPos--
				}
			case "right":
				if i.cursorPos < len(i.inputValue) {
					i.cursorPos++
				}
			case "home":
				i.cursorPos = 0
			case "end":
				i.cursorPos = len(i.inputValue)
			case "ctrl+u":
				// Clear line
				i.inputValue = ""
				i.cursorPos = 0
			default:
				// Insert regular characters (single printable characters)
				if len(k) == 1 && k[0] >= 32 && k[0] <= 126 {
					// Insert at cursor position
					i.inputValue = i.inputValue[:i.cursorPos] + k + i.inputValue[i.cursorPos:]
					i.cursorPos++
				}
			}
			return nil
		}

		// When input not focused, delegate to viewport for scrolling
		var cmd tea.Cmd
		i.viewport, cmd = i.viewport.Update(msg)
		return cmd
	}

	// Delegate other messages to viewport
	var cmd tea.Cmd
	i.viewport, cmd = i.viewport.Update(msg)
	return cmd
}

// Draw renders the inbox panel to the screen buffer.
func (i *InboxPanel) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// Draw panel border with title
	inner := DrawPanel(scr, area, "Inbox", i.focused)

	// Reserve space for input field (separator + prompt + input + help = ~6 lines)
	inputHeight := 6
	messagesHeight := inner.Dy() - inputHeight
	if messagesHeight < 1 {
		messagesHeight = 1
	}

	// Split inner area into messages viewport and input field
	messagesArea := uv.Rectangle{
		Min: inner.Min,
		Max: uv.Position{X: inner.Max.X, Y: inner.Min.Y + messagesHeight},
	}
	inputArea := uv.Rectangle{
		Min: uv.Position{X: inner.Min.X, Y: inner.Min.Y + messagesHeight},
		Max: inner.Max,
	}

	// Draw viewport content (messages)
	content := i.viewport.View()
	DrawText(scr, messagesArea, content)

	// Draw scroll indicator if content overflows
	if i.viewport.TotalLineCount() > i.viewport.Height() {
		percent := i.viewport.ScrollPercent()
		DrawScrollIndicator(scr, area, percent)
	}

	// Draw input field
	i.drawInputField(scr, inputArea)

	// Return cursor position if input is focused
	if i.inputFocused {
		// Calculate cursor position: prompt + cursor offset
		promptWidth := len("Send message: ")
		cursorX := inputArea.Min.X + promptWidth + i.cursorPos
		cursorY := inputArea.Min.Y + 2 // After separator line
		return &tea.Cursor{
			Position: tea.Position{X: cursorX, Y: cursorY},
		}
	}

	return nil
}

// drawInputField renders the input field section.
func (i *InboxPanel) drawInputField(scr uv.Screen, area uv.Rectangle) {
	if area.Dy() < 4 {
		return // Not enough space
	}

	var y int = area.Min.Y

	// Draw separator line
	separatorArea := uv.Rectangle{
		Min: uv.Position{X: area.Min.X, Y: y},
		Max: uv.Position{X: area.Max.X, Y: y + 1},
	}
	separator := strings.Repeat("─", area.Dx())
	DrawText(scr, separatorArea, separator)
	y++

	// Skip a line
	y++

	// Draw prompt + input value
	promptArea := uv.Rectangle{
		Min: uv.Position{X: area.Min.X, Y: y},
		Max: uv.Position{X: area.Max.X, Y: y + 1},
	}
	prompt := styleInputPrompt.Render("Send message: ")

	// Build input text with cursor if focused
	inputText := i.inputValue
	if i.inputFocused && i.cursorPos <= len(inputText) {
		if i.cursorPos == len(inputText) {
			inputText += "▌"
		} else {
			inputText = inputText[:i.cursorPos] + "▌" + inputText[i.cursorPos:]
		}
	}

	inputStyled := styleInputField.Render(inputText)
	line := prompt + inputStyled
	DrawText(scr, promptArea, line)
	y++

	// Draw help text
	helpArea := uv.Rectangle{
		Min: uv.Position{X: area.Min.X, Y: y},
		Max: uv.Position{X: area.Max.X, Y: y + 1},
	}
	var helpText string
	if i.inputFocused {
		helpText = styleDim.Render("Enter=send | Ctrl+U=clear | Esc=unfocus")
	} else {
		helpText = styleDim.Render("Press 'i' to focus input field")
	}
	DrawText(scr, helpArea, helpText)
}

// sendMessage sends the current input value as a message.
func (i *InboxPanel) sendMessage() tea.Cmd {
	content := i.inputValue
	i.inputValue = ""
	i.cursorPos = 0

	// TODO: Actually send message via session store
	// For now, this is a placeholder
	return func() tea.Msg {
		return SendMessageMsg{Content: content}
	}
}

// SendMessageMsg is sent when a message should be sent.
type SendMessageMsg struct {
	Content string
}

// Render returns the inbox panel view as a string.
func (i *InboxPanel) Render() string {
	if i.state == nil {
		return styleEmptyState.Render("No session state loaded")
	}

	var content strings.Builder

	// Title
	content.WriteString(stylePanelTitle.Render("Inbox"))
	content.WriteString("\n\n")

	// Viewport content (messages)
	content.WriteString(i.viewport.View())

	// Add input field at the bottom
	content.WriteString("\n")
	content.WriteString(i.renderInputField())

	// Render in panel style
	return stylePanel.Width(i.width - 4).Height(i.height - 4).Render(content.String())
}

// renderMessage renders a single inbox message.
func (i *InboxPanel) renderMessage(msg *session.Message) string {
	// Message ID (first 8 chars)
	idPrefix := msg.ID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}

	// Format timestamp as "2006-01-02 15:04:05"
	timestamp := msg.CreatedAt.Format("2006-01-02 15:04:05")

	// Format: [id] timestamp: content
	var parts []string
	parts = append(parts, styleMessageUnread.Render(fmt.Sprintf("[%s]", idPrefix)))
	parts = append(parts, styleMessageTimestamp.Render(timestamp+":"))
	parts = append(parts, styleMessageUnread.Render(msg.Content))

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// renderInputField renders the message input field.
func (i *InboxPanel) renderInputField() string {
	var content strings.Builder

	// Separator
	content.WriteString(strings.Repeat("─", i.width-8))
	content.WriteString("\n\n")

	// Prompt
	prompt := styleInputPrompt.Render("Send message: ")
	content.WriteString(prompt)

	// Input value with cursor
	inputText := i.inputValue
	if i.inputFocused && i.cursorPos <= len(inputText) {
		// Insert cursor character at cursor position
		if i.cursorPos == len(inputText) {
			inputText += "▌"
		} else {
			inputText = inputText[:i.cursorPos] + "▌" + inputText[i.cursorPos:]
		}
	}

	content.WriteString(styleInputField.Render(inputText))
	content.WriteString("\n")

	// Help text
	if i.inputFocused {
		help := styleDim.Render("Enter=send | Ctrl+U=clear | Esc=unfocus")
		content.WriteString(help)
	} else {
		help := styleDim.Render("Press 'i' to focus input field")
		content.WriteString(help)
	}

	return content.String()
}

// SetSize updates the inbox panel dimensions.
func (i *InboxPanel) SetSize(width, height int) {
	i.width = width
	i.height = height

	// Account for border (2), title (2), input field (~6 lines)
	viewportHeight := height - 12
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	i.viewport.SetWidth(width - 4)
	i.viewport.SetHeight(viewportHeight)
	i.updateContent()
}

// SetState updates the inbox panel with new session state.
func (i *InboxPanel) SetState(state *session.State) {
	i.state = state
	i.updateContent()
}

// SetFocus sets the focus state of the inbox panel.
func (i *InboxPanel) SetFocus(focused bool) {
	i.focused = focused
}

// IsFocused returns whether the inbox panel is focused.
func (i *InboxPanel) IsFocused() bool {
	return i.focused
}

// UpdateSize updates the inbox panel dimensions (legacy compatibility).
func (i *InboxPanel) UpdateSize(width, height int) tea.Cmd {
	i.SetSize(width, height)
	return nil
}

// UpdateState updates the inbox panel with new session state (legacy compatibility).
func (i *InboxPanel) UpdateState(state *session.State) tea.Cmd {
	i.SetState(state)
	return nil
}

// updateContent rebuilds the viewport content from the current state.
func (i *InboxPanel) updateContent() {
	if i.state == nil {
		i.viewport.SetContent("")
		return
	}

	var content strings.Builder

	// Filter unread messages
	var unreadMessages []*session.Message
	for _, msg := range i.state.Inbox {
		if !msg.Read {
			unreadMessages = append(unreadMessages, msg)
		}
	}

	// Display unread messages
	if len(unreadMessages) == 0 {
		content.WriteString(styleEmptyState.Render("No unread messages"))
	} else {
		// Show count
		content.WriteString(styleBadgeInfo.Render(fmt.Sprintf("%d unread", len(unreadMessages))))
		content.WriteString("\n\n")

		// Render each message
		for _, msg := range unreadMessages {
			content.WriteString(i.renderMessage(msg))
			content.WriteString("\n")
		}
	}

	i.viewport.SetContent(content.String())
}

// Compile-time interface checks
var _ FocusableComponent = (*InboxPanel)(nil)
