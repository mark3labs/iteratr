package agent

import "time"

// FileDiff contains before/after file content from an edit tool call.
type FileDiff struct {
	File      string // Absolute file path
	Before    string // Full file content before edit
	After     string // Full file content after edit
	Additions int    // Number of added lines
	Deletions int    // Number of deleted lines
}

// ToolCallEvent represents a tool lifecycle event from ACP.
// Used to track tool calls from pending → in_progress → completed/error/canceled.
type ToolCallEvent struct {
	ToolCallID string         // Stable ID for tracking updates
	Title      string         // Tool name (e.g., "bash")
	Status     string         // "pending", "in_progress", "completed", "error", "canceled"
	RawInput   map[string]any // Command params (populated on in_progress+)
	Output     string         // Tool output (populated on completed/error)
	Kind       string         // "execute", etc.
	FileDiff   *FileDiff      // File diff data (populated on completed edit tools)
}

// FinishEvent represents the completion of an agent iteration.
// Emitted when prompt() returns, either successfully or with an error.
type FinishEvent struct {
	StopReason string        // "end_turn", "max_tokens", "cancelled", "refusal", "max_turn_requests", "error"
	Error      string        // Error message if StopReason is "error"
	Duration   time.Duration // Time taken for the iteration
	Model      string        // Model used (e.g., "anthropic/claude-sonnet-4-5")
	Provider   string        // Provider extracted from model (e.g., "Anthropic")
}
