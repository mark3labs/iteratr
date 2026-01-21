package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/coder/acp-go-sdk"
	"github.com/mark3labs/iteratr/internal/session"
)

// ACPClient implements acp.Client interface for handling agent requests
type ACPClient struct {
	store            *session.Store
	sessionName      string
	workDir          string
	conn             *acp.ClientSideConnection
	currentSessionID acp.SessionId
	sessionComplete  bool

	// Callback for sending updates to TUI
	onOutput   func(string)
	onToolCall func(string, string) // tool name, status
}

var _ acp.Client = (*ACPClient)(nil)

// NewACPClient creates a new ACP client for managing agent interactions
func NewACPClient(store *session.Store, sessionName, workDir string) *ACPClient {
	return &ACPClient{
		store:       store,
		sessionName: sessionName,
		workDir:     workDir,
	}
}

// SetOutputCallback sets a callback for streaming agent output
func (c *ACPClient) SetOutputCallback(fn func(string)) {
	c.onOutput = fn
}

// SetToolCallCallback sets a callback for tool call updates
func (c *ACPClient) SetToolCallCallback(fn func(string, string)) {
	c.onToolCall = fn
}

// IsSessionComplete returns true if the agent called session_complete
func (c *ACPClient) IsSessionComplete() bool {
	return c.sessionComplete
}

// ReadTextFile implements acp.Client - handles file read requests from agent
func (c *ACPClient) ReadTextFile(ctx context.Context, params acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	content, err := os.ReadFile(params.Path)
	if err != nil {
		return acp.ReadTextFileResponse{}, fmt.Errorf("failed to read file: %w", err)
	}
	return acp.ReadTextFileResponse{Content: string(content)}, nil
}

// WriteTextFile implements acp.Client - handles file write requests from agent
func (c *ACPClient) WriteTextFile(ctx context.Context, params acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	err := os.WriteFile(params.Path, []byte(params.Content), 0o644)
	if err != nil {
		return acp.WriteTextFileResponse{}, fmt.Errorf("failed to write file: %w", err)
	}
	return acp.WriteTextFileResponse{}, nil
}

// SessionUpdate implements acp.Client - handles streaming updates from agent
func (c *ACPClient) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	u := params.Update

	switch {
	case u.AgentMessageChunk != nil:
		// Stream agent text output to TUI
		if u.AgentMessageChunk.Content.Text != nil && c.onOutput != nil {
			c.onOutput(u.AgentMessageChunk.Content.Text.Text)
		}

	case u.ToolCall != nil:
		// Tool call initiated - check if it's one of our custom tools
		return c.handleToolCall(ctx, u.ToolCall)

	case u.ToolCallUpdate != nil:
		// Tool call completed
		if c.onToolCall != nil {
			c.onToolCall(string(u.ToolCallUpdate.ToolCallId), string(*u.ToolCallUpdate.Status))
		}
	}

	return nil
}

// RequestPermission implements acp.Client - handles permission requests
func (c *ACPClient) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	// For now, auto-approve all requests
	// In the future, this could prompt the user via TUI
	if len(params.Options) == 0 {
		return acp.RequestPermissionResponse{
			Outcome: acp.RequestPermissionOutcome{
				Cancelled: &acp.RequestPermissionOutcomeCancelled{
					Outcome: "cancelled",
				},
			},
		}, nil
	}

	return acp.RequestPermissionResponse{
		Outcome: acp.RequestPermissionOutcome{
			Selected: &acp.RequestPermissionOutcomeSelected{
				OptionId: params.Options[0].OptionId,
				Outcome:  "selected",
			},
		},
	}, nil
}

// CreateTerminal implements acp.Client - creates a terminal for command execution
func (c *ACPClient) CreateTerminal(ctx context.Context, params acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	// For now, we don't support terminal creation
	return acp.CreateTerminalResponse{}, fmt.Errorf("terminal support not implemented")
}

// KillTerminalCommand implements acp.Client - kills a running terminal command
func (c *ACPClient) KillTerminalCommand(ctx context.Context, params acp.KillTerminalCommandRequest) (acp.KillTerminalCommandResponse, error) {
	return acp.KillTerminalCommandResponse{}, fmt.Errorf("terminal support not implemented")
}

// TerminalOutput implements acp.Client - gets terminal output and status
func (c *ACPClient) TerminalOutput(ctx context.Context, params acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, fmt.Errorf("terminal support not implemented")
}

// ReleaseTerminal implements acp.Client - releases a terminal and frees resources
func (c *ACPClient) ReleaseTerminal(ctx context.Context, params acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, fmt.Errorf("terminal support not implemented")
}

// WaitForTerminalExit implements acp.Client - waits for terminal command to exit
func (c *ACPClient) WaitForTerminalExit(ctx context.Context, params acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, fmt.Errorf("terminal support not implemented")
}

// handleToolCall processes tool calls from the agent and routes them to the session store
// In ACP, the client receives tool calls via SessionUpdate and processes them inline.
// The agent is responsible for managing tool state; we just execute and return errors if any.
func (c *ACPClient) handleToolCall(ctx context.Context, tc *acp.SessionUpdateToolCall) error {
	// Extract tool name from Title or RawInput
	// The agent encodes the tool name in the call
	toolName := tc.Title
	input, ok := tc.RawInput.(map[string]any)
	if !ok {
		input = make(map[string]any)
	}

	var result any
	var err error

	// Route to appropriate session store method based on tool name
	switch toolName {
	case "task_add":
		result, err = c.handleTaskAdd(ctx, input)
	case "task_status":
		result, err = c.handleTaskStatus(ctx, input)
	case "task_list":
		result, err = c.handleTaskList(ctx)
	case "note_add":
		result, err = c.handleNoteAdd(ctx, input)
	case "note_list":
		result, err = c.handleNoteList(ctx, input)
	case "inbox_list":
		result, err = c.handleInboxList(ctx)
	case "inbox_mark_read":
		result, err = c.handleInboxMarkRead(ctx, input)
	case "session_complete":
		result, err = c.handleSessionComplete(ctx)
		c.sessionComplete = true
	default:
		// Not our tool, agent will handle it
		return nil
	}

	// If there's an error, report it
	if err != nil {
		if c.onToolCall != nil {
			c.onToolCall(toolName, "failed: "+err.Error())
		}
		return err
	}

	// Notify success
	if c.onToolCall != nil {
		c.onToolCall(toolName, formatResult(result, nil))
	}

	return nil
}

// formatResult formats a tool result for display to the agent
func formatResult(result any, err error) string {
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if result == nil {
		return "OK"
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b)
}

// RunIteration launches the agent subprocess and runs a single iteration
func (c *ACPClient) RunIteration(ctx context.Context, prompt string) error {
	// Start opencode as subprocess
	cmd := exec.CommandContext(ctx, "opencode", "acp")
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}

	// Create client-side connection
	c.conn = acp.NewClientSideConnection(c, stdin, stdout)

	// Initialize connection
	_, err = c.conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapability{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	// Create session
	newSess, err := c.conn.NewSession(ctx, acp.NewSessionRequest{
		Cwd: c.workDir,
	})
	if err != nil {
		return fmt.Errorf("new session failed: %w", err)
	}

	c.currentSessionID = newSess.SessionId

	// Send prompt and stream response
	_, err = c.conn.Prompt(ctx, acp.PromptRequest{
		SessionId: newSess.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock(prompt)},
	})
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Wait for agent process to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("agent process failed: %w", err)
	}

	return nil
}
