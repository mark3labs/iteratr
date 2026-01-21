package acp

import (
	"context"
	"fmt"

	"github.com/mark3labs/iteratr/internal/session"
)

// Tool handler methods - these route tool calls from the agent to session store operations

// handleTaskAdd handles the task_add tool call
func (c *ACPClient) handleTaskAdd(ctx context.Context, input map[string]any) (*session.Task, error) {
	// Extract parameters
	content, _ := input["content"].(string)
	status, _ := input["status"].(string)
	iteration, _ := input["iteration"].(float64)

	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	params := session.TaskAddParams{
		Content:   content,
		Status:    status,
		Iteration: int(iteration),
	}

	return c.store.TaskAdd(ctx, c.sessionName, params)
}

// handleTaskStatus handles the task_status tool call
func (c *ACPClient) handleTaskStatus(ctx context.Context, input map[string]any) (any, error) {
	// Extract parameters
	id, _ := input["id"].(string)
	status, _ := input["status"].(string)
	iteration, _ := input["iteration"].(float64)

	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if status == "" {
		return nil, fmt.Errorf("status is required")
	}

	params := session.TaskStatusParams{
		ID:        id,
		Status:    status,
		Iteration: int(iteration),
	}

	err := c.store.TaskStatus(ctx, c.sessionName, params)
	if err != nil {
		return nil, err
	}

	return map[string]string{"status": "updated"}, nil
}

// handleTaskList handles the task_list tool call
func (c *ACPClient) handleTaskList(ctx context.Context) (*session.TaskListResult, error) {
	return c.store.TaskList(ctx, c.sessionName)
}

// handleNoteAdd handles the note_add tool call
func (c *ACPClient) handleNoteAdd(ctx context.Context, input map[string]any) (*session.Note, error) {
	// Extract parameters
	content, _ := input["content"].(string)
	noteType, _ := input["type"].(string)
	iteration, _ := input["iteration"].(float64)

	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if noteType == "" {
		return nil, fmt.Errorf("type is required")
	}

	params := session.NoteAddParams{
		Content:   content,
		Type:      noteType,
		Iteration: int(iteration),
	}

	return c.store.NoteAdd(ctx, c.sessionName, params)
}

// handleNoteList handles the note_list tool call
func (c *ACPClient) handleNoteList(ctx context.Context, input map[string]any) ([]*session.Note, error) {
	// Extract optional type filter
	noteType, _ := input["type"].(string)

	params := session.NoteListParams{
		Type: noteType,
	}

	return c.store.NoteList(ctx, c.sessionName, params)
}

// handleInboxList handles the inbox_list tool call
func (c *ACPClient) handleInboxList(ctx context.Context) ([]*session.Message, error) {
	return c.store.InboxList(ctx, c.sessionName)
}

// handleInboxMarkRead handles the inbox_mark_read tool call
func (c *ACPClient) handleInboxMarkRead(ctx context.Context, input map[string]any) (any, error) {
	// Extract parameters
	id, _ := input["id"].(string)

	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	params := session.InboxMarkReadParams{
		ID: id,
	}

	err := c.store.InboxMarkRead(ctx, c.sessionName, params)
	if err != nil {
		return nil, err
	}

	return map[string]string{"status": "marked_read"}, nil
}

// handleSessionComplete handles the session_complete tool call
func (c *ACPClient) handleSessionComplete(ctx context.Context) (any, error) {
	err := c.store.SessionComplete(ctx, c.sessionName)
	if err != nil {
		return nil, err
	}

	return map[string]string{"status": "session_complete"}, nil
}

// ToolDescriptions returns the tool descriptions to be included in the agent prompt
const ToolDescriptions = `
## Available Tools - all require session_name parameter

### Task Management
- task_add(content, status?, iteration) - Create task (status: remaining|in_progress|completed|blocked)
  * content (required): Task description
  * status (optional): Task status, defaults to "remaining"
  * iteration (required): Current iteration number
  
- task_status(id, status, iteration) - Update task status
  * id (required): Task ID or 8+ character prefix
  * status (required): New status (remaining|in_progress|completed|blocked)
  * iteration (required): Current iteration number
  
- task_list() - List all tasks grouped by status
  * Returns: {remaining: [...], in_progress: [...], completed: [...], blocked: [...]}

### Notes
- note_add(content, type, iteration) - Record note
  * content (required): Note content
  * type (required): Note type (learning|stuck|tip|decision)
  * iteration (required): Current iteration number
  
- note_list(type?) - List notes, optionally filtered by type
  * type (optional): Filter by note type

### Inbox
- inbox_list() - Get unread messages from human
  * Returns: Array of unread messages
  
- inbox_mark_read(id) - Acknowledge message after processing
  * id (required): Message ID or 8+ character prefix

### Session Control
- session_complete() - Signal all tasks done, end iteration loop
  * Call this when ALL tasks in the spec are completed
`

// GetToolDescriptions returns formatted tool descriptions for the prompt
func GetToolDescriptions(sessionName string) string {
	return fmt.Sprintf(`
## Available Tools (session_name="%s")

All tool calls below automatically use session_name="%s". Do NOT include session_name in your tool calls.

%s
`, sessionName, sessionName, ToolDescriptions)
}
