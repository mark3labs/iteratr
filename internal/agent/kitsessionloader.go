package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	kit "github.com/mark3labs/kit/pkg/kit"

	"github.com/mark3labs/iteratr/internal/logger"
)

// FindSessionByID searches for a KIT session file matching the given session ID.
// Returns the session file path, or empty string if not found.
func FindSessionByID(sessionID, workDir string) (string, error) {
	sessions, err := kit.ListSessions(workDir)
	if err != nil {
		return "", err
	}
	for _, s := range sessions {
		if s.ID == sessionID {
			return s.Path, nil
		}
	}
	// Also search all sessions in case the subagent used a different workdir
	allSessions, err := kit.ListAllSessions()
	if err != nil {
		return "", err
	}
	for _, s := range allSessions {
		if s.ID == sessionID {
			return s.Path, nil
		}
	}
	return "", fmt.Errorf("session %s not found", sessionID)
}

// KitSessionLoader loads a KIT session for replay in the subagent viewer.
// Replaces the former SessionLoader that spawned an opencode ACP subprocess.
// Instead, it loads the session in-process via the KIT SDK and iterates
// through structured messages, dispatching to callbacks.
type KitSessionLoader struct {
	messages []kit.StructuredMessage
	index    int
	host     *kit.Kit
}

// NewKitSessionLoader creates a session loader that reads from the KIT session
// file at the given path.
func NewKitSessionLoader(ctx context.Context, sessionPath, workDir string) (*KitSessionLoader, error) {
	logger.Debug("Loading KIT session: %s", sessionPath)

	host, err := kit.New(ctx, &kit.Options{
		SessionPath: sessionPath,
		Quiet:       true,
		NoSession:   false,
		SessionDir:  workDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	msgs := host.GetStructuredMessages()
	logger.Debug("KIT session loaded: %d messages", len(msgs))

	return &KitSessionLoader{
		messages: msgs,
		index:    0,
		host:     host,
	}, nil
}

// ReadAndProcess reads one message and calls the appropriate callback.
// Returns true if a message was processed, false if no more messages.
// Callbacks are: onText, onToolCall, onThinking, onUser
func (s *KitSessionLoader) ReadAndProcess(
	onText func(string),
	onToolCall func(ToolCallEvent),
	onThinking func(string),
	onUser func(string),
) (bool, error) {
	if s.index >= len(s.messages) {
		return false, io.EOF
	}

	msg := s.messages[s.index]
	s.index++

	for _, part := range msg.Parts {
		switch p := part.(type) {
		case kit.TextContent:
			if p.Text == "" {
				continue
			}
			switch msg.Role {
			case "user":
				if onUser != nil {
					onUser(p.Text)
				}
			default:
				if onText != nil {
					onText(p.Text)
				}
			}

		case kit.ReasoningContent:
			if p.Thinking != "" && onThinking != nil {
				onThinking(p.Thinking)
			}

		case kit.ToolCall:
			if onToolCall != nil {
				var parsedArgs map[string]any
				if p.Input != "" {
					_ = json.Unmarshal([]byte(p.Input), &parsedArgs)
				}
				onToolCall(ToolCallEvent{
					ToolCallID: p.ID,
					Title:      p.Name,
					Status:     "pending",
					RawInput:   parsedArgs,
				})
			}

		case kit.ToolResult:
			if onToolCall != nil {
				status := "completed"
				if p.IsError {
					status = "error"
				}
				onToolCall(ToolCallEvent{
					ToolCallID: p.ToolCallID,
					Title:      p.Name,
					Status:     status,
					Output:     p.Content,
				})
			}
		}
	}

	return true, nil
}

// Close releases resources.
func (s *KitSessionLoader) Close() error {
	if s.host != nil {
		err := s.host.Close()
		s.host = nil
		return err
	}
	return nil
}
