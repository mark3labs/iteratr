package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// AgentOutput displays streaming agent output with auto-scroll.
type AgentOutput struct {
	scrollList        *ScrollList
	input             textinput.Model // User input field below viewport
	messages          []MessageItem
	toolIndex         map[string]int // toolCallId → message index
	queuedMsgIDs      []string       // Ordered list of queued message IDs (FIFO for finalization)
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

	// Use theme-provided textinput styles
	input.SetStyles(theme.Current().S().TextInputStyles)

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
		// Intercept PasteMsg to sanitize and collapse newlines for single-line input
		if pasteMsg, ok := msg.(tea.PasteMsg); ok {
			sanitized := SanitizePaste(pasteMsg.Content)
			// Collapse all newlines to single space for single-line textinput
			sanitized = collapseNewlines(sanitized)
			// Create new PasteMsg with sanitized content
			msg = tea.PasteMsg{Content: sanitized}
		}
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

	// Handle subagent spinner animation - forward tick to all running subagents
	var subagentSpinnerCmd tea.Cmd
	for _, m := range a.messages {
		if subagentMsg, ok := m.(*SubagentMessageItem); ok {
			if subagentMsg.spinner != nil && subagentMsg.status == ToolStatusRunning {
				if cmd := subagentMsg.spinner.Update(msg); cmd != nil {
					subagentSpinnerCmd = cmd
					// Invalidate cache to re-render with new spinner frame
					subagentMsg.cachedWidth = 0
				}
			}
		}
	}
	if subagentSpinnerCmd != nil {
		a.refreshContent()
		return subagentSpinnerCmd
	}

	// Handle keyboard input for scrolling and expand/collapse
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			// Scroll up by one line (arrows primary, j/k backup)
			if a.scrollList != nil {
				a.scrollList.ScrollBy(-1)
				a.scrollList.SetAutoScroll(false)
			}
			return nil
		case "down", "j":
			// Scroll down by one line (arrows primary, j/k backup)
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
		return theme.Current().S().Dim.Render("Waiting for agent output...")
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
		waitMsg := theme.Current().S().Dim.Render("Waiting for agent output...")
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

		styledIndicator := theme.Current().S().ScrollIndicator.Render(indicator)
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
		uv.NewStyledString(theme.Current().S().Dim.Render(separatorLine)).Draw(scr, separatorArea)

		// Draw input field below separator
		inputView := a.input.View()
		inputY := separatorY + 1

		inputWidth := inputArea.Dx() - leftPad
		inputContentArea := uv.Rect(inputArea.Min.X+leftPad, inputY, inputWidth, 1)
		uv.NewStyledString(inputView).Draw(scr, inputContentArea)

		// Draw help text below input
		if inputArea.Dy() >= 4 {
			var helpText string
			if a.input.Focused() {
				helpText = HintInputFocused()
			} else {
				helpText = HintInputBlurred()
			}
			helpY := inputY + 1
			helpArea := uv.Rect(inputArea.Min.X+leftPad, helpY, inputArea.Dx()-leftPad, 1)
			uv.NewStyledString(helpText).Draw(scr, helpArea)
		}
		// Bottom margin: last row left empty

		// Return cursor position if input is focused
		if a.input.Focused() {
			cur := a.input.Cursor()
			if cur != nil {
				// Offset cursor by the input's screen position
				cur.X += inputContentArea.Min.X
				cur.Y += inputContentArea.Min.Y
			}
			return cur
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
		a.scrollList.SetItemGap(1)
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
		spinner := NewDefaultGradientSpinner("Generating...")
		a.spinner = &spinner
		spinnerCmd = a.spinner.Tick()
	}

	// Stop spinner when actual content arrives
	if a.isStreaming && content != "" {
		a.isStreaming = false
		a.spinner = nil
	}

	// If last non-queued message is a TextMessageItem, append to it
	if lastMsg, _ := a.lastNonQueuedMessage(); lastMsg != nil {
		if textMsg, ok := lastMsg.(*TextMessageItem); ok {
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

	// Create new TextMessageItem (inserted before any trailing queued messages)
	newMsg := &TextMessageItem{
		id:      fmt.Sprintf("text-%d", len(a.messages)),
		content: content,
	}
	a.appendBeforeQueued(newMsg)
	a.refreshContent()
	return spinnerCmd
}

// AppendToolCall handles tool lifecycle events.
// If toolCallId not in toolIndex: append new message, store index.
// If toolCallId exists: update message in-place (status, input, output).
func (a *AgentOutput) AppendToolCall(msg AgentToolCallMsg) tea.Cmd {
	idx, exists := a.toolIndex[msg.ToolCallID]
	if !exists {
		// Map status strings to ToolStatus enum
		status := mapToolStatus(msg.Status)

		// Detect subagent call by checking Input["subagent_type"]
		subagentType, isSubagent := msg.Input["subagent_type"].(string)

		if isSubagent {
			// Create SubagentMessageItem for subagent tasks
			description, _ := msg.Input["prompt"].(string)
			newMsg := &SubagentMessageItem{
				id:           msg.ToolCallID,
				subagentType: subagentType,
				description:  description,
				status:       status,
				sessionID:    msg.SessionID, // May be empty initially
			}
			// Start spinner if running
			if status == ToolStatusRunning {
				spinner := NewDefaultSpinner()
				newMsg.spinner = &spinner
				a.appendBeforeQueued(newMsg)
				a.toolIndex[msg.ToolCallID] = a.indexOfMessage(msg.ToolCallID)
				a.refreshContent()
				return newMsg.spinner.Tick()
			}
			a.appendBeforeQueued(newMsg)
			a.toolIndex[msg.ToolCallID] = a.indexOfMessage(msg.ToolCallID)
			a.refreshContent()
		} else {
			// Create regular ToolMessageItem
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
			a.appendBeforeQueued(newMsg)
			a.toolIndex[msg.ToolCallID] = a.indexOfMessage(msg.ToolCallID)
			a.refreshContent()
		}
	} else {
		// Update existing tool call in-place
		// Check if it's a SubagentMessageItem or ToolMessageItem
		if subagentMsg, ok := a.messages[idx].(*SubagentMessageItem); ok {
			// Update SubagentMessageItem
			newStatus := mapToolStatus(msg.Status)
			oldStatus := subagentMsg.status
			subagentMsg.status = newStatus
			if msg.SessionID != "" {
				subagentMsg.sessionID = msg.SessionID
			}
			// Manage spinner based on status transition
			var spinnerCmd tea.Cmd
			if newStatus == ToolStatusRunning && oldStatus != ToolStatusRunning {
				// Started running - create spinner
				spinner := NewDefaultSpinner()
				subagentMsg.spinner = &spinner
				spinnerCmd = subagentMsg.spinner.Tick()
			} else if newStatus != ToolStatusRunning && subagentMsg.spinner != nil {
				// No longer running - clear spinner
				subagentMsg.spinner = nil
			}
			// Invalidate cache - ScrollList will re-render on next View() call
			subagentMsg.cachedWidth = 0
			// Only adjust scroll position if needed, no full refresh
			if a.ready && a.scrollList != nil && a.scrollList.autoScroll {
				a.scrollList.GotoBottom()
			}
			if spinnerCmd != nil {
				return spinnerCmd
			}
		} else if toolMsg, ok := a.messages[idx].(*ToolMessageItem); ok {
			// Check if this is a subagent call that we missed on initial creation
			// (RawInput is empty on pending, only populated on in_progress)
			if subagentType, isSubagent := msg.Input["subagent_type"].(string); isSubagent {
				// Convert ToolMessageItem to SubagentMessageItem
				description, _ := msg.Input["prompt"].(string)
				status := mapToolStatus(msg.Status)
				newMsg := &SubagentMessageItem{
					id:           msg.ToolCallID,
					subagentType: subagentType,
					description:  description,
					status:       status,
					sessionID:    msg.SessionID,
				}
				// Start spinner if running
				if status == ToolStatusRunning {
					spinner := NewDefaultSpinner()
					newMsg.spinner = &spinner
					a.messages[idx] = newMsg
					a.refreshContent()
					return newMsg.spinner.Tick()
				}
				a.messages[idx] = newMsg
				a.refreshContent()
				return nil
			}

			// Update regular ToolMessageItem
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
	a.appendBeforeQueued(newMsg)
	a.refreshContent()
	return nil
}

// AppendHook adds a new hook message to the output in running state.
// Returns the generated ID for later updates via UpdateHook.
func (a *AgentOutput) AppendHook(hookID, hookType, command string) tea.Cmd {
	newMsg := &HookMessageItem{
		id:        hookID,
		hookType:  hookType,
		command:   command,
		status:    HookStatusRunning,
		collapsed: true,
		maxLines:  10,
	}
	a.appendBeforeQueued(newMsg)
	a.toolIndex[hookID] = a.indexOfMessage(hookID)
	a.refreshContent()
	return nil
}

// UpdateHook updates an existing hook message with completion status and output.
func (a *AgentOutput) UpdateHook(hookID string, status HookStatus, output string, duration time.Duration) tea.Cmd {
	idx, exists := a.toolIndex[hookID]
	if !exists {
		return nil
	}

	hookMsg, ok := a.messages[idx].(*HookMessageItem)
	if !ok {
		return nil
	}

	hookMsg.status = status
	hookMsg.output = output
	hookMsg.duration = duration
	// Invalidate cache
	hookMsg.cachedWidth = 0
	a.refreshContent()
	return nil
}

// HandleClick processes a mouse click at screen coordinates (x, y).
// Returns a command if the click should open a modal (e.g., subagent viewer).
// Returns nil if the click toggled an expandable message or had no effect.
func (a *AgentOutput) HandleClick(x, y int) tea.Cmd {
	if !a.ready || len(a.messageLineStarts) == 0 {
		return nil
	}

	// Check if click is within the viewport area
	if x < a.viewportArea.Min.X || x >= a.viewportArea.Max.X ||
		y < a.viewportArea.Min.Y || y >= a.viewportArea.Max.Y {
		return nil
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
		return nil
	}

	// Check if it's a SubagentMessageItem with sessionID (clickable to view)
	if subagentMsg, ok := a.messages[msgIdx].(*SubagentMessageItem); ok {
		if subagentMsg.sessionID != "" {
			// Return command to open subagent modal
			return func() tea.Msg {
				return OpenSubagentModalMsg{
					SessionID:    subagentMsg.sessionID,
					SubagentType: subagentMsg.subagentType,
				}
			}
		}
		return nil
	}

	// Toggle if expandable
	if expandable, ok := a.messages[msgIdx].(Expandable); ok {
		expandable.ToggleExpanded()
		a.refreshContent()
		return nil
	}

	return nil
}

// indexOfMessage returns the index of a message with the given ID, or -1 if not found.
func (a *AgentOutput) indexOfMessage(id string) int {
	for i, msg := range a.messages {
		if msg.ID() == id {
			return i
		}
	}
	return -1
}

// lastNonQueuedMessage returns the last message that is not a QueuedUserMessageItem,
// along with its index. Returns nil, -1 if no such message exists.
func (a *AgentOutput) lastNonQueuedMessage() (MessageItem, int) {
	for i := len(a.messages) - 1; i >= 0; i-- {
		if _, ok := a.messages[i].(*QueuedUserMessageItem); !ok {
			return a.messages[i], i
		}
	}
	return nil, -1
}

// appendBeforeQueued inserts a message before any trailing QueuedUserMessageItems.
// This ensures queued messages always stay at the bottom of the message list.
// It also updates toolIndex entries for any shifted messages.
func (a *AgentOutput) appendBeforeQueued(msg MessageItem) {
	// Find the insertion point: right before the first trailing queued message
	insertAt := len(a.messages)
	for i := len(a.messages) - 1; i >= 0; i-- {
		if _, ok := a.messages[i].(*QueuedUserMessageItem); ok {
			insertAt = i
		} else {
			break
		}
	}

	if insertAt == len(a.messages) {
		// No trailing queued messages, just append normally
		a.messages = append(a.messages, msg)
		return
	}

	// Insert at the position before queued messages
	a.messages = append(a.messages, nil) // grow by one
	copy(a.messages[insertAt+1:], a.messages[insertAt:])
	a.messages[insertAt] = msg

	// Update toolIndex for any entries that were shifted
	for id, idx := range a.toolIndex {
		if idx >= insertAt {
			a.toolIndex[id] = idx + 1
		}
	}
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
		spinner := NewDefaultGradientSpinner("Thinking...")
		a.spinner = &spinner
		spinnerCmd = a.spinner.Tick()
	}

	// Stop spinner when actual content arrives
	if a.isStreaming && content != "" {
		a.isStreaming = false
		a.spinner = nil
	}

	// If last non-queued message is a ThinkingMessageItem, append to it
	if lastMsg, _ := a.lastNonQueuedMessage(); lastMsg != nil {
		if thinkingMsg, ok := lastMsg.(*ThinkingMessageItem); ok {
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

	// Create new ThinkingMessageItem (inserted before any trailing queued messages)
	newMsg := &ThinkingMessageItem{
		id:        fmt.Sprintf("thinking-%d", len(a.messages)),
		content:   content,
		collapsed: true, // default true
	}
	a.appendBeforeQueued(newMsg)
	a.refreshContent()
	return spinnerCmd
}

// AppendUserMessage adds a user message to the conversation viewport.
// User messages appear when the agent receives them (not when user sends),
// preserving accurate conversation order. Inserted before any queued messages.
func (a *AgentOutput) AppendUserMessage(text string) tea.Cmd {
	// Generate unique ID with nanosecond timestamp
	id := fmt.Sprintf("user-%d", time.Now().UnixNano())

	// Create new UserMessageItem
	newMsg := &UserMessageItem{
		id:      id,
		content: text,
	}

	// Insert before any trailing queued messages
	a.appendBeforeQueued(newMsg)

	// Refresh content and scroll to bottom
	a.refreshContent()

	return nil
}

// AppendQueuedUserMessage adds a queued user message to the conversation viewport.
// The message is shown immediately with a QUEUED badge, sticky to the bottom.
// Returns the generated message ID for later finalization.
func (a *AgentOutput) AppendQueuedUserMessage(text string) (string, tea.Cmd) {
	id := fmt.Sprintf("queued-%d", time.Now().UnixNano())

	newMsg := &QueuedUserMessageItem{
		id:      id,
		content: text,
	}

	a.messages = append(a.messages, newMsg)
	a.queuedMsgIDs = append(a.queuedMsgIDs, id)
	a.refreshContent()

	return id, nil
}

// FinalizeQueuedMessage converts the oldest queued message into a regular UserMessageItem.
// The message text is matched by the orchestrator's processing order (FIFO).
func (a *AgentOutput) FinalizeQueuedMessage(text string) tea.Cmd {
	if len(a.queuedMsgIDs) == 0 {
		// No queued messages to finalize; fall back to append
		return a.AppendUserMessage(text)
	}

	// Pop the oldest queued message ID
	targetID := a.queuedMsgIDs[0]
	a.queuedMsgIDs = a.queuedMsgIDs[1:]

	// Find and replace in messages slice
	for i, msg := range a.messages {
		if msg.ID() == targetID {
			a.messages[i] = &UserMessageItem{
				id:      targetID,
				content: text,
			}
			a.refreshContent()
			return nil
		}
	}

	// If not found (shouldn't happen), fall back to append
	return a.AppendUserMessage(text)
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

	// 1. Mark last non-queued ThinkingMessageItem as finished with duration
	if lastMsg, _ := a.lastNonQueuedMessage(); lastMsg != nil {
		if thinkingMsg, ok := lastMsg.(*ThinkingMessageItem); ok {
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

	// 2. Append InfoMessageItem with model/provider/duration (before queued messages)
	infoMsg := &InfoMessageItem{
		id:       fmt.Sprintf("info-%d", len(a.messages)),
		model:    msg.Model,
		provider: msg.Provider,
		duration: msg.Duration,
	}
	a.appendBeforeQueued(infoMsg)

	// 3. If error or canceled, append styled finish reason (before queued messages)
	if msg.Error != "" {
		// Error finish
		errorText := theme.Current().S().FinishError.Render(fmt.Sprintf("Error: %s", msg.Error))
		errorItem := &TextMessageItem{
			id:      fmt.Sprintf("finish-error-%d", len(a.messages)),
			content: errorText,
		}
		a.appendBeforeQueued(errorItem)
	} else if msg.Reason == "cancelled" {
		// Canceled finish
		cancelText := theme.Current().S().FinishCanceled.Render("Iteration canceled")
		cancelItem := &TextMessageItem{
			id:      fmt.Sprintf("finish-cancel-%d", len(a.messages)),
			content: cancelText,
		}
		a.appendBeforeQueued(cancelItem)
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

// SetQueueDepth updates the queue depth counter.
// Queued messages are now shown inline in the viewport, so this only tracks the count.
func (a *AgentOutput) SetQueueDepth(depth int) {
	a.queueDepth = depth
}
