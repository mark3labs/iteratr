package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/mark3labs/iteratr/internal/logger"
)

// acpConn wraps stdin/stdout pipes for bidirectional JSON-RPC 2.0 communication.
type acpConn struct {
	stdin   io.WriteCloser
	stdout  io.Reader
	reader  *bufio.Reader
	encoder *json.Encoder
	reqID   atomic.Int32
}

// newACPConn creates a new ACP connection wrapping the given pipes.
func newACPConn(stdin io.WriteCloser, stdout io.Reader) *acpConn {
	return &acpConn{
		stdin:   stdin,
		stdout:  stdout,
		reader:  bufio.NewReader(stdout),
		encoder: json.NewEncoder(stdin),
	}
}

// sendResponse sends a JSON-RPC 2.0 response to a server-initiated request.
func (c *acpConn) sendResponse(id int, result any) error {
	resp := jsonRPCResponseOut{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	logger.Debug("ACP response [%d]", id)
	if err := c.encoder.Encode(resp); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}
	return nil
}

// sendRequest sends a JSON-RPC 2.0 request and returns the assigned request ID.
func (c *acpConn) sendRequest(method string, params any) (int, error) {
	id := int(c.reqID.Add(1))
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	logger.Debug("ACP request [%d]: %s", id, method)
	if err := c.encoder.Encode(req); err != nil {
		return 0, fmt.Errorf("failed to encode request: %w", err)
	}
	return id, nil
}

// readMessage reads one JSON-RPC message from stdout.
// Returns nil if EOF is reached.
func (c *acpConn) readMessage() (*jsonRPCResponse, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		logger.Warn("Failed to parse ACP message: %v | raw: %s", err, line)
		return nil, fmt.Errorf("failed to parse JSON-RPC message: %w", err)
	}

	return &resp, nil
}

// close signals EOF to the subprocess by closing stdin.
func (c *acpConn) close() error {
	logger.Debug("Closing ACP connection stdin")
	return c.stdin.Close()
}

// initialize sends the initialize request and validates the agent response.
func (c *acpConn) initialize(ctx context.Context) error {
	params := initializeParams{
		ProtocolVersion: 1,
		ClientCapabilities: clientCapabilities{
			Fs: fsCapability{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
		},
	}

	reqID, err := c.sendRequest("initialize", params)
	if err != nil {
		return fmt.Errorf("failed to send initialize request: %w", err)
	}

	// Read response
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := c.readMessage()
		if err != nil {
			return fmt.Errorf("failed to read initialize response: %w", err)
		}

		// Skip notifications
		if resp.ID == nil {
			continue
		}

		// Check if this is our response
		if *resp.ID != reqID {
			continue
		}

		// Handle error response
		if resp.Error != nil {
			return fmt.Errorf("initialize failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
		}

		// Parse result
		var result initializeResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return fmt.Errorf("failed to parse initialize result: %w", err)
		}

		// Validate agentInfo is present
		if result.AgentInfo == nil {
			return fmt.Errorf("initialize response missing agentInfo")
		}

		logger.Debug("ACP initialized: %s v%s", result.AgentInfo.Name, result.AgentInfo.Version)
		return nil
	}
}

// newSession creates a new ACP session and returns the session ID.
func (c *acpConn) newSession(ctx context.Context, cwd string) (string, error) {
	params := newSessionParams{
		Cwd:        cwd,
		McpServers: []any{},
	}

	reqID, err := c.sendRequest("session/new", params)
	if err != nil {
		return "", fmt.Errorf("failed to send session/new request: %w", err)
	}

	// Read response (skip notifications)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		resp, err := c.readMessage()
		if err != nil {
			return "", fmt.Errorf("failed to read session/new response: %w", err)
		}

		// Skip notifications
		if resp.ID == nil {
			continue
		}

		// Check if this is our response
		if *resp.ID != reqID {
			continue
		}

		// Handle error response
		if resp.Error != nil {
			return "", fmt.Errorf("session/new failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
		}

		// Parse result
		var result newSessionResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return "", fmt.Errorf("failed to parse session/new result: %w", err)
		}

		if result.SessionID == "" {
			return "", fmt.Errorf("session/new response missing sessionId")
		}

		logger.Debug("ACP session created: %s", result.SessionID)
		return result.SessionID, nil
	}
}

// setModel sets the model for the given session.
func (c *acpConn) setModel(ctx context.Context, sessionID, modelID string) error {
	params := setModelParams{
		SessionID: sessionID,
		ModelID:   modelID,
	}

	reqID, err := c.sendRequest("session/set_model", params)
	if err != nil {
		return fmt.Errorf("failed to send session/set_model request: %w", err)
	}

	// Read response (skip notifications)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := c.readMessage()
		if err != nil {
			return fmt.Errorf("failed to read session/set_model response: %w", err)
		}

		// Skip notifications
		if resp.ID == nil {
			continue
		}

		// Check if this is our response
		if *resp.ID != reqID {
			continue
		}

		// Handle error response
		if resp.Error != nil {
			return fmt.Errorf("session/set_model failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
		}

		logger.Debug("ACP model set: %s", modelID)
		return nil
	}
}

// prompt sends a prompt to the session and streams notifications via callbacks.
// Accepts multiple text blocks which are sent as separate content blocks in the same request.
// Returns the stop reason when the prompt completes or an error occurs.
func (c *acpConn) prompt(ctx context.Context, sessionID string, texts []string, onText func(string), onToolCall func(ToolCallEvent), onThinking func(string), onFileChange func(FileChange)) (string, error) {
	// Build content blocks from texts
	blocks := make([]contentBlock, 0, len(texts))
	for _, text := range texts {
		if text != "" {
			blocks = append(blocks, contentBlock{
				Type: "text",
				Text: text,
			})
		}
	}

	params := promptParams{
		SessionID: sessionID,
		Prompt:    blocks,
	}

	reqID, err := c.sendRequest("session/prompt", params)
	if err != nil {
		return "", fmt.Errorf("failed to send session/prompt request: %w", err)
	}

	// Read messages in loop until response with matching request ID arrives
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		resp, err := c.readMessage()
		if err != nil {
			return "", fmt.Errorf("failed to read prompt response: %w", err)
		}

		// For notifications (id==nil, method=="session/update"): parse update params
		if resp.ID == nil && resp.Method == "session/update" {
			var updateParams sessionUpdateParams
			if err := json.Unmarshal(resp.Params, &updateParams); err != nil {
				logger.Warn("Failed to parse session/update params: %v", err)
				continue
			}

			// Discriminate by sessionUpdate field
			var update sessionUpdate
			if err := json.Unmarshal(updateParams.Update, &update); err != nil {
				logger.Warn("Failed to parse session update: %v", err)
				continue
			}

			switch update.SessionUpdate {
			case "agent_message_chunk":
				// agent_message_chunk: extract update.content.text, call onText
				var chunk agentMessageChunk
				if err := json.Unmarshal(updateParams.Update, &chunk); err != nil {
					logger.Warn("Failed to parse agent_message_chunk: %v", err)
					continue
				}
				if onText != nil {
					onText(chunk.Content.Text)
				}

			case "agent_thought_chunk":
				// agent_thought_chunk: extract update.content.text, call onThinking
				var chunk agentThoughtChunk
				if err := json.Unmarshal(updateParams.Update, &chunk); err != nil {
					logger.Warn("Failed to parse agent_thought_chunk: %v", err)
					continue
				}
				if onThinking != nil {
					onThinking(chunk.Content.Text)
				}

			case "tool_call":
				// tool_call: build ToolCallEvent{Status:"pending", ToolCallID, Title, Kind}, call onToolCall
				var tc toolCall
				if err := json.Unmarshal(updateParams.Update, &tc); err != nil {
					logger.Warn("Failed to parse tool_call: %v", err)
					continue
				}
				if onToolCall != nil {
					onToolCall(ToolCallEvent{
						ToolCallID: tc.ToolCallID,
						Title:      tc.Title,
						Status:     tc.Status,
						Kind:       tc.Kind,
						RawInput:   tc.RawInput,
					})
				}

			case "tool_call_update":
				// tool_call_update (in_progress): build event with RawInput from update, call onToolCall
				// tool_call_update (completed): build event with Output from update.content[0].content.text, call onToolCall
				var tcu toolCallUpdate
				if err := json.Unmarshal(updateParams.Update, &tcu); err != nil {
					logger.Warn("Failed to parse tool_call_update: %v", err)
					continue
				}
				if onToolCall != nil {
					event := ToolCallEvent{
						ToolCallID: tcu.ToolCallID,
						Title:      tcu.Title,
						Status:     tcu.Status,
						Kind:       tcu.Kind,
						RawInput:   tcu.RawInput,
					}
					// Extract output from completed or error tool calls
					if (tcu.Status == "completed" || tcu.Status == "error") && len(tcu.Content) > 0 {
						event.Output = tcu.Content[0].Content.Text
					}
					// Extract filediff from rawOutput metadata for edit tools
					if tcu.Status == "completed" && tcu.Kind == "edit" {
						event.FileDiff = extractFileDiff(tcu.RawOutput)
						// Extract diff blocks from content array
						event.DiffBlocks = extractDiffBlocks(tcu.Content)

						// Extract file changes and call onFileChange callback for each
						if onFileChange != nil {
							changes := extractFileChanges(event)
							for _, change := range changes {
								onFileChange(change)
							}
						}
					}
					onToolCall(event)
				}

			case "available_commands_update":
				// available_commands_update: skip
				continue

			default:
				logger.Debug("Unknown session update type: %s", update.SessionUpdate)
			}

			continue
		}

		// Skip other notifications (no id, no method we handle)
		if resp.ID == nil {
			continue
		}

		// Handle server-initiated permission requests (auto-grant)
		if resp.Method == "session/request_permission" {
			var params permissionRequestParams
			if err := json.Unmarshal(resp.Params, &params); err != nil {
				logger.Warn("Failed to parse permission request: %v", err)
				continue
			}
			// Find "always" option, fall back to first allow option
			optionID := ""
			for _, opt := range params.Options {
				if opt.Kind == "allow_always" {
					optionID = opt.OptionID
					break
				}
			}
			if optionID == "" {
				for _, opt := range params.Options {
					if opt.Kind == "allow_once" {
						optionID = opt.OptionID
						break
					}
				}
			}
			if optionID == "" && len(params.Options) > 0 {
				optionID = params.Options[0].OptionID
			}
			logger.Debug("ACP auto-granting permission [%s] tool=%s option=%s", params.ToolCall.Title, params.ToolCall.ToolCallID, optionID)
			if err := c.sendResponse(*resp.ID, permissionResponse{
				Outcome: permissionOutcome{OptionID: optionID, Outcome: "selected"},
			}); err != nil {
				logger.Warn("Failed to send permission response: %v", err)
			}
			continue
		}

		// Check if this is our response
		if *resp.ID != reqID {
			continue
		}

		// Handle error response
		if resp.Error != nil {
			return "", fmt.Errorf("session/prompt failed: %s (code %d)", resp.Error.Message, resp.Error.Code)
		}

		// Parse result to extract stop reason
		var result promptResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			logger.Warn("Failed to parse prompt result: %v", err)
			// Default to "end_turn" if parsing fails
			return "end_turn", nil
		}

		// Return stop reason (e.g., "end_turn", "max_tokens", "cancelled", "refusal", "max_turn_requests")
		stopReason := result.StopReason
		if stopReason == "" {
			stopReason = "end_turn" // Default if not provided
		}
		logger.Debug("ACP prompt completed with stop reason: %s", stopReason)
		return stopReason, nil
	}
}

// JSON-RPC 2.0 message envelope
// extractFileDiff parses rawOutput.metadata.filediff from a completed edit tool call.
func extractFileDiff(rawOutput map[string]any) *FileDiff {
	if rawOutput == nil {
		return nil
	}
	metadata, ok := rawOutput["metadata"].(map[string]any)
	if !ok {
		return nil
	}
	fd, ok := metadata["filediff"].(map[string]any)
	if !ok {
		return nil
	}

	result := &FileDiff{}
	if f, ok := fd["file"].(string); ok {
		result.File = f
	}
	if b, ok := fd["before"].(string); ok {
		result.Before = b
	}
	if a, ok := fd["after"].(string); ok {
		result.After = a
	}
	if a, ok := fd["additions"].(float64); ok {
		result.Additions = int(a)
	}
	if d, ok := fd["deletions"].(float64); ok {
		result.Deletions = int(d)
	}
	return result
}

// extractDiffBlocks parses diff blocks from tool_call_update content array.
// Returns diff blocks with path, oldText, newText extracted from type:"diff" blocks.
func extractDiffBlocks(content []toolCallContent) []DiffBlock {
	var blocks []DiffBlock
	for _, c := range content {
		if c.Type == "diff" {
			blocks = append(blocks, DiffBlock{
				Path:    c.Path,
				OldText: c.OldText,
				NewText: c.NewText,
			})
		}
	}
	return blocks
}

// extractFileChanges extracts file changes from a ToolCallEvent.
// Primary source: DiffBlocks from content array (provides path, isNew)
// Secondary source: FileDiff metadata (provides additions/deletions for edit tools)
// Fallback: RawInput.filePath if no diff blocks present
//
// Returns a slice of FileChange with:
// - Path: absolute file path
// - IsNew: true if oldText was empty (file was created)
// - Additions/Deletions: from metadata if available, 0 otherwise
func extractFileChanges(event ToolCallEvent) []FileChange {
	var changes []FileChange

	// Primary: Extract from diff blocks (present in both write and edit tools)
	if len(event.DiffBlocks) > 0 {
		for _, block := range event.DiffBlocks {
			change := FileChange{
				AbsPath: block.Path,
				IsNew:   block.OldText == "",
			}

			// Secondary: Merge additions/deletions from metadata if available
			// (only present in edit tools, not write tools)
			if event.FileDiff != nil && event.FileDiff.File == block.Path {
				change.Additions = event.FileDiff.Additions
				change.Deletions = event.FileDiff.Deletions
			}

			changes = append(changes, change)
		}
		return changes
	}

	// Fallback: Extract path from metadata or rawInput if no diff blocks
	var path string
	if event.FileDiff != nil && event.FileDiff.File != "" {
		path = event.FileDiff.File
	} else if fp, ok := event.RawInput["filePath"].(string); ok {
		path = fp
	}

	if path != "" {
		change := FileChange{
			AbsPath: path,
			IsNew:   false, // Unknown without diff blocks
		}
		if event.FileDiff != nil {
			change.Additions = event.FileDiff.Additions
			change.Deletions = event.FileDiff.Deletions
		}
		changes = append(changes, change)
	}

	return changes
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`     // nil for notifications
	Method  string          `json:"method,omitempty"` // set for notifications
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"` // for notifications
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ACP-specific request/response types
type initializeParams struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities clientCapabilities `json:"clientCapabilities"`
}

type clientCapabilities struct {
	Fs fsCapability `json:"fs"`
}

type fsCapability struct {
	ReadTextFile  bool `json:"readTextFile"`
	WriteTextFile bool `json:"writeTextFile"`
}

type initializeResult struct {
	AgentInfo *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"agentInfo,omitempty"`
}

type newSessionParams struct {
	Cwd        string `json:"cwd"`
	McpServers []any  `json:"mcpServers"`
}

type newSessionResult struct {
	SessionID string       `json:"sessionId"`
	Models    *modelsState `json:"models,omitempty"`
}

type modelsState struct {
	CurrentModelID  string      `json:"currentModelId"`
	AvailableModels []modelInfo `json:"availableModels"`
}

type modelInfo struct {
	ModelID string `json:"modelId"`
	Name    string `json:"name"`
}

type setModelParams struct {
	SessionID string `json:"sessionId"`
	ModelID   string `json:"modelId"`
}

type promptParams struct {
	SessionID string         `json:"sessionId"`
	Prompt    []contentBlock `json:"prompt"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type promptResult struct {
	StopReason string `json:"stopReason"`
}

// SessionUpdate notification params
type sessionUpdateParams struct {
	SessionID string          `json:"sessionId"`
	Update    json.RawMessage `json:"update"`
}

// Discriminated by "sessionUpdate" field
type sessionUpdate struct {
	SessionUpdate string `json:"sessionUpdate"` // "agent_message_chunk", "tool_call", "tool_call_update", "available_commands_update"
}

// agent_message_chunk
type agentMessageChunk struct {
	SessionUpdate string      `json:"sessionUpdate"` // "agent_message_chunk"
	Content       contentPart `json:"content"`
}

// agent_thought_chunk
type agentThoughtChunk struct {
	SessionUpdate string      `json:"sessionUpdate"` // "agent_thought_chunk"
	Content       contentPart `json:"content"`
}

type contentPart struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// tool_call (initial, status=pending)
type toolCall struct {
	SessionUpdate string         `json:"sessionUpdate"` // "tool_call"
	ToolCallID    string         `json:"toolCallId"`
	Title         string         `json:"title"`    // tool name, e.g. "bash"
	Kind          string         `json:"kind"`     // "execute"
	Status        string         `json:"status"`   // "pending"
	RawInput      map[string]any `json:"rawInput"` // empty on pending
}

// tool_call_update (in_progress, completed, error, or canceled)
type toolCallUpdate struct {
	SessionUpdate string            `json:"sessionUpdate"` // "tool_call_update"
	ToolCallID    string            `json:"toolCallId"`
	Title         string            `json:"title"`
	Kind          string            `json:"kind"`
	Status        string            `json:"status"`              // "in_progress" | "completed" | "error" | "canceled"
	RawInput      map[string]any    `json:"rawInput"`            // e.g. {"command":"echo hi","description":"..."}
	Content       []toolCallContent `json:"content,omitempty"`   // only on completed/error
	RawOutput     map[string]any    `json:"rawOutput,omitempty"` // only on completed
}

// Nested content in completed tool_call_update
// Handles polymorphic content blocks in tool_call_update
// Can be either: {type:"content", content:{type:"text", text:"..."}}
//
//	or: {type:"diff", path:"...", oldText:"...", newText:"..."}
type toolCallContent struct {
	Type    string      `json:"type"`              // "content" or "diff"
	Content contentPart `json:"content,omitempty"` // for type="content"
	Path    string      `json:"path,omitempty"`    // for type="diff"
	OldText string      `json:"oldText,omitempty"` // for type="diff"
	NewText string      `json:"newText,omitempty"` // for type="diff"
}

// JSON-RPC response sent from client to server (for server-initiated requests)
type jsonRPCResponseOut struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  any    `json:"result"`
}

// Permission request/response types (session/request_permission)
type permissionRequestParams struct {
	SessionID string             `json:"sessionId"`
	ToolCall  permissionTool     `json:"toolCall"`
	Options   []permissionOption `json:"options"`
}

type permissionTool struct {
	ToolCallID string         `json:"toolCallId"`
	Title      string         `json:"title"`
	Kind       string         `json:"kind"`
	Status     string         `json:"status"`
	RawInput   map[string]any `json:"rawInput"`
}

type permissionOption struct {
	OptionID string `json:"optionId"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
}

type permissionResponse struct {
	Outcome permissionOutcome `json:"outcome"`
}

type permissionOutcome struct {
	OptionID string `json:"optionId"`
	Outcome  string `json:"outcome"` // "selected" or "cancelled"
}
