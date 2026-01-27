# Subagent Viewer

Display when root agent spawns subagents as a new message type with modal to view subagent session.

## Overview

When the root agent calls the `task` tool to spawn a subagent, display a dedicated `SubagentMessageItem` in the chat. Clicking opens a full-screen modal that spawns `opencode acp`, loads the subagent session via `session/load`, and displays the conversation.

**Limitation**: Session ID is only available on subagent completion (via `rawOutput.metadata.sessionId`). "Click to view" appears after completion. Live viewing only possible for resumed sessions where `rawInput.session_id` exists.

## User Story

As a user watching agent progress, I want to see when subagents are spawned and view their completed work, so I understand the full execution context.

## Requirements

- New `SubagentMessageItem` message type distinct from tool calls
- Shows subagent type, description, status (pending/running/completed)
- "Click to view" appears only after completion (when sessionID available)
- Modal loads session via `session/load`, replays history
- Full-screen modal (covers chat area, prevents clicking other subagents)
- Reuses existing `ScrollList`, `MessageItem` types, and `LogViewer` modal pattern
- ESC closes modal
- Error states shown in modal if ACP spawn or session load fails

## Technical Implementation

### Component Reuse

| Existing Component | Reuse In |
|-------------------|----------|
| `ScrollList` | SubagentModal viewport for message scrolling |
| `TextMessageItem` | Agent text chunks |
| `ToolMessageItem` | Tool calls in subagent session |
| `ThinkingMessageItem` | Agent thinking blocks |
| `UserMessageItem` | Initial prompt to subagent |
| `LogViewer.Draw()` pattern | Modal layout, centering, styling |
| `acpConn` | ACP protocol communication |

### Data Flow

```
1. Detection: tool_call_update (in_progress) → detect subagent_type in rawInput → create SubagentMessageItem
2. Session ID: extract from rawOutput.metadata.sessionId on completion
3. Click: SubagentMessageItem click → OpenSubagentModalMsg (only if sessionID available)
4. Modal: spawn opencode acp → initialize → session/load → display history
```

### Session ID Extraction

Session ID available on `tool_call_update` with `status="completed"`:

```go
// Extract from rawOutput.metadata.sessionId
func extractSessionID(rawOutput map[string]any) string {
    metadata, ok := rawOutput["metadata"].(map[string]any)
    if !ok {
        return ""
    }
    sessionID, _ := metadata["sessionId"].(string)
    return sessionID
}
```

**Note**: ACP protocol does not expose child session ID during execution. The `<task_metadata>` block containing session ID appears in `rawOutput.output` only on completion, same time as metadata.

### Files to Modify/Create

| File | Change |
|------|--------|
| `internal/agent/types.go` | Add `SessionID string` to `ToolCallEvent` |
| `internal/agent/acp.go` | Extract sessionId from metadata on completion; add `loadSession()` |
| `internal/tui/app.go` | Add `SessionID` to `AgentToolCallMsg`; add `subagentModal` field |
| `internal/orchestrator/orchestrator.go` | Pass SessionID through callback |
| `internal/tui/messages.go` | New `SubagentMessageItem` type |
| `internal/tui/subagent_modal.go` | New modal (reuses ScrollList + MessageItem types) |
| `internal/tui/agent.go` | Detect subagent in `AppendToolCall()`, handle click, return `OpenSubagentModalMsg` |

### ACP Session Loading

`session/load` replays session history as notifications:

```go
type loadSessionParams struct {
    SessionID  string `json:"sessionId"`
    Cwd        string `json:"cwd"`
    McpServers []any  `json:"mcpServers"`
}

func (c *acpConn) loadSession(ctx context.Context, sessionID, cwd string) (string, error) {
    params := loadSessionParams{
        SessionID:  sessionID,
        Cwd:        cwd,
        McpServers: []any{},
    }
    reqID, err := c.sendRequest("session/load", params)
    // Wait for response (same structure as session/new)
    // Then stream notifications until EOF:
    // - user_message_chunk
    // - agent_message_chunk
    // - agent_thought_chunk
    // - tool_call, tool_call_update
    // Returns sessionID from response
}
```

### SubagentMessageItem

```go
type SubagentMessageItem struct {
    id           string
    subagentType string
    description  string
    status       ToolStatus  // Pending, Running, Success, Error
    sessionID    string      // Empty until completion
    cachedRender string
    cachedWidth  int
}

func (s *SubagentMessageItem) CanView() bool {
    return s.sessionID != ""
}
```

Render format ("Click to view" only shown when sessionID available):
```
Running:   ◐ [codebase-analyzer] Running...
           Analyze implementation details

Completed: ● [codebase-analyzer] Completed
           Analyze implementation details
                                          [Click to view]

Error:     × [codebase-analyzer] Error
           Analyze implementation details
```

### SubagentModal

```go
type SubagentModal struct {
    scrollList   *ScrollList     // Reuse existing ScrollList
    messages     []MessageItem   // Reuse existing MessageItem types
    toolIndex    map[string]int  // Track tool updates
    
    sessionID    string
    subagentType string
    workDir      string
    
    // ACP subprocess
    cmd          *exec.Cmd
    conn         *acpConn
    
    // State
    loading      bool
    err          error  // Non-nil shows error message in modal
    width, height int
}
```

Key methods mirror AgentOutput:

```go
func (m *SubagentModal) appendText(content string)
func (m *SubagentModal) appendToolCall(event ToolCallEvent)
func (m *SubagentModal) appendThinking(content string)
func (m *SubagentModal) appendUserMessage(text string)

func (m *SubagentModal) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
    // 1. Calculate modal dimensions (LogViewer pattern - full screen overlay)
    // 2. If loading: show spinner
    // 3. If err != nil: show error message with ESC hint
    // 4. Otherwise: render title "Subagent: {type}"
    // 5. Render scrollList.View() in content area
    // 6. Render hint at bottom: [ESC] Close  [↑↓] Scroll
}
```

Error handling:
```go
func (m *SubagentModal) Start() tea.Cmd {
    // Spawn ACP subprocess
    // If spawn fails: m.err = fmt.Errorf("failed to start ACP: %w", err)
    // If session/load fails: m.err = fmt.Errorf("session not found: %s", sessionID)
}
```

### Session Streaming

Background goroutine processes ACP notifications from `session/load`:

```go
func (m *SubagentModal) streamSession(ctx context.Context) tea.Cmd {
    return func() tea.Msg {
        resp, err := m.conn.readMessage()
        if err == io.EOF {
            return SubagentDoneMsg{} // All history replayed
        }
        if err != nil {
            return SubagentErrorMsg{err}
        }
        
        // Parse notification and return appropriate message type
        switch update.SessionUpdate {
        case "agent_message_chunk":
            return SubagentTextMsg{Text: chunk.Content.Text, Continue: true}
        case "tool_call", "tool_call_update":
            return SubagentToolCallMsg{Event: event, Continue: true}
        case "agent_thought_chunk":
            return SubagentThinkingMsg{Content: chunk.Content.Text, Continue: true}
        case "user_message_chunk":
            return SubagentUserMsg{Text: chunk.Content.Text, Continue: true}
        }
        return SubagentStreamMsg{Continue: true} // Unknown type, keep reading
    }
}

// In App.Update, continue streaming until EOF:
case SubagentTextMsg:
    if a.subagentModal != nil {
        a.subagentModal.appendText(msg.Text)
        if msg.Continue {
            return a, a.subagentModal.streamNext()
        }
    }
```

### App Integration

```go
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Modal takes priority when visible
    if a.subagentModal != nil {
        switch msg := msg.(type) {
        case tea.KeyPressMsg:
            if msg.String() == "esc" {
                a.subagentModal.Close()
                a.subagentModal = nil
                return a, nil
            }
            // Forward scroll keys to modal
            return a, a.subagentModal.Update(msg)
        }
    }
    
    switch msg := msg.(type) {
    case OpenSubagentModalMsg:
        // Close existing modal if any (shouldn't happen with full-screen modal)
        if a.subagentModal != nil {
            a.subagentModal.Close()
        }
        modal := NewSubagentModal(msg.SessionID, msg.SubagentType, a.workDir)
        a.subagentModal = modal
        return a, modal.Start() // Spawns ACP, loads session, starts streaming
        
    case SubagentTextMsg, SubagentToolCallMsg, SubagentThinkingMsg, SubagentUserMsg:
        if a.subagentModal != nil {
            cmd := a.subagentModal.HandleUpdate(msg)
            return a, cmd // Returns streamNext() to continue reading
        }
        
    case SubagentDoneMsg:
        // All history replayed - nothing special to do
        // Modal stays open for viewing until user presses ESC
        
    case SubagentErrorMsg:
        if a.subagentModal != nil {
            a.subagentModal.err = msg.Err
        }
    }
}

// In App.Draw, render modal overlay when visible:
func (a *App) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
    // ... draw other components ...
    if a.subagentModal != nil {
        a.subagentModal.Draw(scr, area) // Full-screen overlay
    }
}
```

## UI Mockup

Main chat view (subagent running - no "Click to view" yet):
```
┌─────────────────────────────────────────────────────┐
│ Agent: I'll analyze the codebase using a subagent.  │
│                                                     │
│   ◐ [codebase-analyzer] Running...                  │
│     Analyze implementation details                  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

Main chat view (subagent completed - "Click to view" available):
```
┌─────────────────────────────────────────────────────┐
│ Agent: I'll analyze the codebase using a subagent.  │
│                                                     │
│   ● [codebase-analyzer] Completed                   │
│     Analyze implementation details                  │
│                                [Click to view]      │
│                                                     │
└─────────────────────────────────────────────────────┘
```

Modal (loading):
```
┌─────────────────────────────────────────────────────────────┐
│              Subagent: codebase-analyzer                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                                                             │
│                    ◐ Loading session...                     │
│                                                             │
│                                                             │
│ ──────────────────────────────────────────────────────────  │
│ [ESC] Close                                                 │
└─────────────────────────────────────────────────────────────┘
```

Modal (error):
```
┌─────────────────────────────────────────────────────────────┐
│              Subagent: codebase-analyzer                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                                                             │
│           × Failed to load session: not found               │
│                                                             │
│                                                             │
│ ──────────────────────────────────────────────────────────  │
│ [ESC] Close                                                 │
└─────────────────────────────────────────────────────────────┘
```

Modal (session loaded):
```
┌─────────────────────────────────────────────────────────────┐
│              Subagent: codebase-analyzer                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─ User ─────────────────────────────────────────────────┐  │
│ │ Analyze implementation details                         │  │
│ └────────────────────────────────────────────────────────┘  │
│                                                             │
│ I'll analyze the codebase to find implementation details.   │
│                                                             │
│ ┌─ grep ─────────────────────────────── ● Completed ─────┐  │
│ │ pattern: "func.*Handler"                               │  │
│ │ Found 15 matches                                       │  │
│ └────────────────────────────────────────────────────────┘  │
│                                                             │
│ Found 15 handler functions across 8 files...                │
│                                                             │
│ ──────────────────────────────────────────────────────────  │
│ [ESC] Close  [↑↓] Scroll                                    │
└─────────────────────────────────────────────────────────────┘
```

## Tasks

### 1. Add SessionID to event chain
- [ ] Add `SessionID string` field to `ToolCallEvent` in agent/types.go
- [ ] Extract sessionId from `rawOutput.metadata.sessionId` on completion in acp.go
- [ ] Add `SessionID string` to `AgentToolCallMsg` in app.go
- [ ] Pass SessionID through orchestrator OnToolCall callback

### 2. Add loadSession to ACP
- [ ] Add `loadSessionParams` struct in acp.go
- [ ] Implement `loadSession()` method (model after `newSession()`)
- [ ] Return after initial response, then stream notifications until EOF

### 3. Create SubagentMessageItem
- [ ] Create `SubagentMessageItem` struct in messages.go
- [ ] Implement `MessageItem` interface (ID, Render, Height)
- [ ] Add status badge rendering (pending/running/completed/error)
- [ ] Add "Click to view" hint (only when sessionID != "")

### 4. Subagent detection in agent output
- [ ] In `AppendToolCall()`, detect subagent by checking `Input["subagent_type"]`
- [ ] Create SubagentMessageItem instead of ToolMessageItem for subagent calls
- [ ] Update sessionID on completion (from AgentToolCallMsg.SessionID)

### 5. Create SubagentModal
- [ ] Create subagent_modal.go with struct (reuse ScrollList, MessageItem types)
- [ ] Implement `Start()` - spawn acp, initialize, load session, return stream cmd
- [ ] Implement continuous streaming with `streamNext()` pattern
- [ ] Implement append methods mirroring AgentOutput
- [ ] Implement `Draw()` with loading/error/content states
- [ ] Implement `Update()` for keyboard scroll handling
- [ ] Implement `Close()` to kill subprocess

### 6. Wire click handling and app integration
- [ ] Add message types: `OpenSubagentModalMsg`, `SubagentTextMsg`, `SubagentToolCallMsg`, `SubagentDoneMsg`, `SubagentErrorMsg`
- [ ] Handle SubagentMessageItem clicks in agent.HandleClick() → return `OpenSubagentModalMsg`
- [ ] Add `subagentModal *SubagentModal` field to App struct
- [ ] Route modal messages in App.Update() with continuous streaming
- [ ] Modal takes keyboard priority when visible; ESC closes
- [ ] Render modal overlay in App.Draw() (full-screen, covers chat)

## Out of Scope

- Nested subagent viewing (subagent's subagents)
- Inline expansion without modal
- Interacting with the subagent (sending messages)
- Live viewing of in-progress subagents (ACP protocol limitation)

## Open Questions

- Cache loaded sessions to avoid re-fetching on re-open?
- Keyboard shortcut to open most recent subagent modal?
