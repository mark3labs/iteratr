package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// AgentOutput displays streaming agent output with auto-scroll.
type AgentOutput struct {
	scrollList        *ScrollList
	input             textinput.Model // User input field below viewport
	messages          []MessageItem
	toolIndex         map[string]int // toolCallId → message index
	width             int
	height            int
	ready             bool // Whether scrollList is initialized
	spinner           *GradientSpinner
	isStreaming       bool
	queueDepth        int          // Number of messages waiting in orchestrator queue
	focusedIndex      int          // Index of the message that has keyboard focus (-1 = no focus)
	viewportArea      uv.Rectangle // Screen area where viewport is drawn (for mouse hit detection)
	inputArea         uv.Rectangle // Screen area where input field is drawn (for mouse hit detection)
	messageLineStarts []int        // Start line index in content for each message
}

// Compile-time interface checks
var _ Drawable = (*AgentOutput)(nil)
var _ Updateable = (*AgentOutput)(nil)
var _ Component = (*AgentOutput)(nil)

// NewAgentOutput creates a new AgentOutput component.
func NewAgentOutput() *AgentOutput {
	// Initialize textinput with dark theme styling
	input := textinput.New()
	input.Placeholder = "Send a message..."
	input.Prompt = "> "

	// Configure styles for textinput (using lipgloss v2)
	styles := textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipglossv2.NewStyle().Foreground(lipglossv2.Color(string(colorText))),     // Base text color when typing
			Placeholder: lipglossv2.NewStyle().Foreground(lipglossv2.Color(string(colorTextDim))),  // Placeholder text (dim/subtle)
			Prompt:      lipglossv2.NewStyle().Foreground(lipglossv2.Color(string(colorTertiary))), // Prompt color (lavender accent)
		},
		Blurred: textinput.StyleState{
			Text:        lipglossv2.NewStyle().Foreground(lipglossv2.Color(string(colorSubtext0))), // Dimmer when unfocused
			Placeholder: lipglossv2.NewStyle().Foreground(lipglossv2.Color(string(colorSubtext0))), // Dim placeholder
			Prompt:      lipglossv2.NewStyle().Foreground(lipglossv2.Color(string(colorMuted))),    // Muted prompt
		},
		Cursor: textinput.CursorStyle{
			Color: lipglossv2.Color(string(colorPrimary)), // Cursor color (mauve accent)
			Shape: tea.CursorBar,                          // Bar shape
			Blink: true,                                   // Enable blinking
		},
	}
	input.SetStyles(styles)

	// Disable virtual cursor (cursor handled by Dashboard Draw)
	input.SetVirtualCursor(false)

	// Set a safe default width to avoid panics if Draw is called before UpdateSize
	input.SetWidth(40)

	return &AgentOutput{
		messages:     make([]MessageItem, 0),
		toolIndex:    make(map[string]int),
		focusedIndex: -1, // No message focused initially
		input:        input,
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

	// If input is focused, forward messages to the input field
	if a.input.Focused() {
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(msg)
		return cmd
	}

	// Handle gradient spinner animation
	if a.spinner != nil {
		if spinnerCmd := a.spinner.Update(msg); spinnerCmd != nil {
			// Refresh content to show updated spinner frame
			a.refreshContent()
			return spinnerCmd
		}
	}

	// Handle keyboard input for expand/collapse on focused message
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "up":
			// Move focus to previous expandable message
			a.moveFocusPrevious()
			return nil
		case "down":
			// Move focus to next expandable message
			a.moveFocusNext()
			return nil
		case "k":
			// Scroll up by one line (vim-style)
			if a.scrollList != nil {
				a.scrollList.ScrollBy(-1)
				a.scrollList.SetAutoScroll(false)
			}
			return nil
		case "j":
			// Scroll down by one line (vim-style)
			if a.scrollList != nil {
				a.scrollList.ScrollBy(1)
				if a.scrollList.AtBottom() {
					a.scrollList.SetAutoScroll(true)
				}
			}
			return nil
		case "space", "enter":
			// Toggle expansion on focused message if it's expandable
			if a.focusedIndex >= 0 && a.focusedIndex < len(a.messages) {
				focusedMsg := a.messages[a.focusedIndex]
				if expandable, ok := focusedMsg.(Expandable); ok {
					expandable.ToggleExpanded()
					a.refreshContent()
					return nil
				}
			}
		}
	}

	// Forward to scroll list for scroll handling (pgup/pgdn/home/end)
	var cmd tea.Cmd
	if a.scrollList != nil {
		cmd = a.scrollList.Update(msg)
	}

	return cmd
}

// Render returns the agent output view as a string.
func (a *AgentOutput) Render() string {
	if !a.ready {
		return styleDim.Render("Waiting for agent output...")
	}
	if a.scrollList == nil {
		return ""
	}
	return a.scrollList.View()
}

// Draw renders the agent output to a screen buffer.
func (a *AgentOutput) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	if area.Dx() < 2 || area.Dy() < 2 {
		return nil
	}

	if !a.ready {
		// Show waiting message
		waitMsg := styleDim.Render("Waiting for agent output...")
		uv.NewStyledString(waitMsg).Draw(scr, area)
		return nil
	}

	// Split layout vertically: viewport gets remaining space, input area gets 5 lines at bottom
	// (top margin + separator + input + help text + bottom margin)
	viewportHeight := area.Dy() - 5
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	viewportArea, inputArea := uv.SplitVertical(area, uv.Fixed(viewportHeight))

	// Store input area for mouse hit detection
	a.inputArea = inputArea

	// Render scroll list content with 1-char left margin
	var content string
	if a.scrollList != nil {
		content = a.scrollList.View()
	}
	contentArea := uv.Rect(viewportArea.Min.X+1, viewportArea.Min.Y, viewportArea.Dx()-1, viewportArea.Dy())
	a.viewportArea = contentArea
	uv.NewStyledString(content).Draw(scr, contentArea)

	// Draw scroll indicator if there's overflow
	if a.scrollList != nil && a.scrollList.TotalLineCount() > a.scrollList.height {
		pct := a.scrollList.ScrollPercent()
		indicator := fmt.Sprintf(" %d%% ", int(pct*100))

		// Position indicator at bottom-right of viewport area
		indicatorArea := uv.Rect(
			viewportArea.Max.X-len(indicator),
			viewportArea.Max.Y-1,
			len(indicator),
			1,
		)

		styledIndicator := styleScrollIndicator.Render(indicator)
		uv.NewStyledString(styledIndicator).Draw(scr, indicatorArea)
	}

	// Draw separator line and input only if input area has valid dimensions
	if inputArea.Dx() > 0 && inputArea.Dy() > 1 {
		// Left padding to match status bar indentation
		leftPad := 2

		// Top margin: skip first row
		separatorY := inputArea.Min.Y + 1
		separatorLine := strings.Repeat(" ", leftPad) + strings.Repeat("─", inputArea.Dx()-leftPad)
		separatorArea := uv.Rect(inputArea.Min.X, separatorY, inputArea.Dx(), 1)
		uv.NewStyledString(styleDim.Render(separatorLine)).Draw(scr, separatorArea)

		// Draw input field below separator
		inputView := a.input.View()
		inputY := separatorY + 1

		// Calculate queue indicator if messages are queued
		var queueIndicator string
		var queueIndicatorWidth int
		if a.queueDepth > 0 {
			queueIndicator = styleDim.Render(fmt.Sprintf("(%d queued)", a.queueDepth))
			// Strip ANSI codes to measure actual width (lipgloss adds invisible codes)
			queueIndicatorWidth = len(fmt.Sprintf("(%d queued)", a.queueDepth))
		}

		// Adjust input area to make room for queue indicator on the right
		inputWidth := inputArea.Dx() - leftPad
		if queueIndicatorWidth > 0 {
			// Leave space for indicator plus 2 spaces padding
			inputWidth = inputWidth - queueIndicatorWidth - 2
			if inputWidth < 10 {
				inputWidth = 10 // Minimum width for input
			}
		}

		inputContentArea := uv.Rect(inputArea.Min.X+leftPad, inputY, inputWidth, 1)
		uv.NewStyledString(inputView).Draw(scr, inputContentArea)

		// Draw queue indicator on the right side if present
		if queueIndicatorWidth > 0 {
			indicatorX := inputArea.Min.X + inputArea.Dx() - queueIndicatorWidth - 1
			indicatorArea := uv.Rect(indicatorX, inputY, queueIndicatorWidth+1, 1)
			uv.NewStyledString(queueIndicator).Draw(scr, indicatorArea)
		}

		// Draw help text below input
		if inputArea.Dy() >= 4 {
			var helpText string
			if a.input.Focused() {
				helpText = styleDim.Render("Enter to send · Esc to cancel")
			} else {
				helpText = styleDim.Render("Press i to type a message")
			}
			helpY := inputY + 1
			helpArea := uv.Rect(inputArea.Min.X+leftPad, helpY, inputArea.Dx()-leftPad, 1)
			uv.NewStyledString(helpText).Draw(scr, helpArea)
		}
		// Bottom margin: last row left empty

		// Return cursor position if input is focused
		if a.input.Focused() {
			return a.input.Cursor()
		}
	}

	return nil
}

// UpdateSize updates the agent output dimensions.
func (a *AgentOutput) UpdateSize(width, height int) tea.Cmd {
	a.width = width
	a.height = height

	// Split height: scrollList gets (height - 3), input area gets 3 lines
	scrollListHeight := height - 3
	if scrollListHeight < 1 {
		scrollListHeight = 1
	}

	// Content width accounts for border, padding, and 1-char left margin
	contentWidth := width - 5

	if a.scrollList == nil {
		a.scrollList = NewScrollList(contentWidth, scrollListHeight)
		a.ready = true
	} else {
		a.scrollList.SetWidth(contentWidth)
		a.scrollList.SetHeight(scrollListHeight)
	}

	// Set input width (accounting for borders and padding)
	inputWidth := width - 4
	if inputWidth < 1 {
		inputWidth = 1
	}
	a.input.SetWidth(inputWidth)

	a.refreshContent()
	return nil
}

// AppendText adds a text message to the output.
func (a *AgentOutput) AppendText(content string) tea.Cmd {
	// Start spinner on first streaming content
	var spinnerCmd tea.Cmd
	if !a.isStreaming && content == "" {
		a.isStreaming = true
		// Create spinner with label "Generating..."
		spinner := NewGradientSpinner("#cba6f7", "#89b4fa", "Generating...")
		a.spinner = &spinner
		spinnerCmd = a.spinner.Tick()
	}

	// Stop spinner when actual content arrives
	if a.isStreaming && content != "" {
		a.isStreaming = false
		a.spinner = nil
	}

	// If last message is a TextMessageItem, append to it
	if len(a.messages) > 0 {
		if textMsg, ok := a.messages[len(a.messages)-1].(*TextMessageItem); ok {
			textMsg.content += content
			// Invalidate cache - ScrollList will re-render on next View() call
			textMsg.cachedWidth = 0
			// Only adjust scroll position if needed, no full refresh
			if a.ready && a.scrollList != nil && a.scrollList.autoScroll {
				a.scrollList.GotoBottom()
			}
			return spinnerCmd
		}
	}

	// Create new TextMessageItem
	newMsg := &TextMessageItem{
		id:      fmt.Sprintf("text-%d", len(a.messages)),
		content: content,
	}
	a.messages = append(a.messages, newMsg)
	a.refreshContent()
	return spinnerCmd
}

// AppendToolCall handles tool lifecycle events.
// If toolCallId not in toolIndex: append new message, store index.
// If toolCallId exists: update message in-place (status, input, output).
func (a *AgentOutput) AppendToolCall(msg AgentToolCallMsg) tea.Cmd {
	idx, exists := a.toolIndex[msg.ToolCallID]
	if !exists {
		// New tool call - create ToolMessageItem
		// Map status strings to ToolStatus enum
		status := mapToolStatus(msg.Status)

		newMsg := &ToolMessageItem{
			id:       msg.ToolCallID,
			toolName: msg.Title,
			kind:     msg.Kind,
			status:   status,
			input:    msg.Input,
			output:   msg.Output,
			fileDiff: msg.FileDiff,
			maxLines: 10,
		}
		a.messages = append(a.messages, newMsg)
		a.toolIndex[msg.ToolCallID] = len(a.messages) - 1
		a.refreshContent()
	} else {
		// Update existing tool call in-place
		if toolMsg, ok := a.messages[idx].(*ToolMessageItem); ok {
			toolMsg.status = mapToolStatus(msg.Status)
			if msg.Kind != "" {
				toolMsg.kind = msg.Kind
			}
			if len(msg.Input) > 0 {
				toolMsg.input = msg.Input
			}
			if msg.Output != "" {
				toolMsg.output = msg.Output
			}
			if msg.FileDiff != nil {
				toolMsg.fileDiff = msg.FileDiff
			}
			// Invalidate cache - ScrollList will re-render on next View() call
			toolMsg.cachedWidth = 0
			// Only adjust scroll position if needed, no full refresh
			if a.ready && a.scrollList != nil && a.scrollList.autoScroll {
				a.scrollList.GotoBottom()
			}
		}
	}
	return nil
}

// mapToolStatus converts status strings to ToolStatus enum.
func mapToolStatus(status string) ToolStatus {
	switch status {
	case "pending":
		return ToolStatusPending
	case "in_progress":
		return ToolStatusRunning
	case "completed":
		return ToolStatusSuccess
	case "error":
		return ToolStatusError
	case "canceled", "cancelled":
		return ToolStatusCanceled
	default:
		return ToolStatusPending
	}
}

// AddIterationDivider adds a horizontal divider for a new iteration.
func (a *AgentOutput) AddIterationDivider(iteration int) tea.Cmd {
	newMsg := &DividerMessageItem{
		id:        fmt.Sprintf("divider-%d", iteration),
		iteration: iteration,
	}
	a.messages = append(a.messages, newMsg)
	a.refreshContent()
	return nil
}

// HandleClick processes a mouse click at screen coordinates (x, y).
// Returns true if the click toggled an expandable message.
func (a *AgentOutput) HandleClick(x, y int) bool {
	if !a.ready || len(a.messageLineStarts) == 0 {
		return false
	}

	// Check if click is within the viewport area
	if x < a.viewportArea.Min.X || x >= a.viewportArea.Max.X ||
		y < a.viewportArea.Min.Y || y >= a.viewportArea.Max.Y {
		return false
	}

	// Translate screen Y to content line (accounting for scroll offset)
	contentLine := (y - a.viewportArea.Min.Y) + a.scrollList.currentOffsetInLines()

	// Find which message this line belongs to
	msgIdx := -1
	for i := len(a.messageLineStarts) - 1; i >= 0; i-- {
		if contentLine >= a.messageLineStarts[i] {
			msgIdx = i
			break
		}
	}

	if msgIdx < 0 || msgIdx >= len(a.messages) {
		return false
	}

	// Toggle if expandable
	if expandable, ok := a.messages[msgIdx].(Expandable); ok {
		expandable.ToggleExpanded()
		a.refreshContent()
		return true
	}

	return false
}

// refreshContent updates the scroll list with current messages.
// ScrollList lazily renders only visible items on View() call, so this method
// only needs to update the items and optionally adjust scroll position.
func (a *AgentOutput) refreshContent() {
	if !a.ready || a.scrollList == nil {
		return
	}

	// Convert messages to ScrollItems
	// MessageItem already implements ScrollItem interface (ID, Render, Height)
	items := make([]ScrollItem, len(a.messages))
	for i, msg := range a.messages {
		items[i] = msg
	}

	a.scrollList.SetItems(items)

	// Compute messageLineStarts for click-to-expand hit detection.
	// Each entry is the cumulative line offset where that message starts.
	a.messageLineStarts = make([]int, len(items))
	offset := 0
	for i, item := range items {
		a.messageLineStarts[i] = offset
		h := item.Height()
		if h == 0 {
			item.Render(a.scrollList.width)
			h = item.Height()
		}
		offset += h
	}

	// Only adjust scroll position if auto-scroll is enabled
	// No full re-render needed - ScrollList renders lazily on next View() call
	if a.scrollList.autoScroll {
		a.scrollList.GotoBottom()
	}
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
	a.messages = make([]MessageItem, 0)
	a.toolIndex = make(map[string]int)
	if a.ready && a.scrollList != nil {
		a.scrollList.SetItems([]ScrollItem{})
		a.scrollList.GotoTop()
		a.scrollList.SetAutoScroll(true)
	}
	return nil
}

// AppendThinking adds thinking/reasoning content to the output.
// If last message is ThinkingMessageItem, appends to it; otherwise creates new one.
func (a *AgentOutput) AppendThinking(content string) tea.Cmd {
	// Start spinner on first streaming content
	var spinnerCmd tea.Cmd
	if !a.isStreaming && content == "" {
		a.isStreaming = true
		// Create spinner with label "Thinking..."
		spinner := NewGradientSpinner("#cba6f7", "#89b4fa", "Thinking...")
		a.spinner = &spinner
		spinnerCmd = a.spinner.Tick()
	}

	// Stop spinner when actual content arrives
	if a.isStreaming && content != "" {
		a.isStreaming = false
		a.spinner = nil
	}

	// If last message is a ThinkingMessageItem, append to it
	if len(a.messages) > 0 {
		if thinkingMsg, ok := a.messages[len(a.messages)-1].(*ThinkingMessageItem); ok {
			thinkingMsg.content += content
			// Invalidate cache - ScrollList will re-render on next View() call
			thinkingMsg.cachedWidth = 0
			// Only adjust scroll position if needed, no full refresh
			if a.ready && a.scrollList != nil && a.scrollList.autoScroll {
				a.scrollList.GotoBottom()
			}
			return spinnerCmd
		}
	}

	// Create new ThinkingMessageItem
	newMsg := &ThinkingMessageItem{
		id:        fmt.Sprintf("thinking-%d", len(a.messages)),
		content:   content,
		collapsed: true, // default true
	}
	a.messages = append(a.messages, newMsg)
	a.refreshContent()
	return spinnerCmd
}

// AppendFinish marks the iteration as finished and displays completion metadata.
// Sets finished=true on last ThinkingMessageItem (with duration), appends InfoMessageItem
// for model/provider/duration, and appends styled finish reason for errors/cancellations.
func (a *AgentOutput) AppendFinish(msg AgentFinishMsg) tea.Cmd {
	// Stop spinner when finish event received
	if a.isStreaming {
		a.isStreaming = false
		a.spinner = nil
	}

	// 1. Mark last ThinkingMessageItem as finished with duration
	if len(a.messages) > 0 {
		if thinkingMsg, ok := a.messages[len(a.messages)-1].(*ThinkingMessageItem); ok {
			thinkingMsg.finished = true
			thinkingMsg.duration = msg.Duration
			// Invalidate cache to trigger re-render with footer
			thinkingMsg.cachedWidth = 0
		}
	}

	// 1.5. If canceled, mark all pending/running tools as canceled
	if msg.Reason == "cancelled" {
		for _, message := range a.messages {
			if toolMsg, ok := message.(*ToolMessageItem); ok {
				if toolMsg.status == ToolStatusPending || toolMsg.status == ToolStatusRunning {
					toolMsg.status = ToolStatusCanceled
					// Invalidate cache
					toolMsg.cachedWidth = 0
				}
			}
		}
	}

	// 2. Append InfoMessageItem with model/provider/duration
	infoMsg := &InfoMessageItem{
		id:       fmt.Sprintf("info-%d", len(a.messages)),
		model:    msg.Model,
		provider: msg.Provider,
		duration: msg.Duration,
	}
	a.messages = append(a.messages, infoMsg)

	// 3. If error or canceled, append styled finish reason
	if msg.Error != "" {
		// Error finish
		errorText := styleFinishError.Render(fmt.Sprintf("Error: %s", msg.Error))
		errorItem := &TextMessageItem{
			id:      fmt.Sprintf("finish-error-%d", len(a.messages)),
			content: errorText,
		}
		a.messages = append(a.messages, errorItem)
	} else if msg.Reason == "cancelled" {
		// Canceled finish
		cancelText := styleFinishCanceled.Render("Iteration canceled")
		cancelItem := &TextMessageItem{
			id:      fmt.Sprintf("finish-cancel-%d", len(a.messages)),
			content: cancelText,
		}
		a.messages = append(a.messages, cancelItem)
	}

	a.refreshContent()
	return nil
}

// MarkToolError marks a tool call as failed with an error message.
// Finds the tool by ID, sets status to ToolStatusError, and updates output with error message.
func (a *AgentOutput) MarkToolError(toolCallID, errMsg string) tea.Cmd {
	idx, exists := a.toolIndex[toolCallID]
	if !exists {
		return nil
	}

	if toolMsg, ok := a.messages[idx].(*ToolMessageItem); ok {
		toolMsg.status = ToolStatusError
		toolMsg.output = errMsg
		// Invalidate cache - ScrollList will re-render on next View() call
		toolMsg.cachedWidth = 0
		// Only adjust scroll position if needed, no full refresh
		if a.ready && a.scrollList != nil && a.scrollList.autoScroll {
			a.scrollList.GotoBottom()
		}
	}

	return nil
}

// MarkToolCanceled marks a tool call as canceled.
// Finds the tool by ID and sets status to ToolStatusCanceled.
func (a *AgentOutput) MarkToolCanceled(toolCallID string) tea.Cmd {
	idx, exists := a.toolIndex[toolCallID]
	if !exists {
		return nil
	}

	if toolMsg, ok := a.messages[idx].(*ToolMessageItem); ok {
		toolMsg.status = ToolStatusCanceled
		// Invalidate cache - ScrollList will re-render on next View() call
		toolMsg.cachedWidth = 0
		// Only adjust scroll position if needed, no full refresh
		if a.ready && a.scrollList != nil && a.scrollList.autoScroll {
			a.scrollList.GotoBottom()
		}
	}

	return nil
}

// Append adds content to the agent output stream (legacy - calls AppendText).
func (a *AgentOutput) Append(content string) tea.Cmd {
	return a.AppendText(content)
}

// moveFocusPrevious moves focus to the previous expandable message.
// Wraps around to the last expandable message if at the beginning.
// Disables autoScroll so new content doesn't snap the view back to bottom.
func (a *AgentOutput) moveFocusPrevious() {
	if len(a.messages) == 0 {
		return
	}

	// Find expandable messages
	expandableIndices := a.findExpandableIndices()
	if len(expandableIndices) == 0 {
		return
	}

	// If no message is focused, focus the last expandable message
	if a.focusedIndex < 0 {
		a.focusedIndex = expandableIndices[len(expandableIndices)-1]
		a.scrollFocusedIntoView()
		return
	}

	// Find current position in expandable list
	currentPos := -1
	for i, idx := range expandableIndices {
		if idx == a.focusedIndex {
			currentPos = i
			break
		}
	}

	// Move to previous expandable message (wrap around if needed)
	if currentPos > 0 {
		a.focusedIndex = expandableIndices[currentPos-1]
	} else {
		// Wrap to last expandable message
		a.focusedIndex = expandableIndices[len(expandableIndices)-1]
	}

	// User is navigating away from bottom - disable auto-scroll
	if a.scrollList != nil {
		a.scrollList.SetAutoScroll(false)
		a.scrollFocusedIntoView()
	}
}

// moveFocusNext moves focus to the next expandable message.
// Wraps around to the first expandable message if at the end.
// Re-enables autoScroll if the user reaches the last message.
func (a *AgentOutput) moveFocusNext() {
	if len(a.messages) == 0 {
		return
	}

	// Find expandable messages
	expandableIndices := a.findExpandableIndices()
	if len(expandableIndices) == 0 {
		return
	}

	// If no message is focused, focus the first expandable message
	if a.focusedIndex < 0 {
		a.focusedIndex = expandableIndices[0]
		if a.scrollList != nil {
			a.scrollList.SetAutoScroll(false)
			a.scrollFocusedIntoView()
		}
		return
	}

	// Find current position in expandable list
	currentPos := -1
	for i, idx := range expandableIndices {
		if idx == a.focusedIndex {
			currentPos = i
			break
		}
	}

	// Move to next expandable message (wrap around if needed)
	if currentPos >= 0 && currentPos < len(expandableIndices)-1 {
		a.focusedIndex = expandableIndices[currentPos+1]
		// Check if we reached the last expandable message - re-enable autoScroll
		if a.scrollList != nil {
			if currentPos+1 == len(expandableIndices)-1 {
				a.scrollList.SetAutoScroll(true)
			}
			a.scrollFocusedIntoView()
		}
	} else {
		// Wrap to first expandable message
		a.focusedIndex = expandableIndices[0]
		if a.scrollList != nil {
			a.scrollList.SetAutoScroll(false)
			a.scrollFocusedIntoView()
		}
	}
}

// findExpandableIndices returns the indices of all messages that implement Expandable.
func (a *AgentOutput) findExpandableIndices() []int {
	var indices []int
	for i, msg := range a.messages {
		if _, ok := msg.(Expandable); ok {
			indices = append(indices, i)
		}
	}
	return indices
}

// scrollFocusedIntoView scrolls the ScrollList so the currently focused message is visible.
func (a *AgentOutput) scrollFocusedIntoView() {
	if a.scrollList == nil || a.focusedIndex < 0 || a.focusedIndex >= len(a.scrollList.items) {
		return
	}
	a.scrollList.ScrollToItem(a.focusedIndex)
}

// InputValue returns the current text in the input field.
func (a *AgentOutput) InputValue() string {
	return a.input.Value()
}

// ResetInput clears the input field.
func (a *AgentOutput) ResetInput() {
	a.input.SetValue("")
}

// SetInputFocused sets the focus state of the input field.
// When focused=true, the input field will be active and ready for typing.
// When focused=false, the input field will be blurred.
func (a *AgentOutput) SetInputFocused(focused bool) {
	if focused {
		a.input.Focus()
	} else {
		a.input.Blur()
	}
}

// IsInputAreaClick checks if the given screen coordinates fall within the input area.
// Returns true if the click is on the input field.
func (a *AgentOutput) IsInputAreaClick(x, y int) bool {
	// Check if click is within the input area bounds
	return x >= a.inputArea.Min.X && x < a.inputArea.Max.X &&
		y >= a.inputArea.Min.Y && y < a.inputArea.Max.Y
}

// SetScrollFocused sets the focus state of the underlying ScrollList.
// When focused, the ScrollList will handle keyboard scroll events (pgup/pgdn/home/end).
func (a *AgentOutput) SetScrollFocused(focused bool) {
	if a.scrollList != nil {
		a.scrollList.SetFocused(focused)
	}
}

// IsViewportArea checks if the given screen coordinates fall within the viewport area.
func (a *AgentOutput) IsViewportArea(x, y int) bool {
	return x >= a.viewportArea.Min.X && x < a.viewportArea.Max.X &&
		y >= a.viewportArea.Min.Y && y < a.viewportArea.Max.Y
}

// ScrollViewport scrolls the agent output viewport by the given number of lines.
// Positive values scroll down, negative values scroll up.
func (a *AgentOutput) ScrollViewport(lines int) {
	if a.scrollList == nil {
		return
	}
	a.scrollList.ScrollBy(lines)
	if lines > 0 && a.scrollList.AtBottom() {
		a.scrollList.SetAutoScroll(true)
	} else {
		a.scrollList.SetAutoScroll(false)
	}
}

// SetBusy updates the input placeholder based on whether the agent is busy.
// When busy, shows "Agent is working..." to indicate the agent is processing.
// When not busy, shows "Send a message..." to invite user input.
func (a *AgentOutput) SetBusy(busy bool) {
	if busy {
		a.input.Placeholder = "Agent is working..."
	} else {
		a.input.Placeholder = "Send a message..."
	}
}

// SetQueueDepth updates the queue depth indicator.
// Shows how many user messages are waiting in the orchestrator queue.
func (a *AgentOutput) SetQueueDepth(depth int) {
	a.queueDepth = depth
}
