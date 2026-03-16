package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	kit "github.com/mark3labs/kit/pkg/kit"

	"github.com/mark3labs/iteratr/internal/logger"
)

// KitAgent wraps the KIT SDK to provide the same interface as the former
// ACP-based Runner. It runs the LLM agent in-process instead of spawning
// a subprocess.
type KitAgent struct {
	model        string
	workDir      string
	mcpServerURL string

	// Callbacks (same signatures as former Runner)
	onText       func(string)
	onToolCall   func(ToolCallEvent)
	onThinking   func(string)
	onFinish     func(FinishEvent)
	onFileChange func(FileChange)

	// KIT SDK instance (created in Start, reused across iterations)
	host         *kit.Kit
	unsubscribes []func()
}

// KitAgentConfig holds configuration for creating a new KitAgent.
type KitAgentConfig struct {
	Model        string              // LLM model to use (e.g., "anthropic/claude-sonnet-4-5")
	WorkDir      string              // Working directory for agent
	SessionName  string              // Session name (unused by KIT but kept for API compat)
	NATSPort     int                 // NATS server port (unused by KIT but kept for API compat)
	MCPServerURL string              // MCP server URL for tool access
	OnText       func(text string)   // Callback for text output
	OnToolCall   func(ToolCallEvent) // Callback for tool lifecycle events
	OnThinking   func(string)        // Callback for thinking/reasoning output
	OnFinish     func(FinishEvent)   // Callback for iteration finish events
	OnFileChange func(FileChange)    // Callback for file modifications
}

// NewKitAgent creates a new KitAgent instance. Call Start() to initialize the
// KIT SDK and subscribe to events.
func NewKitAgent(cfg KitAgentConfig) *KitAgent {
	return &KitAgent{
		model:        cfg.Model,
		workDir:      cfg.WorkDir,
		mcpServerURL: cfg.MCPServerURL,
		onText:       cfg.OnText,
		onToolCall:   cfg.OnToolCall,
		onThinking:   cfg.OnThinking,
		onFinish:     cfg.OnFinish,
		onFileChange: cfg.OnFileChange,
	}
}

// Start initializes the KIT SDK instance and subscribes to events.
// Must be called before RunIteration.
func (a *KitAgent) Start(ctx context.Context) error {
	logger.Debug("Starting KIT SDK agent")

	opts := &kit.Options{
		Model:      a.model,
		Streaming:  true,
		NoSession:  true, // Ephemeral sessions — fresh context per iteration
		SessionDir: a.workDir,
		Quiet:      true,
	}

	// Configure MCP server if URL provided
	if a.mcpServerURL != "" {
		opts.CLI = &kit.CLIOptions{
			MCPConfig: &kit.Config{
				MCPServers: map[string]kit.MCPServerConfig{
					"iteratr-tools": {
						Type: "remote",
						URL:  a.mcpServerURL,
					},
				},
			},
		}
	}

	host, err := kit.New(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to initialize KIT SDK: %w", err)
	}

	a.host = host
	a.subscribeEvents()

	logger.Debug("KIT SDK agent ready")
	return nil
}

// subscribeEvents wires KIT SDK events to the iteratr callback model.
func (a *KitAgent) subscribeEvents() {
	// Text streaming
	a.addUnsub(a.host.OnStreaming(func(e kit.MessageUpdateEvent) {
		if a.onText != nil {
			a.onText(e.Chunk)
		}
	}))

	// Reasoning/thinking + tool execution lifecycle
	a.addUnsub(a.host.Subscribe(func(e kit.Event) {
		switch ev := e.(type) {
		case kit.ReasoningDeltaEvent:
			if a.onThinking != nil {
				a.onThinking(ev.Delta)
			}

		case kit.ToolExecutionStartEvent:
			if a.onToolCall != nil {
				a.onToolCall(ToolCallEvent{
					ToolCallID: ev.ToolCallID,
					Title:      ev.ToolName,
					Kind:       ev.ToolKind,
					Status:     "in_progress",
				})
			}
		}
	}))

	// Tool call parsed (pending)
	a.addUnsub(a.host.OnToolCall(func(e kit.ToolCallEvent) {
		if a.onToolCall != nil {
			a.onToolCall(ToolCallEvent{
				ToolCallID: e.ToolCallID,
				Title:      e.ToolName,
				Kind:       e.ToolKind,
				Status:     "pending",
				RawInput:   e.ParsedArgs,
			})
		}
	}))

	// Tool result (completed/error) with full metadata
	a.addUnsub(a.host.OnToolResult(func(e kit.ToolResultEvent) {
		if a.onToolCall == nil {
			return
		}

		status := "completed"
		if e.IsError {
			status = "error"
		}

		event := ToolCallEvent{
			ToolCallID: e.ToolCallID,
			Title:      e.ToolName,
			Kind:       e.ToolKind,
			Status:     status,
			RawInput:   e.ParsedArgs,
			Output:     e.Result,
		}

		// Extract file diff metadata from edit/write tools
		if e.Metadata != nil && len(e.Metadata.FileDiffs) > 0 {
			fd := e.Metadata.FileDiffs[0]
			event.FileDiff = &FileDiff{
				File:      fd.Path,
				Additions: fd.Additions,
				Deletions: fd.Deletions,
			}
			for _, db := range fd.DiffBlocks {
				event.DiffBlocks = append(event.DiffBlocks, DiffBlock{
					Path:    fd.Path,
					OldText: db.OldText,
					NewText: db.NewText,
				})
			}

			// Notify file change callback for each modified file
			if a.onFileChange != nil {
				for _, fdInfo := range e.Metadata.FileDiffs {
					a.onFileChange(FileChange{
						AbsPath:   fdInfo.Path,
						IsNew:     fdInfo.IsNew,
						Additions: fdInfo.Additions,
						Deletions: fdInfo.Deletions,
					})
				}
			}
		}

		a.onToolCall(event)
	}))

	// Turn completion with stop reason
	a.addUnsub(a.host.OnTurnEnd(func(e kit.TurnEndEvent) {
		// Note: onFinish is called by RunIteration/SendMessages with timing info.
		// This subscription is only used for error-path finish events that bypass
		// the normal return path. The main finish event is dispatched in
		// RunIteration/SendMessages to include Duration.
	}))
}

// RunIteration clears the session for fresh context and sends the prompt.
// Optional hookOutput is prepended to the prompt.
func (a *KitAgent) RunIteration(ctx context.Context, prompt string, hookOutput string) error {
	if a.host == nil {
		return fmt.Errorf("KIT agent not started — call Start() first")
	}

	// Clear session for fresh context
	a.host.ClearSession()

	// Build prompt: hook output + main prompt
	fullPrompt := prompt
	if hookOutput != "" {
		fullPrompt = hookOutput + "\n\n" + prompt
	}

	logger.Debug("Running KIT iteration, prompt length: %d", len(fullPrompt))

	startTime := time.Now()
	result, err := a.host.PromptResult(ctx, fullPrompt)
	duration := time.Since(startTime)

	if err != nil {
		if a.onFinish != nil {
			stopReason := "error"
			if ctx.Err() == context.Canceled {
				stopReason = "cancelled"
			}
			a.onFinish(FinishEvent{
				StopReason: stopReason,
				Error:      err.Error(),
				Duration:   duration,
				Model:      a.model,
				Provider:   extractProvider(a.model),
			})
		}
		return fmt.Errorf("KIT prompt failed: %w", err)
	}

	// Map KIT stop reason to iteratr conventions
	stopReason := mapStopReason(result.StopReason)

	// Detect silent failures: agent returned without producing text
	if stopReason == "end_turn" && result.Response == "" {
		logger.Warn("Agent returned end_turn without producing any output — possible credential or API error")
		if a.onFinish != nil {
			a.onFinish(FinishEvent{
				StopReason: "error",
				Error:      "agent returned no output — this may indicate a credential error or model availability issue",
				Duration:   duration,
				Model:      a.model,
				Provider:   extractProvider(a.model),
			})
		}
		return fmt.Errorf("agent returned no output — this may indicate a credential error or model availability issue")
	}

	if a.onFinish != nil {
		a.onFinish(FinishEvent{
			StopReason: stopReason,
			Duration:   duration,
			Model:      a.model,
			Provider:   extractProvider(a.model),
		})
	}

	logger.Debug("KIT iteration completed: %s", stopReason)
	return nil
}

// SendMessages sends user messages to the current session as a single prompt.
func (a *KitAgent) SendMessages(ctx context.Context, texts []string) error {
	if a.host == nil {
		return fmt.Errorf("KIT agent not started — call Start() first")
	}
	if len(texts) == 0 {
		return nil
	}

	combined := strings.Join(texts, "\n\n")
	logger.Debug("Sending %d user message(s) to KIT session", len(texts))

	startTime := time.Now()
	result, err := a.host.PromptResult(ctx, combined)
	duration := time.Since(startTime)

	if err != nil {
		if a.onFinish != nil {
			stopReason := "error"
			if ctx.Err() == context.Canceled {
				stopReason = "cancelled"
			}
			a.onFinish(FinishEvent{
				StopReason: stopReason,
				Error:      err.Error(),
				Duration:   duration,
				Model:      a.model,
				Provider:   extractProvider(a.model),
			})
		}
		return fmt.Errorf("KIT user message failed: %w", err)
	}

	if a.onFinish != nil {
		a.onFinish(FinishEvent{
			StopReason: mapStopReason(result.StopReason),
			Duration:   duration,
			Model:      a.model,
			Provider:   extractProvider(a.model),
		})
	}

	logger.Debug("User message processed successfully")
	return nil
}

// Stop cleans up the KIT SDK instance and unsubscribes all event listeners.
func (a *KitAgent) Stop() {
	for _, unsub := range a.unsubscribes {
		unsub()
	}
	a.unsubscribes = nil

	if a.host != nil {
		logger.Debug("Closing KIT SDK instance")
		if err := a.host.Close(); err != nil {
			logger.Warn("Failed to close KIT SDK: %v", err)
		}
		a.host = nil
	}
	logger.Debug("KIT agent stopped")
}

// addUnsub stores an unsubscribe function for cleanup in Stop().
func (a *KitAgent) addUnsub(unsub func()) {
	a.unsubscribes = append(a.unsubscribes, unsub)
}

// extractProvider parses provider name from model string.
// Model format is typically "provider/model-name" (e.g., "anthropic/claude-sonnet-4-5").
// Returns capitalized provider name (e.g., "Anthropic") or empty string if no slash.
func extractProvider(model string) string {
	if idx := strings.Index(model, "/"); idx >= 0 {
		provider := model[:idx]
		if len(provider) > 0 {
			return strings.ToUpper(provider[:1]) + provider[1:]
		}
		return provider
	}
	return ""
}

// mapStopReason translates KIT SDK stop reasons to iteratr conventions.
// KIT uses: "stop", "length", "tool-calls", "content-filter", "error", "other", "unknown"
// iteratr uses: "end_turn", "max_tokens", "cancelled", "refusal", "max_turn_requests", "error"
func mapStopReason(kitReason string) string {
	switch kitReason {
	case "stop", "end_turn":
		return "end_turn"
	case "length", "max_tokens":
		return "max_tokens"
	case "content-filter":
		return "refusal"
	case "error":
		return "error"
	case "tool-calls":
		// Agent stopped to execute tools — shouldn't surface as a final reason
		// since KIT handles the tool loop internally. Treat as end_turn.
		return "end_turn"
	case "":
		return "end_turn"
	default:
		return kitReason
	}
}
