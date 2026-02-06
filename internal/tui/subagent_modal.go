package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/agent"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// SubagentModal displays a full-screen modal that loads and replays a subagent session.
// It reuses the existing ScrollList and MessageItem infrastructure from AgentOutput.
type SubagentModal struct {
	// Content display (reuses AgentOutput infrastructure)
	scrollList *ScrollList    // For scrolling and rendering
	messages   []MessageItem  // Message accumulation
	toolIndex  map[string]int // toolCallId → message index

	// Session metadata
	sessionID    string
	subagentType string
	workDir      string

	// ACP subprocess (populated by Start())
	loader *agent.SessionLoader

	// State
	loading bool
	err     error // Non-nil shows error message in modal

	// Spinner for loading state (created lazily when needed)
	spinner *GradientSpinner

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Mouse interaction support
	viewportArea      uv.Rectangle // Screen area where content is drawn (for mouse hit detection)
	messageLineStarts []int        // Start line index in content for each message

	// First user message tracking (omit the task prompt)
	skippedFirstUser bool
}

// NewSubagentModal creates a new SubagentModal.
// Initial dimensions are placeholder - will be updated on first Draw().
func NewSubagentModal(sessionID, subagentType, workDir string) *SubagentModal {
	ctx, cancel := context.WithCancel(context.Background())
	spinner := NewDefaultGradientSpinner("Loading session...")
	return &SubagentModal{
		sessionID:    sessionID,
		subagentType: subagentType,
		workDir:      workDir,
		scrollList:   NewScrollList(80, 20), // Placeholder dimensions
		messages:     make([]MessageItem, 0),
		toolIndex:    make(map[string]int),
		loading:      true,
		ctx:          ctx,
		cancel:       cancel,
		spinner:      &spinner,
	}
}

// Start spawns the ACP subprocess, initializes it, and begins loading the session.
// Returns a command that will start the session loading process.
func (m *SubagentModal) Start() tea.Cmd {
	return func() tea.Msg {
		// Spawn SessionLoader subprocess
		loader, err := agent.NewSessionLoader(m.ctx, m.workDir)
		if err != nil {
			logger.Warn("Failed to start ACP subprocess for subagent modal: %v", err)
			return SubagentErrorMsg{Err: fmt.Errorf("failed to start ACP: %w", err)}
		}
		m.loader = loader

		// Load the session (triggers replay)
		logger.Debug("subagent modal: loading session %s", m.sessionID)
		if err := loader.LoadAndStream(m.ctx, m.sessionID, m.workDir); err != nil {
			logger.Warn("Failed to load session %s: %v", m.sessionID, err)
			return SubagentErrorMsg{Err: fmt.Errorf("session not found: %s", m.sessionID)}
		}

		// Session loading started - modal no longer in loading state
		logger.Debug("subagent modal: session loaded, starting stream")
		m.loading = false

		// Start streaming notifications
		return m.streamNext()
	}
}

// Draw renders the modal as a full-screen overlay.
// Handles three states: loading (spinner), error (message), and content (scroll list).
func (m *SubagentModal) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// Calculate modal dimensions (full-screen with small margins)
	modalWidth := area.Dx() - 4
	modalHeight := area.Dy() - 4
	if modalWidth < 40 {
		modalWidth = area.Dx()
	}
	if modalHeight < 10 {
		modalHeight = area.Dy()
	}

	// Calculate content area dimensions
	contentWidth := modalWidth - 6   // Account for border (2) + padding (4)
	contentHeight := modalHeight - 5 // Account for padding (2) + title (1) + separator (1) + hint (1)
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	s := theme.Current().S()

	// Build title with subagent type
	titleText := fmt.Sprintf("Subagent: %s", m.subagentType)
	title := renderModalTitle(titleText, contentWidth)
	separator := s.ModalSeparator.Render(strings.Repeat("─", contentWidth))

	// Build content based on state
	var content string
	var hint string

	if m.err != nil {
		// Error state: show error message centered both vertically and horizontally
		errorMsg := s.Error.Render(fmt.Sprintf("× %s", m.err.Error()))

		// Center error message both vertically and horizontally
		errorCentered := lipgloss.NewStyle().
			Width(contentWidth).
			Height(contentHeight).
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(errorMsg)

		content = strings.Join([]string{
			title,
			separator,
			errorCentered,
		}, "\n")
		hint = RenderHint(KeyEsc, "close")

	} else if m.loading {
		// Loading state: show spinner centered both vertically and horizontally
		spinnerView := ""
		if m.spinner != nil {
			spinnerView = m.spinner.View()
		}

		// Center spinner both vertically and horizontally
		spinnerCentered := lipgloss.NewStyle().
			Width(contentWidth).
			Height(contentHeight).
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(spinnerView)

		content = strings.Join([]string{
			title,
			separator,
			spinnerCentered,
		}, "\n")
		hint = RenderHint(KeyEsc, "close")

	} else {
		// Content state: show session history via scrollList
		// Update scrollList dimensions to match content area
		m.scrollList.SetWidth(contentWidth)
		m.scrollList.SetHeight(contentHeight)

		// Get scrollList view
		listContent := m.scrollList.View()

		// Debug: log if content is empty
		if listContent == "" && len(m.messages) == 0 {
			logger.Debug("subagent modal Draw: no messages to display")
		}

		// Pad content to fill available height so hint stays at bottom
		// Count actual lines in listContent
		actualLines := strings.Count(listContent, "\n") + 1
		if listContent == "" {
			actualLines = 0
		}
		if actualLines < contentHeight {
			padding := strings.Repeat("\n", contentHeight-actualLines)
			listContent = listContent + padding
		}

		content = strings.Join([]string{
			title,
			separator,
			listContent,
		}, "\n")
		hint = RenderHintBar(KeyEsc, "close", KeyUpDown, "scroll")

		// Calculate viewport area for mouse interaction
		// Modal border/padding: 1 top padding, 3 left (border 1 + padding 2)
		titleLines := 1
		separatorLines := 1
		topPadding := 1
		leftPadding := 3

		// Calculate modal position (will be centered on screen)
		modalX := (area.Dx() - modalWidth) / 2
		modalY := (area.Dy() - modalHeight) / 2
		if modalX < 0 {
			modalX = 0
		}
		if modalY < 0 {
			modalY = 0
		}

		// Viewport area within the modal
		m.viewportArea = uv.Rectangle{
			Min: uv.Position{
				X: area.Min.X + modalX + leftPadding,
				Y: area.Min.Y + modalY + topPadding + titleLines + separatorLines,
			},
			Max: uv.Position{
				X: area.Min.X + modalX + leftPadding + contentWidth,
				Y: area.Min.Y + modalY + topPadding + titleLines + separatorLines + contentHeight,
			},
		}
	}

	// Add hint at bottom
	content = strings.Join([]string{content, hint}, "\n")

	// Style the modal
	modalStyle := s.ModalContainer.
		Width(modalWidth).
		Height(modalHeight)

	modalContent := modalStyle.Render(content)

	// Center on screen
	renderedWidth := lipgloss.Width(modalContent)
	renderedHeight := lipgloss.Height(modalContent)
	x := (area.Dx() - renderedWidth) / 2
	y := (area.Dy() - renderedHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	modalArea := uv.Rectangle{
		Min: uv.Position{X: area.Min.X + x, Y: area.Min.Y + y},
		Max: uv.Position{X: area.Min.X + x + renderedWidth, Y: area.Min.Y + y + renderedHeight},
	}
	uv.NewStyledString(modalContent).Draw(scr, modalArea)

	return nil
}

// Update handles keyboard input for scrolling.
// Forwards scroll key events (up/down/pgup/pgdown/home/end) to the internal scrollList.
func (m *SubagentModal) Update(msg tea.Msg) tea.Cmd {
	if m.scrollList == nil {
		return nil
	}

	// Set scrollList as focused to enable keyboard handling
	m.scrollList.SetFocused(true)
	defer m.scrollList.SetFocused(false)

	// Forward message to scrollList
	return m.scrollList.Update(msg)
}

// HandleUpdate processes streaming messages from the subagent session.
// Returns a command to continue streaming if Continue is true.
func (m *SubagentModal) HandleUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case SubagentTextMsg:
		m.appendText(msg.Text)
		if msg.Continue {
			return m.streamNext
		}

	case SubagentToolCallMsg:
		m.appendToolCall(msg.Event)
		if msg.Continue {
			return m.streamNext
		}

	case SubagentThinkingMsg:
		m.appendThinking(msg.Content)
		if msg.Continue {
			return m.streamNext
		}

	case SubagentUserMsg:
		m.appendUserMessage(msg.Text)
		if msg.Continue {
			return m.streamNext
		}

	case SubagentStreamMsg:
		if msg.Continue {
			return m.streamNext
		}
	}

	return nil
}

// streamNext reads the next message from the session stream and returns a tea.Cmd.
// Runs in a background goroutine to avoid blocking the TUI event loop.
func (m *SubagentModal) streamNext() tea.Msg {
	// Guard: loader must be initialized
	if m.loader == nil {
		logger.Debug("subagent modal: loader is nil, returning done")
		return SubagentDoneMsg{}
	}

	// Read and process one notification from session stream
	// The loader calls our callbacks which capture the data
	var textData string
	var toolData agent.ToolCallEvent
	var thinkingData string
	var userData string
	var msgType string // "text", "tool", "thinking", "user", ""

	processed, err := m.loader.ReadAndProcess(
		func(text string) {
			textData = text
			msgType = "text"
		},
		func(event agent.ToolCallEvent) {
			toolData = event
			msgType = "tool"
		},
		func(content string) {
			thinkingData = content
			msgType = "thinking"
		},
		func(text string) {
			userData = text
			msgType = "user"
		},
	)

	if err != nil {
		// EOF means session replay is complete
		if err.Error() == "EOF" {
			logger.Debug("subagent modal: EOF reached, session replay complete, messages=%d", len(m.messages))
			return SubagentDoneMsg{}
		}
		// Other errors are real failures
		logger.Warn("Failed to read session message: %v", err)
		return SubagentErrorMsg{Err: fmt.Errorf("stream error: %w", err)}
	}

	if !processed {
		// No more messages
		logger.Debug("subagent modal: no message processed, returning done, messages=%d", len(m.messages))
		return SubagentDoneMsg{}
	}

	logger.Debug("subagent modal: processed message type=%s", msgType)

	// Return appropriate message based on what was processed
	switch msgType {
	case "text":
		return SubagentTextMsg{Text: textData, Continue: true}
	case "tool":
		return SubagentToolCallMsg{Event: toolData, Continue: true}
	case "thinking":
		return SubagentThinkingMsg{Content: thinkingData, Continue: true}
	case "user":
		return SubagentUserMsg{Text: userData, Continue: true}
	default:
		// Unknown or skipped notification - continue streaming
		return SubagentStreamMsg{Continue: true}
	}
}

// appendText appends agent text content to the message list.
// Mirrors AgentOutput.AppendText behavior.
func (m *SubagentModal) appendText(content string) {
	// If last message is a TextMessageItem, append to it
	if len(m.messages) > 0 {
		if textMsg, ok := m.messages[len(m.messages)-1].(*TextMessageItem); ok {
			textMsg.content += content
			// Invalidate cache - ScrollList will re-render on next View() call
			textMsg.cachedWidth = 0
			// Auto-scroll to bottom if needed
			if m.scrollList != nil && m.scrollList.autoScroll {
				m.scrollList.GotoBottom()
			}
			return
		}
	}

	// Create new TextMessageItem
	newMsg := &TextMessageItem{
		id:      fmt.Sprintf("text-%d", len(m.messages)),
		content: content,
	}
	m.messages = append(m.messages, newMsg)
	m.refreshContent()
}

// appendToolCall appends or updates a tool call in the message list.
// Mirrors AgentOutput.AppendToolCall behavior.
func (m *SubagentModal) appendToolCall(event agent.ToolCallEvent) {
	idx, exists := m.toolIndex[event.ToolCallID]
	if !exists {
		// Map status strings to ToolStatus enum
		status := mapToolStatus(event.Status)

		// Create new ToolMessageItem (always use ToolMessageItem, not SubagentMessageItem)
		newMsg := &ToolMessageItem{
			id:       event.ToolCallID,
			toolName: event.Title,
			status:   status,
			input:    event.RawInput,
			output:   event.Output,
			kind:     event.Kind,
			fileDiff: convertFileDiff(event.FileDiff),
		}
		m.messages = append(m.messages, newMsg)
		m.toolIndex[event.ToolCallID] = len(m.messages) - 1
		m.refreshContent()
	} else {
		// Update existing ToolMessageItem
		if toolMsg, ok := m.messages[idx].(*ToolMessageItem); ok {
			toolMsg.status = mapToolStatus(event.Status)
			if event.RawInput != nil {
				toolMsg.input = event.RawInput
			}
			if event.Output != "" {
				toolMsg.output = event.Output
			}
			if event.FileDiff != nil {
				toolMsg.fileDiff = convertFileDiff(event.FileDiff)
			}
			// Invalidate cache
			toolMsg.cachedWidth = 0
			m.refreshContent()
		}
	}
}

// appendThinking appends agent thinking content to the message list.
// Mirrors AgentOutput.AppendThinking behavior.
func (m *SubagentModal) appendThinking(content string) {
	// If last message is a ThinkingMessageItem, append to it
	if len(m.messages) > 0 {
		if thinkingMsg, ok := m.messages[len(m.messages)-1].(*ThinkingMessageItem); ok {
			thinkingMsg.content += content
			// Invalidate cache
			thinkingMsg.cachedWidth = 0
			// Auto-scroll to bottom if needed
			if m.scrollList != nil && m.scrollList.autoScroll {
				m.scrollList.GotoBottom()
			}
			return
		}
	}

	// Create new ThinkingMessageItem
	newMsg := &ThinkingMessageItem{
		id:      fmt.Sprintf("thinking-%d", len(m.messages)),
		content: content,
	}
	m.messages = append(m.messages, newMsg)
	m.refreshContent()
}

// appendUserMessage appends a user message to the message list.
// Mirrors AgentOutput.AppendUserMessage behavior.
// Skips the first user message (the task prompt) as it's redundant in the modal view.
func (m *SubagentModal) appendUserMessage(text string) {
	// Skip the first user message (the task prompt)
	if !m.skippedFirstUser {
		m.skippedFirstUser = true
		return
	}

	// If last message is a UserMessageItem, append to it
	if len(m.messages) > 0 {
		if userMsg, ok := m.messages[len(m.messages)-1].(*UserMessageItem); ok {
			userMsg.content += text
			// Invalidate cache
			userMsg.cachedWidth = 0
			// Auto-scroll to bottom if needed
			if m.scrollList != nil && m.scrollList.autoScroll {
				m.scrollList.GotoBottom()
			}
			return
		}
	}

	// Create new UserMessageItem
	newMsg := &UserMessageItem{
		id:      fmt.Sprintf("user-%d", len(m.messages)),
		content: text,
	}
	m.messages = append(m.messages, newMsg)
	m.refreshContent()
}

// refreshContent updates the ScrollList with the current message items.
func (m *SubagentModal) refreshContent() {
	if m.scrollList == nil {
		return
	}

	// Convert []MessageItem to []ScrollItem
	items := make([]ScrollItem, len(m.messages))
	for i, msg := range m.messages {
		items[i] = msg
	}
	m.scrollList.SetItems(items)

	// Compute messageLineStarts for click-to-expand hit detection.
	// Must include gap lines between items to match currentOffsetInLines.
	m.messageLineStarts = make([]int, len(items))
	offset := 0
	gap := m.scrollList.ItemGap()
	for i, item := range items {
		m.messageLineStarts[i] = offset
		// Always render to get accurate height after expand/collapse toggle
		item.Render(m.scrollList.width)
		offset += item.Height()
		if gap > 0 && i < len(items)-1 {
			offset += gap
		}
	}

	// Auto-scroll to bottom if enabled
	if m.scrollList.autoScroll {
		m.scrollList.GotoBottom()
	}
}

// convertFileDiff converts agent.FileDiff to local FileDiff type.
func convertFileDiff(agentDiff *agent.FileDiff) *FileDiff {
	if agentDiff == nil {
		return nil
	}
	return &FileDiff{
		File:      agentDiff.File,
		Before:    agentDiff.Before,
		After:     agentDiff.After,
		Additions: agentDiff.Additions,
		Deletions: agentDiff.Deletions,
	}
}

// HandleClick processes a mouse click at screen coordinates (x, y).
// Returns nil after toggling an expandable message.
func (m *SubagentModal) HandleClick(x, y int) tea.Cmd {
	if len(m.messageLineStarts) == 0 || m.scrollList == nil {
		return nil
	}

	// Check if click is within the viewport area
	if x < m.viewportArea.Min.X || x >= m.viewportArea.Max.X ||
		y < m.viewportArea.Min.Y || y >= m.viewportArea.Max.Y {
		return nil
	}

	// Translate screen Y to content line (accounting for scroll offset)
	contentLine := (y - m.viewportArea.Min.Y) + m.scrollList.currentOffsetInLines()

	// Find which message this line belongs to
	msgIdx := -1
	for i := len(m.messageLineStarts) - 1; i >= 0; i-- {
		if contentLine >= m.messageLineStarts[i] {
			msgIdx = i
			break
		}
	}

	if msgIdx < 0 || msgIdx >= len(m.messages) {
		return nil
	}

	// Toggle if expandable
	if expandable, ok := m.messages[msgIdx].(Expandable); ok {
		expandable.ToggleExpanded()
		m.refreshContent()
	}

	return nil
}

// ScrollViewport scrolls the modal content by the given number of lines.
// Positive values scroll down, negative values scroll up.
func (m *SubagentModal) ScrollViewport(lines int) {
	if m.scrollList == nil {
		return
	}
	m.scrollList.ScrollBy(lines)
	// Disable auto-scroll when user scrolls up
	if lines < 0 {
		m.scrollList.SetAutoScroll(false)
	} else if m.scrollList.AtBottom() {
		m.scrollList.SetAutoScroll(true)
	}
}

// Close terminates the ACP subprocess and cleans up resources.
// Safe to call multiple times or if Start() was never called.
func (m *SubagentModal) Close() {
	// Cancel context to stop any ongoing operations
	if m.cancel != nil {
		m.cancel()
	}

	// Close SessionLoader if established
	if m.loader != nil {
		if err := m.loader.Close(); err != nil {
			logger.Warn("Failed to close session loader: %v", err)
		}
		m.loader = nil
	}
}
