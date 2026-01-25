# Message Queue Enhancement

## Overview

Replace UI-layer single-message queue with orchestrator-layer FIFO queue. Matches crush pattern: unlimited capacity, thread-safe, processes all queued messages sequentially after agent finishes.

## User Story

As a user, I want to send multiple messages while the agent is busy without losing any, and have them processed in order when the agent becomes available.

## Requirements

- Queue unlimited messages while agent is processing (no drops)
- Process all queued messages in FIFO order after each agent response
- Visual feedback: show queue depth when messages pending
- Remove UI-layer `queuedMsg` field (single-message limit)
- Thread-safe queue access (orchestrator receives from TUI goroutine)
- Preserve existing behavior: `sendChan` buffered channel from TUI to orchestrator

## Current State Analysis

### Architecture (Before)

```
User types → Dashboard.Update()
           → if agentBusy: queuedMsg = text (DROPS previous!)
           → else: emit UserInputMsg
                 → App.Update() → sendChan <- text
                 → Orchestrator: select { case <-sendChan } (between iterations only)
                 → runner.SendMessage() [BLOCKS]
```

**Problems:**
1. `queuedMsg` is single string - second message overwrites first
2. Queue at wrong layer (UI instead of orchestrator)
3. Only checks `sendChan` between iterations, not after each `SendMessage()`

### Crush Pattern (Target)

```go
// Agent layer - crush/internal/agent/agent.go
type Agent struct {
    messageQueue   *csync.Map[string, []SessionAgentCall]  // sessionID → queued calls
    activeRequests *csync.Map[string, context.CancelFunc]  // tracks busy sessions
}

func (a *Agent) Run(ctx context.Context, call SessionAgentCall) (*Result, error) {
    // Check if session is busy
    if a.IsSessionBusy(call.SessionID) {
        // Queue and return immediately
        existing, _ := a.messageQueue.Get(call.SessionID)
        a.messageQueue.Set(call.SessionID, append(existing, call))
        return nil, nil
    }
    
    // Mark session busy
    ctx, cancel := context.WithCancel(ctx)
    a.activeRequests.Set(call.SessionID, cancel)
    
    // Execute request...
    result, err := a.execute(ctx, call)
    
    // Release lock
    a.activeRequests.Del(call.SessionID)
    cancel()
    
    // Process queued messages recursively
    queued, ok := a.messageQueue.Get(call.SessionID)
    if ok && len(queued) > 0 {
        first := queued[0]
        a.messageQueue.Set(call.SessionID, queued[1:])
        return a.Run(ctx, first)  // Recursive call
    }
    
    return result, err
}
```

## Technical Implementation

### Architecture (After)

```
User types → Dashboard.Update()
           → Always emit UserInputMsg (no local queue)
           → App.Update() → sendChan <- text (non-blocking, drops if full)
           → Orchestrator.processUserMessages() (called after each agent response)
              → Drains sendChan into local slice
              → Processes each message sequentially via runner.SendMessage()
```

### Key Design Decisions

1. **Queue location**: Orchestrator layer (not agent/runner) - matches existing `sendChan` pattern
2. **Queue storage**: Drain `sendChan` into `[]string` slice before processing - simpler than map
3. **No thread-safe map needed**: Single goroutine drains channel, processes sequentially
4. **Recursive vs loop**: Use loop (Go idiom) instead of crush's recursive pattern

### Code Changes

#### 1. Orchestrator: Add message processing loop

```go
// internal/orchestrator/orchestrator.go

// processUserMessages drains sendChan and processes all queued messages sequentially.
// Called after each agent response (iteration or user message).
// Returns when channel is empty and all messages processed.
func (o *Orchestrator) processUserMessages() error {
    for {
        select {
        case <-o.ctx.Done():
            return o.ctx.Err()
        case userMsg := <-o.sendChan:
            logger.Info("Processing queued user message: %s", userMsg)
            
            // Notify TUI that we're processing a queued message
            if o.tuiProgram != nil {
                o.tuiProgram.Send(tui.QueuedMessageProcessingMsg{Text: userMsg})
            }
            
            if err := o.runner.SendMessage(o.ctx, userMsg); err != nil {
                logger.Error("Failed to send user message: %v", err)
                if o.tuiProgram != nil {
                    o.tuiProgram.Send(tui.AgentOutputMsg{
                        Content: fmt.Sprintf("\n[Error sending message: %v]\n", err),
                    })
                }
                // Continue processing remaining messages
            }
            // Loop continues - check for more messages
        default:
            // Channel empty, all messages processed
            return nil
        }
    }
}
```

#### 2. Orchestrator: Update iteration loop

```go
// internal/orchestrator/orchestrator.go (in runIterationLoop)

for {
    // ... run iteration ...
    
    // After iteration completes, process ALL queued user messages
    if err := o.processUserMessages(); err != nil {
        if errors.Is(err, context.Canceled) {
            return nil
        }
        return err
    }
    
    iterationCount++
}
```

#### 3. Dashboard: Remove UI-layer queue

```go
// internal/tui/dashboard.go

type Dashboard struct {
    // ... existing fields ...
    // REMOVE: queuedMsg string
    agentBusy bool  // Keep for placeholder text only
}

func (d *Dashboard) Update(msg tea.Msg) tea.Cmd {
    // ... existing code ...
    
    case "enter":
        if d.agentOutput != nil {
            text := d.agentOutput.InputValue()
            if text != "" {
                // Always emit immediately - orchestrator handles queueing
                d.agentOutput.ResetInput()
                d.inputFocused = false
                d.agentOutput.SetInputFocused(false)
                d.focusPane = FocusAgent
                return func() tea.Msg {
                    return UserInputMsg{Text: text}
                }
            }
        }
        return nil
}

func (d *Dashboard) SetAgentBusy(busy bool) tea.Cmd {
    d.agentBusy = busy
    if d.agentOutput != nil {
        d.agentOutput.SetBusy(busy)
    }
    // REMOVE: queued message emission logic
    return nil
}
```

#### 4. App: Track queue depth for visual feedback

```go
// internal/tui/app.go

type App struct {
    // ... existing fields ...
    queueDepth int  // Number of messages waiting in orchestrator queue
}

// New message types
type QueueDepthMsg struct {
    Depth int
}

type QueuedMessageProcessingMsg struct {
    Text string
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case UserInputMsg:
        // Increment queue depth when sending
        a.queueDepth++
        select {
        case a.sendChan <- msg.Text:
            // Message queued successfully
        default:
            // Channel full - this shouldn't happen with proper sizing
            a.queueDepth--
            logger.Warn("sendChan full, message dropped: %s", msg.Text)
        }
        // Update UI to show queue depth
        return a, a.dashboard.SetQueueDepth(a.queueDepth)
        
    case QueuedMessageProcessingMsg:
        // Decrement queue depth when processing starts
        a.queueDepth--
        if a.queueDepth < 0 {
            a.queueDepth = 0
        }
        return a, a.dashboard.SetQueueDepth(a.queueDepth)
    }
}
```

#### 5. AgentOutput: Show queue indicator

```go
// internal/tui/agent.go

type AgentOutput struct {
    // ... existing fields ...
    queueDepth int
}

func (a *AgentOutput) SetQueueDepth(depth int) {
    a.queueDepth = depth
}

// In Draw() - show indicator in input area when messages queued
func (a *AgentOutput) renderInputArea() string {
    // ... existing input rendering ...
    
    if a.queueDepth > 0 {
        indicator := fmt.Sprintf(" (%d queued)", a.queueDepth)
        // Append to input line or show below
    }
}
```

### Message Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ TUI (Bubbletea goroutine)                                        │
│                                                                  │
│  User types "fix bug" → Enter                                    │
│       ↓                                                          │
│  Dashboard.Update() → emit UserInputMsg{Text: "fix bug"}        │
│       ↓                                                          │
│  App.Update() → queueDepth++ → sendChan <- "fix bug"            │
│       ↓                                                          │
│  Dashboard shows "(1 queued)" indicator                          │
└──────────────────────────────┬──────────────────────────────────┘
                               │ sendChan (buffered, cap 10)
                               ↓
┌─────────────────────────────────────────────────────────────────┐
│ Orchestrator (separate goroutine)                                │
│                                                                  │
│  runIterationLoop():                                             │
│    1. RunIteration() [BLOCKS until agent done]                   │
│    2. processUserMessages():                                     │
│       for { select { case msg := <-sendChan: ... } }             │
│       - Send QueuedMessageProcessingMsg to TUI                   │
│       - runner.SendMessage(msg) [BLOCKS]                         │
│       - Loop until sendChan empty                                │
│    3. Continue to next iteration                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Thread Safety Analysis

| Component | Access Pattern | Safety Mechanism |
|-----------|----------------|------------------|
| `sendChan` | TUI writes, Orchestrator reads | Go channel (thread-safe) |
| `queueDepth` | TUI reads/writes only | Single goroutine (Bubbletea) |
| `processUserMessages` | Orchestrator only | Single goroutine |
| `runner.SendMessage` | Orchestrator only | Sequential calls |

No additional synchronization needed - existing channel provides safety.

### Channel Capacity Consideration

Current: `sendChan: make(chan string, 10)`

- 10 messages should be sufficient for normal use
- If user spam-types, oldest messages may drop (existing behavior)
- Could increase to 100 if needed, but indicates UX problem
- Alternative: Unbuffered channel + dedicated drain goroutine (over-engineering)

**Decision**: Keep capacity at 10, add warning log when dropped.

## Tasks

### 1. Tracer bullet: processUserMessages loop

Add the message processing loop to orchestrator. Verify it drains channel correctly.

- [ ] Add `processUserMessages() error` method to `Orchestrator` in `orchestrator.go`
- [ ] Implement drain loop: `for { select { case msg := <-sendChan: ... default: return } }`
- [ ] Log each message processed: `logger.Info("Processing queued user message: %s", msg)`
- [ ] Call `runner.SendMessage()` for each message
- [ ] Handle errors without breaking loop (log and continue)
- [ ] Add unit test: send 3 messages to channel, verify all processed in order

### 2. Tracer bullet: Wire into iteration loop

Replace single-message handling with processUserMessages call.

- [ ] In `runIterationLoop()`, replace existing `select { case userMsg := <-o.sendChan }` block
- [ ] Call `o.processUserMessages()` after each `RunIteration()` completes
- [ ] Call `o.processUserMessages()` after agent finish (stop reason received)
- [ ] Handle context cancellation: return early if `ctx.Done()`
- [ ] Verify: send message while agent busy → processed after agent finishes
- [ ] Verify: send 3 messages while busy → all 3 processed in order

### 3. Remove UI-layer queue

Dashboard should always emit UserInputMsg, no local queueing.

- [ ] Remove `queuedMsg string` field from `Dashboard` struct
- [ ] In `Update()` Enter handler: remove `if d.agentBusy` branch
- [ ] Always emit `UserInputMsg` regardless of busy state
- [ ] In `SetAgentBusy()`: remove queued message emission logic (lines 239-244)
- [ ] Keep `agentBusy` field for input placeholder text only
- [ ] Verify: type message while busy → no local storage, goes to sendChan

### 4. Add queue depth tracking

Visual feedback showing how many messages are waiting.

- [ ] Add `queueDepth int` field to `App` struct
- [ ] Define `QueueDepthMsg struct { Depth int }` in `app.go`
- [ ] Define `QueuedMessageProcessingMsg struct { Text string }` in `app.go`
- [ ] In `UserInputMsg` handler: increment `queueDepth`, send to dashboard
- [ ] In `QueuedMessageProcessingMsg` handler: decrement `queueDepth`
- [ ] Add `SetQueueDepth(depth int)` to `Dashboard`
- [ ] Dashboard forwards to `AgentOutput.SetQueueDepth()`

### 5. Send queue notifications from orchestrator

Orchestrator notifies TUI when processing queued messages.

- [ ] In `processUserMessages()`: before each `SendMessage()`, send `QueuedMessageProcessingMsg`
- [ ] Import tui package in orchestrator (already imported)
- [ ] Verify: queue 2 messages → TUI receives 2 `QueuedMessageProcessingMsg`

### 6. Display queue indicator in input area

Show "(N queued)" when messages are waiting.

- [ ] Add `queueDepth int` field to `AgentOutput`
- [ ] Add `SetQueueDepth(depth int)` method
- [ ] In input area rendering: if `queueDepth > 0`, append indicator
- [ ] Style indicator with `styleDim` (subtle, not distracting)
- [ ] Position: after input text or on right side of input area
- [ ] Verify: queue 2 messages → shows "(2 queued)" → processes → shows "(1 queued)" → "(0 queued)" disappears

### 7. Add drop warning

Log warning when sendChan is full and message dropped.

- [ ] In `App.Update()` UserInputMsg handler: check if send succeeded
- [ ] If `default` case hit (channel full): log warning with message text
- [ ] Optionally: show brief toast/status message to user
- [ ] Verify: fill channel to capacity → next message logs warning

### 8. Cleanup and tests

Final polish and test coverage.

- [ ] Run `go build ./...` - verify no compile errors
- [ ] Run `go test ./...` - verify existing tests pass
- [ ] Update `dashboard_test.go`: remove queuedMsg-related tests
- [ ] Add test: `processUserMessages` processes all messages in order
- [ ] Add test: queue depth increments/decrements correctly
- [ ] Add test: message drop warning logged when channel full
- [ ] Manual test: type 5 messages rapidly while agent busy → all appear in output

### 9. Tracer bullet: UserMessageItem type

Add user message type to display sent messages in the conversation viewport. Messages appear when agent receives them (not when user sends), preserving accurate conversation order.

- [ ] Add `UserMessageItem` struct to `messages.go`: `id`, `content`, `cachedRender`, `cachedWidth`
- [ ] Implement `ID() string` - returns unique ID
- [ ] Implement `Render(width int) string` - right-aligned with right border
- [ ] Implement `Height() int` - count lines in cached render

```go
type UserMessageItem struct {
    id           string
    content      string
    cachedRender string
    cachedWidth  int
}
```

### 10. Style user messages

Visual distinction from assistant messages - right-aligned, different accent.

- [ ] Add `styleUserMessage` to `styles.go` - right border (instead of left), user accent color
- [ ] Right-align message content within available width
- [ ] Use `▐` right border vs assistant's `▌` left border

```go
func (u *UserMessageItem) Render(width int) string {
    // Cap width, wrap text
    effectiveWidth := min(width-2, 120)
    wrapped := wrapText(u.content, effectiveWidth)
    
    // Apply user style with right border
    styled := styleUserMessage.Render(wrapped)
    
    // Right-align the entire block
    return rightAlign(styled, width)
}
```

### 11. AppendUserMessage method

Add method to append user messages to viewport.

- [ ] Add `AppendUserMessage(text string) tea.Cmd` to `agent.go`
- [ ] Generate ID: `fmt.Sprintf("user-%d", time.Now().UnixNano())`
- [ ] Create `UserMessageItem`, append to messages slice
- [ ] Call `refreshContent()` and scroll to bottom

```go
func (a *AgentOutput) AppendUserMessage(text string) tea.Cmd {
    id := fmt.Sprintf("user-%d", time.Now().UnixNano())
    msg := &UserMessageItem{
        id:      id,
        content: text,
    }
    a.messages = append(a.messages, msg)
    a.refreshContent()
    return nil
}
```

### 12. Display user message on delivery

Orchestrator notifies TUI when processing message; TUI appends to viewport.

- [ ] Update `QueuedMessageProcessingMsg` to include message text: `struct { Text string }`
- [ ] In `App.Update()` handle `QueuedMessageProcessingMsg`: call `agentOutput.AppendUserMessage(msg.Text)`
- [ ] Message appears in viewport at the moment agent starts processing it
- [ ] Agent response will appear below the user message

```go
case QueuedMessageProcessingMsg:
    // User message delivered - show in viewport now
    a.queueDepth--
    if a.queueDepth < 0 {
        a.queueDepth = 0
    }
    cmd := a.dashboard.agentOutput.AppendUserMessage(msg.Text)
    return a, tea.Batch(cmd, a.dashboard.SetQueueDepth(a.queueDepth))
```

### 13. Final integration and tests

- [ ] Verify message appears at correct position (when agent receives, not when user sends)
- [ ] Verify agent response appears below user message
- [ ] Add test: send message while agent idle → appears immediately, agent responds below
- [ ] Add test: send message while agent busy → appears when agent processes it
- [ ] Add test: multiple queued messages appear in order as processed
- [ ] Manual test: full conversation flow with interleaved user/agent messages

## UI Mockup

### User sends message while agent busy (message queued, not yet visible)

```
┌─ Agent Output ─────────────────────────────────────────────────┐
│ ▌ Running the build process...                                  │
│ ✓ bash: go build ./...                                          │
│                                                                 │  ← no user message yet
├─────────────────────────────────────────────────────────────────┤
│ > █                                              (1 queued)     │  ← queue indicator only
│ Agent is working...                                             │
└─────────────────────────────────────────────────────────────────┘
```

### Agent finishes, processes queued message (message appears at delivery)

```
┌─ Agent Output ─────────────────────────────────────────────────┐
│ ▌ Build completed successfully.                                 │
│                                                                 │
│                                       fix the bug ▐            │  ← appears NOW
│                                                                 │
│ ▌ I'll fix that bug now. Looking at the error...               │  ← agent responds below
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ > █                                                             │
└─────────────────────────────────────────────────────────────────┘
```

### Multiple messages in conversation (accurate order)

```
┌─ Agent Output ─────────────────────────────────────────────────┐
│ ▌ Hello! How can I help you today?                              │
│                                                                 │
│                           check the logs ▐                     │  ← user message
│                                                                 │
│ ▌ I checked the logs and found 3 errors...                      │  ← agent response
│                                                                 │
│                              fix them all ▐                    │  ← user message
│                                                                 │
│ ▌ I'll fix each error one by one...                            │  ← agent response
│ ✓ edit: src/main.go                                             │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ > █                                                             │
└─────────────────────────────────────────────────────────────────┘
```

### Visual distinction: left border (assistant) vs right border (user)

```
│ ▌ Assistant message has left border                             │
│                                                                 │
│                    User message has right border ▐             │
```

## Out of Scope

- Priority queue (all messages equal priority)
- Message reordering or cancellation
- Persistent queue across restarts
- Queue size limit configuration
- Per-session queues (single session per orchestrator)

## Open Questions

None.

## Resolved

- **Q: Where should queue live - agent layer or orchestrator?**
  A: Orchestrator. We already have `sendChan` there, and runner is stateless per-call. Crush puts it in agent layer because their agent manages multiple sessions; we have one session per orchestrator.

- **Q: Thread-safe map like crush uses `csync.Map`?**
  A: Not needed. Go channel provides thread-safety between TUI and orchestrator goroutines. Processing is single-threaded within orchestrator.

- **Q: Recursive processing like crush or loop?**
  A: Loop. More idiomatic Go, easier to debug, same result.
