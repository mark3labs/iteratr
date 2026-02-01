package tui

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	chroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	udiff "github.com/aymanbagabas/go-udiff"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// MessageItem represents a displayable message item in the agent output.
type MessageItem interface {
	// ID returns the unique identifier for this message item.
	ID() string
	// Render returns the rendered string representation at the given width.
	Render(width int) string
	// Height returns the number of lines this item occupies (0 if not yet rendered).
	Height() int
}

// Expandable is an optional interface for message items that support expand/collapse.
type Expandable interface {
	IsExpanded() bool
	ToggleExpanded()
}

// ToolStatus represents the execution status of a tool call.
type ToolStatus int

const (
	ToolStatusPending ToolStatus = iota
	ToolStatusRunning
	ToolStatusSuccess
	ToolStatusError
	ToolStatusCanceled
)

// TextMessageItem represents assistant text content.
type TextMessageItem struct {
	id           string
	content      string
	cachedRender string
	cachedWidth  int
}

// UserMessageItem represents user text content.
type UserMessageItem struct {
	id           string
	content      string
	cachedRender string
	cachedWidth  int
}

// ID returns the unique identifier for this text message.
func (t *TextMessageItem) ID() string {
	return t.id
}

// Render renders the text message at the given width.
// Wraps text, applies assistant border, and caps width at min(width-2, 120).
func (t *TextMessageItem) Render(width int) string {
	// Return cached render if width matches
	if t.cachedWidth == width && t.cachedRender != "" {
		return t.cachedRender
	}

	// Cap effective width at min(width-2, 120) to prevent overly long lines
	// Subtract 2 for the left border and padding added by styleAssistantBorder
	effectiveWidth := width - 2
	if effectiveWidth > 120 {
		effectiveWidth = 120
	}
	if effectiveWidth < 1 {
		effectiveWidth = 1
	}

	// Render content as markdown with syntax highlighting
	// Falls back to plain text wrapping if glamour fails
	rendered := renderMarkdown(t.content, effectiveWidth)

	// Apply assistant border styling (left border + padding)
	result := theme.Current().S().AssistantBorder.Render(rendered)

	// Cache and return
	t.cachedRender = result
	t.cachedWidth = width
	return result
}

// Height returns the number of lines this text message occupies.
func (t *TextMessageItem) Height() int {
	if t.cachedRender == "" {
		return 0
	}
	// Count newlines in cached render
	lines := 1
	for _, ch := range t.cachedRender {
		if ch == '\n' {
			lines++
		}
	}
	return lines
}

// ID returns the unique identifier for this user message.
func (u *UserMessageItem) ID() string {
	return u.id
}

// Render renders the user message at the given width.
// Wraps text, applies user border (right border), right-aligns content, and caps width at min(width-2, 120).
func (u *UserMessageItem) Render(width int) string {
	// Return cached render if width matches
	if u.cachedWidth == width && u.cachedRender != "" {
		return u.cachedRender
	}

	// Cap effective width at min(width-2, 120) to prevent overly long lines
	// Subtract 2 for the right border and padding added by styleUserBorder
	effectiveWidth := width - 2
	if effectiveWidth > 120 {
		effectiveWidth = 120
	}
	if effectiveWidth < 1 {
		effectiveWidth = 1
	}

	// Wrap text (plain text, no markdown for user messages)
	wrapped := wrapText(u.content, effectiveWidth)

	// Apply user border styling (right border + padding)
	styled := theme.Current().S().UserBorder.Render(wrapped)

	// Right-align the entire styled block
	result := rightAlign(styled, width)

	// Cache and return
	u.cachedRender = result
	u.cachedWidth = width
	return result
}

// Height returns the number of lines this user message occupies.
func (u *UserMessageItem) Height() int {
	if u.cachedRender == "" {
		return 0
	}
	// Count newlines in cached render
	lines := 1
	for _, ch := range u.cachedRender {
		if ch == '\n' {
			lines++
		}
	}
	return lines
}

// ThinkingMessageItem represents agent thinking/reasoning content.
type ThinkingMessageItem struct {
	id           string
	content      string
	collapsed    bool // default true
	duration     time.Duration
	finished     bool
	cachedRender string
	cachedWidth  int
}

// ID returns the unique identifier for this thinking message.
func (t *ThinkingMessageItem) ID() string {
	return t.id
}

// Render renders the thinking message at the given width.
// If collapsed and content has >10 lines, shows last 10 with truncation hint.
// Adds footer "Thought for Xs" if finished. Wraps in styleThinkingBox.
func (t *ThinkingMessageItem) Render(width int) string {
	// Return cached render if width matches
	if t.cachedWidth == width && t.cachedRender != "" {
		return t.cachedRender
	}

	var result strings.Builder

	// Split content into lines
	lines := strings.Split(t.content, "\n")

	// If collapsed and >10 lines, show last 10 with truncation hint
	var displayLines []string
	hiddenCount := 0
	if t.collapsed && len(lines) > 10 {
		hiddenCount = len(lines) - 10
		displayLines = lines[len(lines)-10:]
	} else {
		displayLines = lines
	}

	// Add truncation hint if we hid lines
	if hiddenCount > 0 {
		hint := theme.Current().S().ThinkingTruncationHint.Render(
			fmt.Sprintf("… (%d lines hidden)", hiddenCount),
		)
		result.WriteString(theme.Current().S().ThinkingContent.Render(hint))
		result.WriteString("\n")
	}

	// Render visible lines with thinking content style
	for i, line := range displayLines {
		if i > 0 || hiddenCount > 0 {
			result.WriteString("\n")
		}
		result.WriteString(theme.Current().S().ThinkingContent.Render(line))
	}

	// Add footer with duration if finished
	if t.finished && t.duration > 0 {
		result.WriteString("\n")
		durationStr := formatDuration(t.duration)
		footer := theme.Current().S().ThinkingFooter.Render("Thought for ") +
			theme.Current().S().ThinkingDuration.Render(durationStr)
		result.WriteString(footer)
	}

	// Wrap entire content in thinking box style with full-width background
	boxed := theme.Current().S().ThinkingBox.Width(width).Render(result.String())

	// Cache and return
	t.cachedRender = boxed
	t.cachedWidth = width
	return boxed
}

// Height returns the number of lines this thinking message occupies.
func (t *ThinkingMessageItem) Height() int {
	if t.cachedRender == "" {
		return 0
	}
	// Count newlines in cached render
	lines := 1
	for _, ch := range t.cachedRender {
		if ch == '\n' {
			lines++
		}
	}
	return lines
}

// IsExpanded returns whether the thinking message is expanded.
func (t *ThinkingMessageItem) IsExpanded() bool {
	return !t.collapsed
}

// ToggleExpanded toggles the expanded/collapsed state.
func (t *ThinkingMessageItem) ToggleExpanded() {
	t.collapsed = !t.collapsed
	// Invalidate cache
	t.cachedWidth = 0
}

// ToolMessageItem represents a tool call with status, input, and output.
type ToolMessageItem struct {
	id           string
	toolName     string
	kind         string // "edit", "execute", "read", etc.
	status       ToolStatus
	input        map[string]any
	output       string
	fileDiff     *FileDiff
	expanded     bool
	maxLines     int // default 10
	cachedRender string
	cachedWidth  int
}

// ID returns the unique identifier for this tool message.
func (t *ToolMessageItem) ID() string {
	return t.id
}

// Render renders the tool message at the given width.
// Shows header with status icon, tool name, and formatted params.
// Shows output body capped at maxLines (or all if expanded).
// Code output (tools with filePath) uses syntax highlighting.
func (t *ToolMessageItem) Render(width int) string {
	// Return cached render if width matches
	if t.cachedWidth == width && t.cachedRender != "" {
		return t.cachedRender
	}

	var result strings.Builder

	// --- HEADER: [icon] [name] [params] ---
	var icon string
	var iconStyle lipgloss.Style
	s := theme.Current().S()
	switch t.status {
	case ToolStatusPending:
		icon = "●"
		iconStyle = s.ToolIconPending
	case ToolStatusRunning:
		icon = "●"
		iconStyle = s.ToolIconPending // Running uses same as pending
	case ToolStatusSuccess:
		icon = "✓"
		iconStyle = s.ToolIconSuccess
	case ToolStatusError:
		icon = "×"
		iconStyle = s.ToolIconError
	case ToolStatusCanceled:
		icon = "×"
		iconStyle = s.ToolIconCanceled
	default:
		icon = "●"
		iconStyle = s.ToolIconPending
	}

	// Build header: [indent] [icon] [name] [params]
	// Capitalize tool name for display
	displayName := t.toolName
	if displayName != "" {
		displayName = strings.ToUpper(displayName[:1]) + displayName[1:]
	}
	header := "  " + iconStyle.Render(icon) + " " + s.ToolName.Render(displayName)

	// Add formatted params if present
	if len(t.input) > 0 {
		// For edit/write tools, only show filePath in header (body shows the rest)
		paramInput := t.input
		if t.kind == "edit" || (t.input["content"] != nil && t.input["filePath"] != nil) {
			paramInput = make(map[string]any)
			if fp, ok := t.input["filePath"]; ok {
				paramInput["filePath"] = fp
			}
		}

		// Reserve space for icon, name, and spacing
		usedWidth := 2 + len(displayName) + 1 // "● " + name + " "
		paramWidth := width - usedWidth
		if paramWidth < 10 {
			paramWidth = 10
		}
		params := formatToolParams(paramInput, paramWidth)
		if params != "" {
			header += " " + s.ToolParams.Render(params)
		}
	}

	result.WriteString(header)

	// --- BODY: output with truncation ---

	// Calculate output width (available width minus indent margin)
	outputWidth := width - 2 // account for MarginLeft(2) on output styles
	if outputWidth < 1 {
		outputWidth = 1
	}

	// Check for Edit tool with diff data
	isEditDiff := false
	if t.kind == "edit" && t.status == ToolStatusSuccess {
		if t.fileDiff != nil && t.fileDiff.Before != "" && t.fileDiff.After != "" {
			// Use full file before/after for proper diff with context
			isEditDiff = true
			result.WriteString("\n\n")
			filePath := t.fileDiff.File
			if fp, ok := t.input["filePath"]; ok {
				filePath = fmt.Sprintf("%v", fp)
			}
			diffRendered := renderDiffBlock(t.fileDiff.Before, t.fileDiff.After, filePath, outputWidth)
			result.WriteString(diffRendered)
			result.WriteString("\n")
		} else if oldStr, hasOld := t.input["oldString"]; hasOld {
			// Fallback: use rawInput oldString/newString
			if newStr, hasNew := t.input["newString"]; hasNew {
				isEditDiff = true
				result.WriteString("\n\n")
				diffRendered := renderDiffBlock(
					fmt.Sprintf("%v", oldStr),
					fmt.Sprintf("%v", newStr),
					"",
					outputWidth,
				)
				result.WriteString(diffRendered)
				result.WriteString("\n")
			}
		}
	}

	// Check for Write file tool with content (write tools also have kind "edit"
	// but use "content" input instead of "oldString"/"newString")
	isWriteFile := false
	if !isEditDiff && t.status == ToolStatusSuccess {
		if content, hasContent := t.input["content"]; hasContent {
			if fp, hasFP := t.input["filePath"]; hasFP {
				isWriteFile = true
				result.WriteString("\n\n")
				fileName := fmt.Sprintf("%v", fp)
				contentStr := fmt.Sprintf("%v", content)

				// Apply truncation like read file blocks
				contentLines := strings.Split(contentStr, "\n")
				totalLines := len(contentLines)
				var hiddenCount int
				if !t.expanded && totalLines > t.maxLines {
					hiddenCount = totalLines - t.maxLines
					contentLines = contentLines[:t.maxLines]
					contentStr = strings.Join(contentLines, "\n")
				}

				var footer string
				if hiddenCount > 0 {
					footer = fmt.Sprintf("…(%d more lines, click to expand)", hiddenCount)
				} else {
					footer = fmt.Sprintf("(End of file - total %d lines)", totalLines)
				}

				writeRendered := renderWriteBlock(contentStr, fileName, footer, outputWidth)
				result.WriteString(writeRendered)
				result.WriteString("\n")
			}
		}
	}

	// Check for diagnostics in output
	hasDiagnostics := false
	if t.output != "" && strings.Contains(t.output, "<diagnostics") {
		diagStart := strings.Index(t.output, "<diagnostics")
		diagEnd := strings.Index(t.output, "</diagnostics>")
		if diagStart != -1 && diagEnd != -1 {
			hasDiagnostics = true
			diagContent := t.output[diagStart : diagEnd+len("</diagnostics>")]
			rendered := renderDiagnostics(diagContent, outputWidth)
			if rendered != "" {
				if !isEditDiff && !isWriteFile {
					result.WriteString("\n\n")
				} else {
					result.WriteString("\n")
				}
				result.WriteString(rendered)
				result.WriteString("\n")
			}
		}
	}

	// Render remaining output if not fully handled by diff/diagnostics/write
	if t.output != "" && !isEditDiff && !isWriteFile && !hasDiagnostics {
		result.WriteString("\n\n") // blank line between header and output

		// Split output into lines
		outputLines := strings.Split(t.output, "\n")

		// Determine visible lines based on expansion state
		var visibleLines []string
		var hiddenCount int

		if t.status == ToolStatusError {
			// Error output: show all lines styled as error
			visibleLines = outputLines
		} else if t.expanded || len(outputLines) <= t.maxLines {
			// Show all lines if expanded or within limit
			visibleLines = outputLines
		} else {
			// Show first maxLines with truncation hint
			visibleLines = outputLines[:t.maxLines]
			hiddenCount = len(outputLines) - t.maxLines
		}

		// Check if this is code output (tool has filePath param)
		isCodeOutput := false
		var fileName string
		if fp, ok := t.input["filePath"]; ok {
			isCodeOutput = true
			fileName = fmt.Sprintf("%v", fp)
		}

		// Render visible lines
		if t.status == ToolStatusError {
			// Error output: red styling with full-width background
			for _, line := range visibleLines {
				result.WriteString(s.ToolError.Width(outputWidth).Render(line))
				result.WriteString("\n")
			}
		} else if isCodeOutput {
			// Code output: properly styled code block with line numbers and syntax highlighting
			codeBlock := strings.Join(visibleLines, "\n")
			rendered := renderCodeBlock(codeBlock, fileName, outputWidth)
			result.WriteString(rendered)
			result.WriteString("\n")
		} else {
			// Plain output: background styling with full-width fill
			for _, line := range visibleLines {
				result.WriteString(s.ToolOutput.Width(outputWidth).Render(line))
				result.WriteString("\n")
			}
		}

		// Add truncation hint if lines were hidden
		if hiddenCount > 0 {
			truncMsg := fmt.Sprintf("…(%d more lines, click to expand)", hiddenCount)
			if isCodeOutput {
				// Code truncation: match code block background
				hint := s.CodeTruncation.Width(outputWidth - 2).Render(truncMsg)
				result.WriteString(hint)
			} else {
				hint := s.ToolTruncation.Width(outputWidth).Render(truncMsg)
				result.WriteString(hint)
			}
			result.WriteString("\n")
		}
	}

	// Cache and return
	rendered := result.String()
	t.cachedRender = rendered
	t.cachedWidth = width
	return rendered
}

// Height returns the number of lines this tool message occupies.
func (t *ToolMessageItem) Height() int {
	if t.cachedRender == "" {
		return 0
	}
	// Count newlines in cached render
	lines := 1
	for _, ch := range t.cachedRender {
		if ch == '\n' {
			lines++
		}
	}
	return lines
}

// IsExpanded returns whether the tool message is expanded.
func (t *ToolMessageItem) IsExpanded() bool {
	return t.expanded
}

// ToggleExpanded toggles the expanded/collapsed state.
func (t *ToolMessageItem) ToggleExpanded() {
	t.expanded = !t.expanded
	// Invalidate cache
	t.cachedWidth = 0
}

// InfoMessageItem represents agent metadata (model, provider, duration).
type InfoMessageItem struct {
	id           string
	model        string
	provider     string
	duration     time.Duration
	cachedRender string
	cachedWidth  int
}

// ID returns the unique identifier for this info message.
func (i *InfoMessageItem) ID() string {
	return i.id
}

// Render renders the info message at the given width.
// Formats as "◇ Model via Provider ⏱ Duration ────────"
func (i *InfoMessageItem) Render(width int) string {
	// Return cached render if width matches
	if i.cachedWidth == width && i.cachedRender != "" {
		return i.cachedRender
	}

	// Format duration as human-readable string
	durationStr := formatDuration(i.duration)

	// Build the info string: ◇ Model via Provider ⏱ Duration
	s := theme.Current().S()
	var infoText string
	if i.model != "" && i.provider != "" {
		infoText = s.InfoIcon.Render("◇") + " " +
			s.InfoModel.Render(i.model) + " " +
			s.InfoProvider.Render("via") + " " +
			s.InfoProvider.Render(i.provider) + " " +
			s.InfoDuration.Render("⏱") + " " +
			s.InfoDuration.Render(durationStr)
	} else if i.model != "" {
		infoText = s.InfoIcon.Render("◇") + " " +
			s.InfoModel.Render(i.model) + " " +
			s.InfoDuration.Render("⏱") + " " +
			s.InfoDuration.Render(durationStr)
	} else {
		infoText = s.InfoIcon.Render("◇") + " " +
			s.InfoDuration.Render("⏱") + " " +
			s.InfoDuration.Render(durationStr)
	}

	// Cache and return
	i.cachedRender = infoText
	i.cachedWidth = width
	return infoText
}

// Height returns the number of lines this info message occupies.
func (i *InfoMessageItem) Height() int {
	if i.cachedRender == "" {
		return 0
	}
	// Count newlines in cached render
	lines := 1
	for _, ch := range i.cachedRender {
		if ch == '\n' {
			lines++
		}
	}
	return lines
}

// SubagentMessageItem represents a subagent task call.
type SubagentMessageItem struct {
	id           string
	subagentType string
	description  string
	status       ToolStatus // Pending, Running, Success, Error
	sessionID    string     // Empty until completion
	spinner      *Spinner   // Spinner for running state
	cachedRender string
	cachedWidth  int
}

// ID returns the unique identifier for this subagent message.
func (s *SubagentMessageItem) ID() string {
	return s.id
}

// Render renders the subagent message at the given width.
// Shows status icon, subagent type in brackets, and description.
// If completed with sessionID, shows "Click to view" hint.
// Uses subtle background to distinguish from regular tool calls.
func (s *SubagentMessageItem) Render(width int) string {
	// Return cached render if width matches
	if s.cachedWidth == width && s.cachedRender != "" {
		return s.cachedRender
	}

	t := theme.Current()
	th := t.S()
	bg := lipgloss.Color(t.BgSurface0)

	// Calculate content width same as tool output blocks:
	// Style has MarginLeft(2), so content width = width - 2
	// The box frame (border + padding) determines the actual content area inside
	contentWidth := width - 2
	if contentWidth < 20 {
		contentWidth = 20
	}

	// --- HEADER: [icon] [type] status ---
	// Create styles with explicit background to prevent ANSI reset from clearing it
	var icon string
	var statusText string

	iconWithBg := func(style lipgloss.Style, char string) string {
		return style.Background(bg).Render(char)
	}

	switch s.status {
	case ToolStatusPending:
		icon = iconWithBg(th.ToolIconPending, "●")
		statusText = "Pending"
	case ToolStatusRunning:
		icon = s.spinner.View()
		statusText = "Running..."
	case ToolStatusSuccess:
		icon = iconWithBg(th.ToolIconSuccess, "●")
		statusText = "Completed"
	case ToolStatusError:
		icon = iconWithBg(th.ToolIconError, "×")
		statusText = "Error"
	case ToolStatusCanceled:
		icon = iconWithBg(th.ToolIconCanceled, "×")
		statusText = "Canceled"
	default:
		icon = iconWithBg(th.ToolIconPending, "●")
		statusText = "Unknown"
	}

	// Build header line with explicit backgrounds on styled parts
	// Style spaces too to prevent black gaps
	space := lipgloss.NewStyle().Background(bg).Render(" ")
	nameStyled := th.ToolName.Background(bg).Render("[" + s.subagentType + "]")
	statusStyled := th.ToolParams.Background(bg).Render(statusText)
	header := icon + space + nameStyled + space + statusStyled

	// --- DESCRIPTION ---
	description := s.description

	// Build content lines
	var content strings.Builder
	content.WriteString(header)
	content.WriteString("\n\n")
	content.WriteString(description)

	// --- "Click to view" HINT (only if sessionID available) ---
	if s.sessionID != "" {
		content.WriteString("\n\n")
		hintStyled := lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.FgSubtle)).
			Background(bg).
			Render("[Click to view]")
		content.WriteString(hintStyled)
	}

	// Apply box styling - MarginLeft(2) is built into the style
	boxed := th.SubagentMessageBox.Width(contentWidth).Render(content.String())

	// Cache and return
	s.cachedRender = boxed
	s.cachedWidth = width
	return s.cachedRender
}

// Height returns the number of lines this subagent message occupies.
func (s *SubagentMessageItem) Height() int {
	if s.cachedRender == "" {
		return 0
	}
	// Count newlines in cached render
	lines := 1
	for _, ch := range s.cachedRender {
		if ch == '\n' {
			lines++
		}
	}
	return lines
}

// DividerMessageItem represents an iteration divider.
type DividerMessageItem struct {
	id           string
	iteration    int
	cachedRender string
	cachedWidth  int
}

// ID returns the unique identifier for this divider message.
func (d *DividerMessageItem) ID() string {
	return d.id
}

// Render renders the divider at the given width.
// Shows centered iteration label with horizontal rules.
func (d *DividerMessageItem) Render(width int) string {
	// Return cached render if width matches
	if d.cachedWidth == width && d.cachedRender != "" {
		return d.cachedRender
	}

	// Create the iteration label
	label := fmt.Sprintf(" Iteration #%d ", d.iteration)
	labelWidth := len(label)

	// Calculate line widths on each side
	lineWidth := (width - labelWidth) / 2
	if lineWidth < 3 {
		lineWidth = 3
	}

	// Build the horizontal rule with centered label
	line := strings.Repeat("─", lineWidth)
	divider := line + label + line

	// Style the divider using theme
	result := theme.Current().S().IterationDivider.Render(divider)

	// Cache and return
	d.cachedRender = result
	d.cachedWidth = width
	return result
}

// Height returns the number of lines this divider occupies.
func (d *DividerMessageItem) Height() int {
	if d.cachedRender == "" {
		return 0
	}
	// Count newlines in cached render
	lines := 1
	for _, ch := range d.cachedRender {
		if ch == '\n' {
			lines++
		}
	}
	return lines
}

// PauseStateMsg signals pause state change to TUI.
type PauseStateMsg struct{ Paused bool }

// AgentBusyMsg signals agent busy state change to TUI.
// Used by status bar to determine PAUSED vs PAUSING display.
type AgentBusyMsg struct{ Busy bool }

// GitInfoMsg carries git repository status for the status bar.
type GitInfoMsg struct {
	Branch string // Branch name or "HEAD" if detached
	Hash   string // Short commit hash (7 chars)
	Dirty  bool   // Uncommitted changes exist
	Ahead  int    // Commits ahead of remote
	Behind int    // Commits behind remote
	Valid  bool   // false if not a git repo
}

// formatDuration formats a duration as a human-readable string.
// Examples: "1.2s", "345ms", "2m30s"
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	if d < time.Minute {
		return d.Round(100 * time.Millisecond).String()
	}
	return d.Round(time.Second).String()
}

// formatToolParams formats tool input parameters for display.
// Shows primary param (command/filePath) first, then remaining params as (key=val, ...).
// Truncates result to fit within maxWidth.
func formatToolParams(input map[string]any, maxWidth int) string {
	if len(input) == 0 {
		return ""
	}

	// Identify primary parameter (command or filePath)
	var primaryKey string
	var primaryVal any
	if cmd, ok := input["command"]; ok {
		primaryKey = "command"
		primaryVal = cmd
	} else if fp, ok := input["filePath"]; ok {
		primaryKey = "filePath"
		primaryVal = fp
	}

	var result strings.Builder

	// Add primary parameter if found
	if primaryKey != "" {
		result.WriteString(fmt.Sprintf("%v", primaryVal))
	}

	// Collect remaining parameters
	var remaining []string
	for key, val := range input {
		if key == primaryKey {
			continue
		}
		remaining = append(remaining, fmt.Sprintf("%s=%v", key, val))
	}

	// Add remaining params in parentheses if any
	if len(remaining) > 0 {
		if result.Len() > 0 {
			result.WriteString(" ")
		}
		result.WriteString("(")
		result.WriteString(strings.Join(remaining, ", "))
		result.WriteString(")")
	}

	// Truncate if necessary
	str := result.String()
	if len(str) > maxWidth {
		if maxWidth > 3 {
			return str[:maxWidth-3] + "..."
		}
		return str[:maxWidth]
	}

	return str
}

// renderCodeBlock parses tool output with line numbers and renders it as a
// properly styled code block with separate line number gutter and syntax-highlighted
// code content with full-width background fill.
//
// Expected input format:
//
//	<file>
//	00001| code line
//	00002| code line
//	</file>
func renderCodeBlock(content, fileName string, width int) string {
	// Strip <file> and </file> tags
	content = strings.TrimPrefix(content, "<file>")
	content = strings.TrimSuffix(content, "</file>")
	content = strings.TrimPrefix(content, "\n")
	content = strings.TrimSuffix(content, "\n")

	lines := strings.Split(content, "\n")

	// Parse line numbers and code content separately
	type codeLine struct {
		lineNum string
		code    string
	}
	var parsed []codeLine
	maxNumWidth := 0

	for _, line := range lines {
		// Try to parse "NNNNN| code" format (line number + pipe + content)
		if idx := strings.Index(line, "| "); idx > 0 && idx <= 7 {
			_ = strings.TrimLeft(line[:idx], "0 ")
			parsed = append(parsed, codeLine{
				lineNum: line[:idx],
				code:    line[idx+2:], // skip "| "
			})
			if len(line[:idx]) > maxNumWidth {
				maxNumWidth = len(line[:idx])
			}
		} else if idx := strings.Index(line, "|"); idx > 0 && idx <= 7 {
			// Handle "NNNNN|" with no space (empty lines)
			parsed = append(parsed, codeLine{
				lineNum: line[:idx],
				code:    line[idx+1:],
			})
			if len(line[:idx]) > maxNumWidth {
				maxNumWidth = len(line[:idx])
			}
		} else {
			// No line number format, use as-is
			parsed = append(parsed, codeLine{code: line})
		}
	}

	if len(parsed) == 0 {
		return theme.Current().S().ToolOutput.Width(width).Render(content)
	}

	// Extract just the code for syntax highlighting
	var codeOnly []string
	for _, p := range parsed {
		codeOnly = append(codeOnly, p.code)
	}
	highlighted := syntaxHighlight(strings.Join(codeOnly, "\n"), fileName)
	highlightedLines := strings.Split(highlighted, "\n")

	// Calculate widths (subtract 2 for the left indent)
	gutterWidth := maxNumWidth + 2 // line number + padding
	codeWidth := width - gutterWidth - 2
	if codeWidth < 10 {
		codeWidth = 10
	}

	// Render each line: [indent] [gutter] [code with bg]
	const codeIndent = "  " // 2-char indent to align with tool header
	s := theme.Current().S()
	var result []string
	for i, p := range parsed {
		// Style the line number gutter, with leading zeros hidden
		var gutterContent string
		if p.lineNum != "" {
			trimmed := strings.TrimLeft(p.lineNum, "0")
			if trimmed == "" {
				trimmed = "0"
			}
			leadingZeros := p.lineNum[:len(p.lineNum)-len(trimmed)]
			if leadingZeros != "" {
				gutterContent = s.CodeLineNumZero.Render(leadingZeros) + trimmed
			} else {
				gutterContent = trimmed
			}
		}
		gutter := s.CodeLineNum.
			Width(gutterWidth).
			Render(gutterContent)

		// Get the highlighted code line (or fallback to raw)
		var codePart string
		if i < len(highlightedLines) {
			codePart = highlightedLines[i]
		} else {
			codePart = p.code
		}

		// Apply code background with full-width fill
		styledCode := s.CodeContent.
			Width(codeWidth).
			Render(codePart)

		result = append(result, codeIndent+lipgloss.JoinHorizontal(lipgloss.Top, gutter, styledCode))
	}

	return strings.Join(result, "\n")
}

// renderWriteBlock renders raw file content as a code block with line numbers
// and syntax highlighting, using green-tinted background (matching diff insert style).
func renderWriteBlock(content, fileName, footer string, width int) string {
	lines := strings.Split(content, "\n")

	// Generate line numbers with zero-padded width
	numDigits := len(fmt.Sprintf("%d", len(lines)))
	if numDigits < 5 {
		numDigits = 5
	}

	// Apply syntax highlighting
	highlighted := syntaxHighlight(content, fileName)
	highlightedLines := strings.Split(highlighted, "\n")

	// Calculate widths (subtract 2 for the left indent)
	gutterWidth := numDigits + 2 // line number + padding
	codeWidth := width - gutterWidth - 2
	if codeWidth < 10 {
		codeWidth = 10
	}

	// Render each line: [indent] [gutter] [code with bg]
	const codeIndent = "  " // 2-char indent to align with tool header
	s := theme.Current().S()
	var result []string
	for i, line := range lines {
		// Format line number with leading zeros
		numStr := fmt.Sprintf("%0*d", numDigits, i+1)
		trimmed := strings.TrimLeft(numStr, "0")
		if trimmed == "" {
			trimmed = "0"
		}
		leadingZeros := numStr[:len(numStr)-len(trimmed)]

		var gutterContent string
		if leadingZeros != "" {
			gutterContent = s.WriteLineNumZero.Render(leadingZeros) + trimmed
		} else {
			gutterContent = trimmed
		}
		gutter := s.WriteLineNum.
			Width(gutterWidth).
			Render(gutterContent)

		// Get the highlighted code line (or fallback to raw)
		var codePart string
		if i < len(highlightedLines) {
			codePart = highlightedLines[i]
		} else {
			codePart = line
		}

		// Apply green background with full-width fill
		styledCode := s.WriteContent.
			Width(codeWidth).
			Render(codePart)

		result = append(result, codeIndent+lipgloss.JoinHorizontal(lipgloss.Top, gutter, styledCode))
	}

	// Render footer line with empty gutter
	if footer != "" {
		emptyGutter := s.WriteLineNum.
			Width(gutterWidth).
			Render("")
		footerContent := s.WriteContent.
			Width(codeWidth).
			Foreground(lipgloss.Color(theme.Current().FgSubtle)).
			Render(footer)
		result = append(result, codeIndent+lipgloss.JoinHorizontal(lipgloss.Top, emptyGutter, footerContent))
	}

	return strings.Join(result, "\n")
}

// splitLine represents one row in a side-by-side diff view.
type splitLine struct {
	beforeNum  int    // 0 means no line on this side
	afterNum   int    // 0 means no line on this side
	beforeText string // content for before side
	afterText  string // content for after side
	beforeKind udiff.OpKind
	afterKind  udiff.OpKind
}

// renderDiffBlock renders a side-by-side diff using go-udiff.
// Shows context lines around changes with proper file line numbers.
func renderDiffBlock(before, after, filePath string, width int) string {
	// Replace tabs with spaces before diff computation for consistent visual width
	before = strings.ReplaceAll(before, "\t", "    ")
	after = strings.ReplaceAll(after, "\t", "    ")

	// Ensure trailing newlines for proper diff computation
	if before != "" && !strings.HasSuffix(before, "\n") {
		before += "\n"
	}
	if after != "" && !strings.HasSuffix(after, "\n") {
		after += "\n"
	}

	// Compute edits
	edits := udiff.Strings(before, after)
	if len(edits) == 0 {
		return "" // No changes
	}

	// Convert to unified diff with context
	unified, err := udiff.ToUnifiedDiff("a", "b", before, edits, 3)
	if err != nil || len(unified.Hunks) == 0 {
		return ""
	}

	// Convert hunks to split lines
	var lines []splitLine
	for _, h := range unified.Hunks {
		beforeLine := h.FromLine
		afterLine := h.ToLine

		// Add hunk separator if not first hunk
		if len(lines) > 0 {
			lines = append(lines, splitLine{beforeKind: -1, afterKind: -1}) // sentinel for separator
		}

		// Process hunk lines, pairing deletes with inserts
		i := 0
		for i < len(h.Lines) {
			l := h.Lines[i]
			switch l.Kind {
			case udiff.Equal:
				lines = append(lines, splitLine{
					beforeNum:  beforeLine,
					afterNum:   afterLine,
					beforeText: l.Content,
					afterText:  l.Content,
					beforeKind: udiff.Equal,
					afterKind:  udiff.Equal,
				})
				beforeLine++
				afterLine++
				i++

			case udiff.Delete:
				// Collect consecutive deletes
				var deletes []udiff.Line
				for i < len(h.Lines) && h.Lines[i].Kind == udiff.Delete {
					deletes = append(deletes, h.Lines[i])
					i++
				}
				// Collect consecutive inserts that follow
				var inserts []udiff.Line
				for i < len(h.Lines) && h.Lines[i].Kind == udiff.Insert {
					inserts = append(inserts, h.Lines[i])
					i++
				}
				// Pair them side-by-side
				maxPairs := len(deletes)
				if len(inserts) > maxPairs {
					maxPairs = len(inserts)
				}
				for j := 0; j < maxPairs; j++ {
					sl := splitLine{}
					if j < len(deletes) {
						sl.beforeNum = beforeLine
						sl.beforeText = deletes[j].Content
						sl.beforeKind = udiff.Delete
						beforeLine++
					}
					if j < len(inserts) {
						sl.afterNum = afterLine
						sl.afterText = inserts[j].Content
						sl.afterKind = udiff.Insert
						afterLine++
					}
					lines = append(lines, sl)
				}

			case udiff.Insert:
				lines = append(lines, splitLine{
					afterNum:  afterLine,
					afterText: l.Content,
					afterKind: udiff.Insert,
				})
				afterLine++
				i++
			}
		}
	}

	// Calculate layout widths
	const indent = "  "
	availableWidth := width - 2            // subtract indent
	panelWidth := (availableWidth - 3) / 2 // -3 for " │ " divider
	if panelWidth < 20 {
		panelWidth = 20
	}

	// Calculate gutter width from max line numbers
	maxLineNum := 1
	for _, l := range lines {
		if l.beforeNum > maxLineNum {
			maxLineNum = l.beforeNum
		}
		if l.afterNum > maxLineNum {
			maxLineNum = l.afterNum
		}
	}
	gutterWidth := len(fmt.Sprintf("%d", maxLineNum))
	if gutterWidth < 3 {
		gutterWidth = 3
	}
	// contentWidth is the space after the gutter and symbol (e.g. " 14 - ")
	contentWidth := panelWidth - gutterWidth - 4 // " NN" (gutter+1) + " - " (3) = gutterWidth+4
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Render each line
	s := theme.Current().S()
	var result []string
	for _, sl := range lines {
		// Separator line between hunks
		if sl.beforeKind == -1 {
			sep := indent + s.DiffDivider.Render(padRight("···", panelWidth)) +
				" " + s.DiffDivider.Render("│") + " " +
				s.DiffDivider.Render(padRight("···", panelWidth))
			result = append(result, sep)
			continue
		}

		beforeText := strings.TrimRight(sl.beforeText, "\n")
		afterText := strings.TrimRight(sl.afterText, "\n")

		// Left panel (before)
		var left string
		switch {
		case sl.beforeNum > 0 && sl.beforeKind == udiff.Delete:
			gutter := fmt.Sprintf(" %*d", gutterWidth, sl.beforeNum)
			code := padRight(truncateLine(beforeText, contentWidth), contentWidth)
			left = s.DiffLineNumDelete.Render(gutter) +
				s.DiffContentDelete.Render(" - "+code)
		case sl.beforeNum > 0 && sl.beforeKind == udiff.Equal:
			gutter := fmt.Sprintf(" %*d", gutterWidth, sl.beforeNum)
			code := padRight(truncateLine(beforeText, contentWidth), contentWidth)
			left = s.DiffLineNumEqual.Render(gutter) +
				s.DiffContentEqual.Render("   "+code)
		default:
			left = s.DiffLineNumMissing.Render(padRight("", gutterWidth+1)) +
				s.DiffContentMissing.Render(padRight("", contentWidth+3))
		}

		// Right panel (after)
		var right string
		switch {
		case sl.afterNum > 0 && sl.afterKind == udiff.Insert:
			gutter := fmt.Sprintf(" %*d", gutterWidth, sl.afterNum)
			code := padRight(truncateLine(afterText, contentWidth), contentWidth)
			right = s.DiffLineNumInsert.Render(gutter) +
				s.DiffContentInsert.Render(" + "+code)
		case sl.afterNum > 0 && sl.afterKind == udiff.Equal:
			gutter := fmt.Sprintf(" %*d", gutterWidth, sl.afterNum)
			code := padRight(truncateLine(afterText, contentWidth), contentWidth)
			right = s.DiffLineNumEqual.Render(gutter) +
				s.DiffContentEqual.Render("   "+code)
		default:
			right = s.DiffLineNumMissing.Render(padRight("", gutterWidth+1)) +
				s.DiffContentMissing.Render(padRight("", contentWidth+3))
		}

		row := indent + left + " " + s.DiffDivider.Render("│") + " " + right
		result = append(result, row)
	}

	return strings.Join(result, "\n")
}

// padRight pads a string with spaces to reach the target width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// renderDiagnostics parses <diagnostics> tagged output and renders it as
// a nicely formatted list of errors/warnings with file path, position, and message.
//
// Expected format:
//
//	<diagnostics file="/path/to/file.go">
//	ERROR [line:col] message
//	WARNING [line:col] message
//	</diagnostics>
func renderDiagnostics(output string, width int) string {
	var result strings.Builder

	// Parse file path from the opening tag
	fileStart := strings.Index(output, `file="`)
	filePath := ""
	if fileStart != -1 {
		fileStart += len(`file="`)
		fileEnd := strings.Index(output[fileStart:], `"`)
		if fileEnd != -1 {
			filePath = output[fileStart : fileStart+fileEnd]
		}
	}

	// Extract content between tags
	contentStart := strings.Index(output, ">")
	contentEnd := strings.Index(output, "</diagnostics>")
	if contentStart == -1 || contentEnd == -1 || contentStart+1 > contentEnd {
		return ""
	}
	content := strings.TrimSpace(output[contentStart+1 : contentEnd])
	if content == "" {
		return ""
	}

	// Render file path header
	s := theme.Current().S()
	if filePath != "" {
		result.WriteString(s.DiagFile.Render(filePath))
		result.WriteString("\n")
	}

	// Parse and render each diagnostic line
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse: "ERROR [line:col] message" or "WARNING [line:col] message"
		var severity, position, message string

		if strings.HasPrefix(line, "ERROR") {
			severity = "ERROR"
			line = strings.TrimPrefix(line, "ERROR")
		} else if strings.HasPrefix(line, "WARNING") {
			severity = "WARNING"
			line = strings.TrimPrefix(line, "WARNING")
		} else {
			// Unknown format, render as-is
			result.WriteString("    " + s.DiagMessage.Render(line) + "\n")
			continue
		}

		line = strings.TrimSpace(line)

		// Extract [line:col]
		if strings.HasPrefix(line, "[") {
			end := strings.Index(line, "]")
			if end != -1 {
				position = line[1:end]
				message = strings.TrimSpace(line[end+1:])
			} else {
				message = line
			}
		} else {
			message = line
		}

		// Render: "    icon SEVERITY [pos] message"
		var icon string
		var sevStyle lipgloss.Style
		if severity == "ERROR" {
			icon = "×"
			sevStyle = s.DiagError
		} else {
			icon = "!"
			sevStyle = s.DiagWarning
		}

		diagLine := "    " + sevStyle.Render(icon+" "+severity)
		if position != "" {
			diagLine += " " + s.DiagPosition.Render("["+position+"]")
		}
		if message != "" {
			diagLine += " " + s.DiagMessage.Render(message)
		}

		result.WriteString(diagLine + "\n")
	}

	return strings.TrimRight(result.String(), "\n")
}

// truncateLine truncates a line to fit within maxWidth, adding ellipsis if needed.
func truncateLine(line string, maxWidth int) string {
	if len(line) <= maxWidth {
		return line
	}
	if maxWidth > 3 {
		return line[:maxWidth-1] + "…"
	}
	return line[:maxWidth]
}

// rightAlign right-aligns a styled block of text to the given width.
// The block may contain ANSI escape codes, so we use lipgloss.Width for accurate measurement.
func rightAlign(content string, width int) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Get the actual visual width (accounts for ANSI codes)
		lineWidth := lipgloss.Width(line)

		// Add padding to right-align
		if lineWidth < width {
			padding := strings.Repeat(" ", width-lineWidth)
			result.WriteString(padding)
		}

		result.WriteString(line)
	}

	return result.String()
}

// syntaxHighlight applies syntax highlighting to source code and returns
// a string with ANSI color codes for terminal display.
//
// It uses the fileName to detect the language, falling back to content analysis,
// and finally to a plain text lexer. The output uses true color (24-bit) ANSI codes.
// Only foreground colors are emitted; backgrounds are cleared so lipgloss can handle them.
func syntaxHighlight(source, fileName string) string {
	// Try to detect lexer from filename
	lexer := lexers.Match(fileName)
	if lexer == nil {
		// Fall back to content-based detection
		lexer = lexers.Analyse(source)
	}
	if lexer == nil {
		// Fall back to plain text
		lexer = lexers.Fallback
	}

	// Use terminal16m formatter for true color output
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		// Fallback to terminal256 if terminal16m is not available
		formatter = formatters.Get("terminal256")
	}
	if formatter == nil {
		// Last resort: return source unchanged with background style
		return theme.Current().S().ToolOutput.Render(source)
	}

	// Use catppuccin-mocha style (matches our UI color palette)
	baseStyle := styles.Get("catppuccin-mocha")
	if baseStyle == nil {
		baseStyle = styles.Fallback
	}

	// Clear all token backgrounds so the formatter emits only foreground ANSI codes.
	// The containing block's lipgloss style handles the background fill uniformly.
	style, err := baseStyle.Builder().Transform(func(entry chroma.StyleEntry) chroma.StyleEntry {
		entry.Background = 0
		return entry
	}).Build()
	if err != nil {
		style = baseStyle
	}

	// Tokenize and format
	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return theme.Current().S().ToolOutput.Render(source)
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return theme.Current().S().ToolOutput.Render(source)
	}

	// Replace full ANSI resets (\x1b[0m) with foreground-only resets so they
	// don't clear the background color set by the containing lipgloss style.
	// Resets: 39=default fg, 22=no bold, 23=no italic, 24=no underline.
	result := strings.ReplaceAll(buf.String(), "\x1b[0m", "\x1b[39;22;23;24m")
	return strings.TrimRight(result, "\n")
}
