# KIT SDK Migration

Replace `opencode acp` subprocess + custom JSON-RPC client with in-process KIT Go SDK (`github.com/mark3labs/kit/pkg/kit`).

## Overview

The current architecture spawns `opencode acp` as a subprocess and communicates via a hand-rolled JSON-RPC 2.0 client over stdin/stdout pipes. KIT SDK provides an in-process Go API that eliminates subprocess management entirely, replacing ~1,400 lines of protocol code with direct SDK calls. KIT handles LLM providers, tool execution, streaming, sessions, and MCP integration natively.

## User Story

**As a** developer maintaining iteratr
**I want** to replace the ACP subprocess with the KIT SDK
**So that** I eliminate subprocess lifecycle complexity, gain access to richer SDK features (compaction, hooks, extensions), and stop maintaining a custom JSON-RPC client

## Feasibility: FULL COMPATIBILITY

KIT SDK covers 100% of current ACP functionality. Every feature iteratr uses has a direct SDK equivalent.

### Complete Feature Mapping

| Current (ACP) | KIT SDK Equivalent |
|---|---|
| Spawn `opencode acp` subprocess | `kit.New(ctx, &Options{})` (in-process) |
| `initialize` handshake | Handled by `kit.New()` |
| `session/new` + `session/set_model` | `kit.New()` with `Model` option or `SetModel()` |
| `session/prompt` with callbacks | `kit.Prompt()` + event subscriptions |
| `agent_message_chunk` → onText | `OnStreaming()` / `EventMessageUpdate` |
| `agent_thought_chunk` → onThinking | `EventReasoningDelta` |
| `tool_call` (pending) | `ToolCallEvent` with `ToolCallID`, `ToolKind` |
| `tool_call_update` (in_progress) | `ToolExecutionStartEvent` with `ToolCallID`, `ToolKind` |
| `tool_call_update` (completed) | `ToolResultEvent` with `ToolCallID`, `ToolKind`, `Metadata` |
| `tool_call_update` RawInput | `ToolCallEvent.ParsedArgs` (pre-parsed `map[string]any`) |
| `tool_call_update` FileDiff | `ToolResultMetadata.FileDiffs[].{Path,Additions,Deletions,IsNew}` |
| `tool_call_update` DiffBlocks | `FileDiffInfo.DiffBlocks[].{OldText,NewText}` |
| `tool_call_update` tool Kind | `ToolKind` field: "execute", "edit", "read", "search", "agent" |
| `tool_call_update` ToolCallID | `ToolCallID` on all tool lifecycle events |
| Subagent session ID | `ListChildSessions(parentID)` via session store |
| StopReason from prompt result | `TurnResult.StopReason` and `TurnEndEvent.StopReason` |
| `session/request_permission` auto-grant | `OnBeforeToolCall` hook (return nil = allow) |
| `session/load` for replay | `GetEntries()`, `GetBranch()`, `BuildContext()` on TreeManager |
| MCP server registration | `MCPServerConfig` in Kit config |
| Process kill on context cancel | `kit.Close()` + context propagation |
| Fresh session per iteration | `ClearSession()` or `NoSession: true` |

## Architecture Change

**Before (ACP subprocess):**
```
orchestrator → Runner → acpConn → [opencode acp subprocess]
                                     ↕ stdin/stdout JSON-RPC
                                     ↕ session/update notifications
                                     └──Bash──→ iteratr tool <cmd>
```

**After (KIT SDK in-process):**
```
orchestrator → KitAgent → kit.Kit (in-process)
                              ├─ Built-in tools (bash, read, write, edit, grep, find, ls)
                              ├─ MCP tools (iteratr-tools server)
                              ├─ Event bus (streaming, tool calls, turns)
                              └─ Session management (tree-based JSONL)
```

No subprocess, no pipes, no JSON-RPC. Direct Go function calls.

## Technical Implementation

### New: `internal/agent/kitagent.go`

Replaces `acp.go` + `runner.go`. Wraps `kit.Kit` with iteratr event mapping.

```go
type KitAgent struct {
    kit          *kit.Kit
    model        string
    workDir      string
    mcpServerURL string
    onText       func(string)
    onToolCall   func(ToolCallEvent)
    onThinking   func(string)
    onFinish     func(FinishEvent)
    onFileChange func(FileChange)
    unsubscribes []func()
}

type KitAgentConfig struct {
    Model        string
    WorkDir      string
    MCPServerURL string
    OnText       func(string)
    OnToolCall   func(ToolCallEvent)
    OnThinking   func(string)
    OnFinish     func(FinishEvent)
    OnFileChange func(FileChange)
}
```

### Initialization

```go
func NewKitAgent(ctx context.Context, cfg KitAgentConfig) (*KitAgent, error) {
    host, err := kit.New(ctx, &kit.Options{
        Model:     cfg.Model,
        Streaming: true,
        NoSession: true, // Ephemeral sessions - fresh context per iteration
        // MCP config for iteratr-tools
    })
    if err != nil {
        return nil, err
    }

    agent := &KitAgent{kit: host, ...}
    agent.subscribeEvents()
    return agent, nil
}
```

### Event Mapping

```go
func (a *KitAgent) subscribeEvents() {
    // Text streaming
    a.unsub(a.kit.OnStreaming(func(e kit.MessageUpdateEvent) {
        if a.onText != nil {
            a.onText(e.Chunk)
        }
    }))

    // Reasoning/thinking
    a.unsub(a.kit.Subscribe(func(e kit.Event) {
        if re, ok := e.(kit.ReasoningDeltaEvent); ok && a.onThinking != nil {
            a.onThinking(re.Delta)
        }
    }))

    // Tool call lifecycle (pending)
    a.unsub(a.kit.OnToolCall(func(e kit.ToolCallEvent) {
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

    // Tool call lifecycle (in_progress)
    a.unsub(a.kit.Subscribe(func(e kit.Event) {
        if ev, ok := e.(kit.ToolExecutionStartEvent); ok && a.onToolCall != nil {
            a.onToolCall(ToolCallEvent{
                ToolCallID: ev.ToolCallID,
                Title:      ev.ToolName,
                Kind:       ev.ToolKind,
                Status:     "in_progress",
            })
        }
    }))

    // Tool result (completed/error) with full metadata
    a.unsub(a.kit.OnToolResult(func(e kit.ToolResultEvent) {
        status := "completed"
        if e.IsError {
            status = "error"
        }
        if a.onToolCall != nil {
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

                // File change callback
                if a.onFileChange != nil {
                    a.onFileChange(FileChange{
                        AbsPath:   fd.Path,
                        IsNew:     fd.IsNew,
                        Additions: fd.Additions,
                        Deletions: fd.Deletions,
                    })
                }
            }

            a.onToolCall(event)
        }
    }))

    // Turn completion with stop reason
    a.unsub(a.kit.OnTurnEnd(func(e kit.TurnEndEvent) {
        if a.onFinish != nil {
            errMsg := ""
            if e.Error != nil {
                errMsg = e.Error.Error()
            }
            a.onFinish(FinishEvent{
                StopReason: e.StopReason,
                Error:      errMsg,
                Model:      a.model,
                Provider:   extractProvider(a.model),
            })
        }
    }))
}
```

### Iteration Execution

```go
func (a *KitAgent) RunIteration(ctx context.Context, prompt string, hookOutput string) error {
    a.kit.ClearSession()

    fullPrompt := prompt
    if hookOutput != "" {
        fullPrompt = hookOutput + "\n\n" + prompt
    }

    startTime := time.Now()
    result, err := a.kit.PromptResult(ctx, fullPrompt)
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
        return fmt.Errorf("prompt failed: %w", err)
    }

    if a.onFinish != nil {
        a.onFinish(FinishEvent{
            StopReason: result.StopReason,
            Duration:   duration,
            Model:      a.model,
            Provider:   extractProvider(a.model),
        })
    }
    return nil
}

func (a *KitAgent) SendMessages(ctx context.Context, texts []string) error {
    combined := strings.Join(texts, "\n\n")
    _, err := a.kit.PromptResult(ctx, combined)
    return err
}

func (a *KitAgent) Stop() {
    for _, unsub := range a.unsubscribes {
        unsub()
    }
    a.kit.Close()
}
```

### MCP Server Configuration

```go
kit.New(ctx, &kit.Options{
    CLI: &kit.CLIOptions{
        MCPConfig: &config.Config{
            MCPServers: map[string]config.MCPServerConfig{
                "iteratr-tools": {
                    Type: "remote",
                    URL:  mcpServerURL,
                },
            },
        },
    },
})
```

### Subagent Viewer (Session Replay)

Replace `SessionLoader` (spawns subprocess) with direct KIT session reads.

```go
// Load session and iterate entries for rendering
func LoadSubagentSession(ctx context.Context, parentSessionID string) ([]SessionEntry, error) {
    // Discover child sessions via session store
    children, err := kit.ListChildSessions(parentSessionID)
    if err != nil || len(children) == 0 {
        return nil, fmt.Errorf("no subagent sessions found")
    }

    // Load the session via Kit
    host, err := kit.New(ctx, &kit.Options{
        SessionPath: children[0].Path,
    })
    if err != nil {
        return nil, err
    }
    defer host.Close()

    // Get ordered session entries with full metadata
    tm := host.GetTreeSession()
    branch := tm.GetBranch("") // Root to leaf
    // Parse entries into renderable format for subagent modal
    return convertEntries(branch), nil
}
```

### File Change Tracking

Direct extraction from `ToolResultEvent.Metadata`:

```go
// In tool result event handler (shown in subscribeEvents above):
if e.Metadata != nil {
    for _, fd := range e.Metadata.FileDiffs {
        onFileChange(FileChange{
            AbsPath:   fd.Path,
            IsNew:     fd.IsNew,
            Additions: fd.Additions,
            Deletions: fd.Deletions,
        })
    }
}
```

## Files Changed

### Deleted
- `internal/agent/acp.go` (1147 lines) - Entire JSON-RPC client
- `internal/agent/runner.go` (271 lines) - Subprocess manager

### New
- `internal/agent/kitagent.go` (~250 lines) - KIT SDK wrapper

### Modified
- `internal/orchestrator/orchestrator.go` - Replace `Runner` → `KitAgent`
- `internal/tui/subagent_modal.go` - Replace `SessionLoader` → KIT session reads
- `internal/tui/specwizard/wizard.go` - Replace `Runner` → `KitAgent`
- `go.mod` - Add `github.com/mark3labs/kit`

### Unchanged
- `internal/agent/filetracker.go` - FileTracker stays, input source changes
- `internal/agent/types.go` - ToolCallEvent, FinishEvent, FileChange types preserved
- `internal/tui/app.go` - Message types unchanged

## Tasks

### 1. Add KIT SDK dependency
- [ ] Add `github.com/mark3labs/kit` to go.mod
- [ ] Run `go mod tidy` to resolve dependency tree

### 2. Create KitAgent wrapper
- [ ] Create `internal/agent/kitagent.go` with KitAgent struct
- [ ] Implement `NewKitAgent(ctx, cfg)` with `kit.New()` initialization
- [ ] Configure MCP server (iteratr-tools) in Kit options
- [ ] Implement `subscribeEvents()` with full event mapping:
  - OnStreaming → onText
  - ReasoningDelta → onThinking
  - ToolCallEvent → onToolCall (status=pending, with ToolCallID, ToolKind, ParsedArgs)
  - ToolExecutionStartEvent → onToolCall (status=in_progress)
  - ToolResultEvent → onToolCall (status=completed/error, with FileDiff, DiffBlocks, Output)
  - ToolResultEvent.Metadata.FileDiffs → onFileChange
  - TurnEndEvent → onFinish (with StopReason)
- [ ] Implement `RunIteration(ctx, prompt, hookOutput)` with ClearSession + PromptResult
- [ ] Implement `SendMessages(ctx, texts)` with combined prompt
- [ ] Implement `Stop()` with unsubscribe + Close cleanup

### 3. Update orchestrator
- [ ] Replace `agent.NewRunner()` with `agent.NewKitAgent()` in orchestrator.go
- [ ] Update TUI mode callback wiring (same callback signatures, no changes needed)
- [ ] Update headless mode callback wiring
- [ ] Replace `runner.Start(ctx)` with KitAgent creation (init happens in constructor)
- [ ] Replace `runner.Stop()` with `kitAgent.Stop()`
- [ ] Replace `runner.RunIteration()` calls with `kitAgent.RunIteration()`
- [ ] Replace `runner.SendMessages()` calls with `kitAgent.SendMessages()`

### 4. Update spec wizard
- [ ] Replace `agent.NewRunner()` with `agent.NewKitAgent()` in wizard.go
- [ ] Update runner lifecycle (Start/Stop → constructor/Stop)
- [ ] Update AgentPhaseReadyMsg to carry KitAgent

### 5. Replace SessionLoader in subagent viewer
- [ ] Replace `agent.NewSessionLoader()` with KIT session loading
- [ ] Use `kit.ListChildSessions(parentID)` to discover subagent sessions
- [ ] Load session via `kit.New(ctx, &Options{SessionPath: ...})`
- [ ] Use `GetTreeSession().GetBranch("")` for ordered entry retrieval
- [ ] Map KIT session message parts to subagent modal rendering:
  - MessageEntry (role=user) → onUser
  - MessageEntry (role=assistant) text parts → onText
  - MessageEntry reasoning parts → onThinking
  - MessageEntry tool_call parts → onToolCall (with ToolCallID, Kind)
  - MessageEntry tool_result parts → onToolCall (completed, with output)
- [ ] Remove SessionLoader struct and NewSessionLoader

### 6. Delete old ACP code
- [ ] Delete `internal/agent/acp.go`
- [ ] Delete `internal/agent/runner.go`
- [ ] Remove `opencode` binary dependency from build/docs

### 7. Update documentation
- [ ] Update AGENTS.md to reference KIT SDK instead of opencode ACP
- [ ] Mark specs/acp-migration.md as superseded by this spec

### 8. Verify and test
- [ ] Manual test: verify agent text streams to TUI in real-time
- [ ] Manual test: verify tool calls show lifecycle (pending → in_progress → completed)
- [ ] Manual test: verify edit tool shows file diffs in TUI
- [ ] Manual test: verify auto-commit tracks file changes with line counts
- [ ] Manual test: verify subagent viewer loads and displays session
- [ ] Manual test: verify headless mode works
- [ ] Manual test: verify Ctrl+C gracefully shuts down
- [ ] Manual test: verify model selection works
- [ ] Run existing teatest suite

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| KIT SDK API instability | Medium | Pin version, wrap in KitAgent adapter |
| Session format differs from opencode | Low | Fresh start OK (iteratr sessions are ephemeral per iteration) |
| MCP server config model differs | Low | KIT supports remote MCP natively |
| Performance change (in-process vs subprocess) | Low | Should improve (no IPC overhead) |
| KIT dependency size | Low | Single Go module, no CGO |

## Benefits

1. **-1,400 lines** - Delete acp.go (1147) + runner.go (271), add kitagent.go (~250)
2. **No subprocess management** - No pipe handling, process kill, EOF detection
3. **No `opencode` binary dependency** - One less external tool to install/maintain
4. **Richer tool metadata** - ToolCallID, ToolKind, ParsedArgs, FileDiffs, DiffBlocks natively
5. **Context compaction** - Auto-summarize long conversations (available if needed)
6. **Hook system** - BeforeToolCall, AfterToolResult, BeforeTurn for extensibility
7. **Connection pooling** - KIT manages MCP connections with health checks
8. **Model registry** - KIT knows all providers/models natively
9. **Better error handling** - Go errors instead of JSON-RPC error code parsing
