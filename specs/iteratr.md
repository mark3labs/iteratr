# iteratr

AI coding agent orchestrator with embedded persistence and TUI.

## Overview

iteratr is a Go CLI tool that orchestrates AI coding agents in an iterative loop. It manages session state (tasks, notes, inbox) via embedded NATS JetStream, communicates with opencode via ACP (Agent Control Protocol) over stdio, and presents a full-screen TUI using Bubbletea v2.

Spiritual successor to ralph.nu - same concepts, modern Go implementation.

## User Story

**As a** developer using AI coding agents  
**I want** an orchestrator that manages iterative agent loops with persistent state  
**So that** I can run multi-iteration coding sessions with task tracking, inter-iteration memory, and human-in-the-loop messaging

## Requirements

### Functional

1. **Session Management**
   - Named sessions derived from spec filename or explicit `--name` flag
   - Session names must be alphanumeric with hyphens/underscores (no dots - breaks NATS subjects)
   - Session continuation (resume from last iteration)
   - Session completion signal (agent calls `session_complete` tool)
   - Data stored in `.iteratr/` directory

2. **Task System** (append-only event log)
   - `task_add(content, status?)` - create task, default status: remaining
   - `task_status(id, status)` - update task status (remaining, in_progress, completed, blocked)
   - `task_list` - list tasks grouped by status
   - Statuses: remaining, in_progress, completed, blocked
   - ID prefix matching (8+ chars) for convenience
   - Track which iteration modified each task

3. **Notes System**
   - `note_add(content, type)` - record learnings/tips/blockers/decisions
   - `note_list(type?)` - list notes, optionally filtered
   - Types: learning, stuck, tip, decision
   - Notes from previous iterations shown in prompt

4. **Inbox System**
   - `message` subcommand to send messages to running session
   - `inbox_list` - get unread messages
   - `inbox_mark_read(id)` - acknowledge message
   - Unread messages injected into iteration prompt

5. **Iteration Loop**
   - Configurable iteration count (0 = infinite)
   - Log iteration start/complete events
   - Check for session_complete signal after each iteration
   - Build prompt from template with current state

6. **Prompt Template System**
   - Default embedded template with `{{variable}}` placeholders
   - Custom template file support (`.iteratr.template` or `--template` flag)
   - Variables: session, iteration, spec, inbox, notes, tasks, extra
   - Extra instructions via `--extra-instructions` flag

7. **TUI (Alt Screen)**
   - Dashboard: session name, current iteration, progress bar, current task
   - Task list: filterable by status, shows task IDs
   - Log viewer: scrollable iteration/event history
   - Notes panel: grouped by type
   - Inbox: view messages, send new messages
   - Agent output: inline streaming in real-time

8. **CLI Commands**
   - `iteratr build` - main agent loop
   - `iteratr doctor` - check dependencies (opencode)
   - `iteratr gen-template` - export default template
   - `iteratr message --name <session> <message>` - send message to session
   - `iteratr version` - show version

### Non-Functional

1. Single self-contained binary (embedded NATS)
2. Cross-platform (Linux, macOS, Windows)
3. Graceful shutdown on SIGINT/SIGTERM
4. Clean alt screen restoration on exit

## Technical Implementation

### Architecture

```
+------------------+       ACP/stdio        +------------------+
|     iteratr      | <-------------------> |     opencode     |
|                  |                       |                  |
|  +------------+  |                       |  +------------+  |
|  | Bubbletea  |  |                       |  |   Agent    |  |
|  |    TUI     |  |                       |  +------------+  |
|  +------------+  |                       +------------------+
|        |         |
|  +------------+  |
|  |    ACP     |  |
|  |   Client   |  |
|  +------------+  |
|        |         |
|  +------------+  |
|  |   NATS     |  |
|  | JetStream  |  |
|  | (embedded) |  |
|  +------------+  |
+------------------+
```

### Package Structure

```
cmd/
  iteratr/
    main.go           # CLI entry point (cobra/kong)
internal/
  acp/
    client.go         # ACP client wrapper using acp-go-sdk
    tools.go          # Tool definitions for agent
  nats/
    server.go         # Embedded NATS server setup
    store.go          # JetStream KV/Stream operations
  session/
    session.go        # Session state management
    task.go           # Task operations (add, status, list)
    note.go           # Note operations
    inbox.go          # Inbox operations
    iteration.go      # Iteration logging
  template/
    template.go       # Prompt template rendering
    default.go        # Default template constant
  tui/
    app.go            # Main Bubbletea model
    dashboard.go      # Dashboard view component
    tasks.go          # Task list component
    logs.go           # Log viewer component
    notes.go          # Notes panel component
    inbox.go          # Inbox component
    agent.go          # Agent output streaming component
    styles.go         # Lipgloss styles
```

### Embedded NATS with JetStream

```go
import (
    "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/jetstream"
)

func StartEmbeddedNATS(dataDir string) (*server.Server, error) {
    opts := &server.Options{
        JetStream:  true,
        StoreDir:   dataDir,
        DontListen: true, // No network ports - in-process only
    }
    ns, err := server.NewServer(opts)
    if err != nil {
        return nil, err
    }
    go ns.Start()
    if !ns.ReadyForConnections(4 * time.Second) {
        return nil, errors.New("nats server failed to start")
    }
    return ns, nil
}

// Use InProcessServer option for true in-process connection (no network)
func ConnectInProcess(ns *server.Server) (*nats.Conn, error) {
    return nats.Connect("", nats.InProcessServer(ns))
}

// Create JetStream context using modern API
func CreateJetStream(nc *nats.Conn) (jetstream.JetStream, error) {
    return jetstream.New(nc)
}
```

#### Stream and Consumer Setup

```go
func SetupStream(ctx context.Context, js jetstream.JetStream, session string) (jetstream.Stream, error) {
    return js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
        Name:     "iteratr_events",
        Subjects: []string{fmt.Sprintf("iteratr.%s.>", session)},
        Storage:  jetstream.FileStorage,
        MaxAge:   30 * 24 * time.Hour, // 30 day retention
    })
}

// Create durable consumer for reading event history
func CreateConsumer(ctx context.Context, stream jetstream.Stream, name string) (jetstream.Consumer, error) {
    return stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
        Durable:       name,
        AckPolicy:     jetstream.AckExplicitPolicy,
        DeliverPolicy: jetstream.DeliverAllPolicy, // Start from beginning
    })
}
```

#### Publishing Events

```go
func (s *Store) PublishEvent(ctx context.Context, event Event) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    subject := fmt.Sprintf("iteratr.%s.%s", event.Session, event.Type)
    _, err = s.js.Publish(ctx, subject, data)
    return err
}
```

#### Reading Event History (Reduce Pattern)

```go
func (s *Store) LoadState(ctx context.Context, session string) (*State, error) {
    consumer, err := s.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
        FilterSubject: fmt.Sprintf("iteratr.%s.>", session),
        DeliverPolicy: jetstream.DeliverAllPolicy,
    })
    if err != nil {
        return nil, err
    }
    
    state := &State{Tasks: make(map[string]*Task)}
    msgs, err := consumer.Fetch(1000) // Batch fetch
    if err != nil {
        return nil, err
    }
    
    for msg := range msgs.Messages() {
        var event Event
        if err := json.Unmarshal(msg.Data(), &event); err != nil {
            continue
        }
        state.Apply(event) // Reduce: fold event into state
        msg.Ack()
    }
    return state, nil
}
```

### JetStream Streams

```
Stream: iteratr_events
  Subjects: iteratr.{session}.>
  
Subject patterns (using NATS wildcards):
  iteratr.{session}.task      - Task events (add, status change)
  iteratr.{session}.note      - Note events
  iteratr.{session}.inbox     - Inbox events (add, mark_read)
  iteratr.{session}.iteration - Iteration events (start, complete)
  iteratr.{session}.control   - Control events (session_complete)

NATS Wildcard Reference:
  *  - Single token wildcard (e.g., iteratr.*.task matches any session)
  >  - Multi-level wildcard, must be last (e.g., iteratr.mysession.> matches all events)

Consumer Filtering:
  FilterSubject: "iteratr.mysession.>"     - All events for a session
  FilterSubject: "iteratr.mysession.task"  - Only task events
  FilterSubjects: ["iteratr.*.task", "iteratr.*.note"] - Multiple patterns
```

### ACP Integration

Using `github.com/coder/acp-go-sdk`:

```go
import "github.com/coder/acp-go-sdk"

// ACPClient implements acp.Client interface for handling agent requests
type ACPClient struct {
    session *session.Session
    tui     *tui.App
    conn    *acp.ClientSideConnection
}

var _ acp.Client = (*ACPClient)(nil)
```

#### Implement Required Client Interface Methods

```go
// Handle file system requests from agent
func (c *ACPClient) ReadTextFile(ctx context.Context, params acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
    content, err := os.ReadFile(params.Path)
    if err != nil {
        return acp.ReadTextFileResponse{}, err
    }
    return acp.ReadTextFileResponse{Content: string(content)}, nil
}

func (c *ACPClient) WriteTextFile(ctx context.Context, params acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
    err := os.WriteFile(params.Path, []byte(params.Content), 0o644)
    return acp.WriteTextFileResponse{}, err
}

// Handle streaming updates - forward to TUI
func (c *ACPClient) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
    u := params.Update
    
    switch {
    case u.AgentMessageChunk != nil:
        // Stream agent text to TUI
        if u.AgentMessageChunk.Content.Text != nil {
            c.tui.Send(AgentOutputMsg{Content: u.AgentMessageChunk.Content.Text.Text})
        }
        
    case u.ToolCall != nil:
        // Tool call initiated - check if it's one of our custom tools
        c.handleToolCall(ctx, u.ToolCall)
        
    case u.ToolCallUpdate != nil:
        // Tool call completed
        c.tui.Send(ToolUpdateMsg{ID: string(u.ToolCallUpdate.ToolCallId), Status: string(*u.ToolCallUpdate.Status)})
    }
    return nil
}

// Handle permission requests
func (c *ACPClient) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
    // Auto-approve or prompt user via TUI
    return acp.RequestPermissionResponse{
        Outcome: acp.PermissionOutcome{
            Selected: &acp.PermissionOutcomeSelected{
                OptionId: params.Options[0].OptionId, // First option = allow
            },
        },
    }, nil
}
```

#### Launch Agent Subprocess and Run Iteration

```go
func (c *ACPClient) RunIteration(ctx context.Context, prompt string) error {
    // Start opencode as subprocess
    cmd := exec.CommandContext(ctx, "opencode", "acp")
    cmd.Stderr = os.Stderr
    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start opencode: %w", err)
    }
    
    // Create client-side connection
    c.conn = acp.NewClientSideConnection(c, stdin, stdout)
    
    // Initialize connection
    initResp, err := c.conn.Initialize(ctx, acp.InitializeRequest{
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
        Cwd: c.session.WorkDir,
    })
    if err != nil {
        return fmt.Errorf("new session failed: %w", err)
    }
    
    // Send prompt and stream response
    _, err = c.conn.Prompt(ctx, acp.PromptRequest{
        SessionId: newSess.SessionId,
        Prompt:    []acp.ContentBlock{acp.TextBlock(prompt)},
    })
    
    return err
}
```

### Tool Exposure via ACP

Tools are exposed dynamically during execution via `SessionUpdate` notifications. iteratr intercepts tool calls from the agent and handles custom session tools:

```go
// Handle tool calls from agent - check for our custom tools
func (c *ACPClient) handleToolCall(ctx context.Context, tc *acp.SessionUpdateToolCall) {
    // Extract tool name from RawInput (agent sends tool name + args)
    input, ok := tc.RawInput.(map[string]any)
    if !ok {
        return
    }
    toolName, _ := input["tool"].(string)
    
    var result any
    var err error
    
    switch toolName {
    case "task_add":
        result, err = c.session.TaskAdd(input)
    case "task_status":
        result, err = c.session.TaskStatus(input)
    case "task_list":
        result, err = c.session.TaskList()
    case "note_add":
        result, err = c.session.NoteAdd(input)
    case "note_list":
        result, err = c.session.NoteList(input)
    case "inbox_list":
        result, err = c.session.InboxList()
    case "inbox_mark_read":
        result, err = c.session.InboxMarkRead(input)
    case "session_complete":
        result, err = c.session.Complete()
        c.sessionComplete = true
    default:
        return // Not our tool, let agent handle it
    }
    
    // Send tool result back via SessionUpdate
    status := acp.ToolCallStatusCompleted
    if err != nil {
        status = acp.ToolCallStatusFailed
    }
    
    c.conn.SessionUpdate(ctx, acp.SessionNotification{
        SessionId: c.currentSessionId,
        Update: acp.UpdateToolCall(
            tc.ToolCallId,
            acp.WithUpdateStatus(status),
            acp.WithUpdateContent([]acp.ToolCallContent{
                acp.ToolContent(acp.TextBlock(formatResult(result, err))),
            }),
            acp.WithUpdateRawOutput(result),
        ),
    })
}
```

#### Tool Definitions for Agent Prompt

Tools are described in the prompt template so the agent knows how to call them:

```go
const toolDescriptions = `
## Available Tools

### Task Management
- task_add(content, status?) - Create task (status: remaining|in_progress|completed|blocked)
- task_status(id, status) - Update task status (id: 8+ char prefix)
- task_list - List all tasks grouped by status

### Notes
- note_add(content, type) - Record note (type: learning|stuck|tip|decision)
- note_list(type?) - List notes, optionally filtered by type

### Inbox
- inbox_list - Get unread messages from human
- inbox_mark_read(id) - Acknowledge message after processing

### Session Control
- session_complete - Signal all tasks done, end iteration loop
`
```

#### Helper for Formatting Tool Results

```go
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
```

### Bubbletea v2 TUI

```go
import tea "github.com/charmbracelet/bubbletea/v2"

type View int
const (
    ViewDashboard View = iota
    ViewTasks
    ViewLogs
    ViewNotes
    ViewInbox
)

type App struct {
    // Views
    dashboard  *Dashboard
    tasks      *TaskList
    logs       *LogViewer
    notes      *NotesPanel
    inbox      *InboxPanel
    agent      *AgentOutput
    
    // State
    activeView View
    session    *session.Session
    width      int
    height     int
    quitting   bool
}

func (a *App) Init() tea.Cmd {
    // In v2, Init returns only tea.Cmd (not Model)
    // AltScreen/Mouse handled in View() via tea.View struct
    return tea.Batch(
        a.subscribeToEvents(),
        a.agent.Init(),
    )
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        k := msg.String()
        // Global navigation
        switch k {
        case "1": a.activeView = ViewDashboard
        case "2": a.activeView = ViewTasks
        case "3": a.activeView = ViewLogs
        case "4": a.activeView = ViewNotes
        case "5": a.activeView = ViewInbox
        case "q", "ctrl+c":
            a.quitting = true
            return a, tea.Quit
        }
    case tea.WindowSizeMsg:
        a.width = msg.Width
        a.height = msg.Height
    case AgentOutputMsg:
        return a, a.agent.Append(msg.Content)
    case IterationStartMsg:
        return a, a.dashboard.SetIteration(msg.N)
    }
    // Delegate to active view component
    var cmd tea.Cmd
    switch a.activeView {
    case ViewDashboard:
        cmd = a.dashboard.Update(msg)
    case ViewTasks:
        cmd = a.tasks.Update(msg)
    case ViewLogs:
        cmd = a.logs.Update(msg)
    case ViewNotes:
        cmd = a.notes.Update(msg)
    case ViewInbox:
        cmd = a.inbox.Update(msg)
    }
    return a, cmd
}

// v2 View() returns tea.View struct with display options
func (a *App) View() tea.View {
    if a.quitting {
        return tea.NewView("Goodbye!\n")
    }
    
    header := a.renderHeader()
    content := a.renderActiveView()
    footer := a.renderFooter()
    output := lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
    
    v := tea.NewView(output)
    v.AltScreen = true                    // Full-screen mode
    v.MouseMode = tea.MouseModeCellMotion // Click/scroll/drag events
    v.ReportFocus = true                  // Focus gain/loss events
    return v
}
```

#### Sending Messages from Goroutines

Use `Program.Send()` for external goroutines (e.g., NATS subscriptions):

```go
func (a *App) subscribeToEvents() tea.Cmd {
    return func() tea.Msg {
        // This runs in a managed goroutine
        sub, _ := a.nc.Subscribe(fmt.Sprintf("iteratr.%s.>", a.session.Name), func(m *nats.Msg) {
            var event Event
            json.Unmarshal(m.Data, &event)
            a.program.Send(EventMsg{Event: event}) // Send to Update loop
        })
        <-a.ctx.Done()
        sub.Unsubscribe()
        return nil
    }
}

// Custom message types
type EventMsg struct{ Event Event }
type AgentOutputMsg struct{ Content string }
type IterationStartMsg struct{ N int }
```

#### Using tea.Batch for Concurrent Commands

```go
func (a *App) Init() tea.Cmd {
    return tea.Batch(
        a.subscribeToEvents(),  // All run concurrently
        a.loadInitialState(),   // No ordering guarantees
        a.agent.Init(),
    )
}
```

### CLI Flags

```
iteratr build [flags]
  -n, --name string              Session name (default: spec filename stem)
  -s, --spec string              Spec file path (default: ./specs/SPEC.md)
  -t, --template string          Custom template file
  -e, --extra-instructions string Extra instructions for prompt
  -i, --iterations int           Max iterations, 0=infinite (default: 0)
      --headless                 Run without TUI (logging only)

iteratr message [flags] <message>
  -n, --name string              Session name (required)

iteratr gen-template [flags]
  -o, --output string            Output file (default: .iteratr.template)

iteratr doctor

iteratr version
```

### Environment Variables

```
ITERATR_DATA_DIR     - Data directory (default: .iteratr)
ITERATR_LOG_FILE     - Log file path for debugging
ITERATR_LOG_LEVEL    - Log level: debug, info, warn, error
```

### Default Prompt Template

```
## Context
Session: {{session}} | Iteration: #{{iteration}}
Spec: {{spec}}
{{inbox}}{{notes}}
## Task State
{{tasks}}

## Tools - all require session_name="{{session}}"
- inbox_list / inbox_mark_read(id) - check/ack messages
- task_add(content, status?) / task_status(id, status) / task_list - manage tasks
- note_add(content, type) / note_list(type?) - record learnings/tips/blockers/decisions
- session_complete - call when ALL tasks done

## Workflow
1. Check inbox, mark read after processing
2. Ensure all spec tasks exist in task list
3. Pick ONE task, mark in_progress, do work, mark completed
4. Run tests, commit with clear message
5. If stuck/learned something: note_add
6. When ALL done: session_complete

Rules: ONE task/iteration. Test before commit. Call session_complete to end.
{{extra}}
```

### Event Message Format

Stored as JSON in JetStream:

```go
type Event struct {
    ID        string          `json:"id"`        // NATS message ID
    Timestamp time.Time       `json:"timestamp"`
    Session   string          `json:"session"`
    Type      string          `json:"type"`      // task, note, inbox, iteration, control
    Action    string          `json:"action"`    // add, status, mark_read, start, complete
    Meta      json.RawMessage `json:"meta"`      // Action-specific data
    Data      string          `json:"data"`      // Content (task text, note text, etc)
}
```

## Tasks

### 1. Project Setup
- [ ] Initialize Go module with dependencies
- [ ] Set up package structure
- [ ] Add Makefile with build/test/lint targets

### 2. Embedded NATS
- [ ] Implement embedded NATS server startup
- [ ] Configure JetStream with file-based storage
- [ ] Create stream and subject helpers
- [ ] Add graceful shutdown

### 3. Session Store
- [ ] Implement event append (generic)
- [ ] Implement event stream reading with reduce pattern
- [ ] Add task operations (add, status, list, get-state)
- [ ] Add note operations (add, list)
- [ ] Add inbox operations (add, mark_read, list-unread)
- [ ] Add iteration logging (start, complete)
- [ ] Add session_complete control event

### 4. ACP Client
- [ ] Integrate acp-go-sdk
- [ ] Implement tool definitions for all session tools
- [ ] Implement agent subprocess management
- [ ] Handle streaming updates from agent
- [ ] Route tool calls to session store

### 5. Prompt Templates
- [ ] Embed default template
- [ ] Implement {{variable}} substitution
- [ ] Load custom template from file
- [ ] Build prompt with current state injection

### 6. TUI Foundation
- [ ] Create main Bubbletea app model
- [ ] Set up alt screen and mouse support
- [ ] Implement view switching (1-5 keys)
- [ ] Add header/footer chrome
- [ ] Define Lipgloss styles

### 7. TUI Dashboard View
- [ ] Display session name and iteration
- [ ] Show progress indicator
- [ ] Display current in_progress task
- [ ] Show task completion stats

### 8. TUI Task List View
- [ ] List tasks grouped by status
- [ ] Show task IDs (8 char prefix)
- [ ] Add status filtering (j/k navigation)
- [ ] Highlight current task

### 9. TUI Log Viewer
- [ ] Subscribe to iteration events
- [ ] Scrollable event history
- [ ] Color-coded by event type
- [ ] Timestamps

### 10. TUI Notes Panel
- [ ] Group notes by type
- [ ] Color-coded type headers
- [ ] Show iteration number per note

### 11. TUI Inbox Panel
- [ ] List unread messages
- [ ] Show message timestamps
- [ ] Input field for sending new messages

### 12. TUI Agent Output
- [ ] Stream agent output in real-time
- [ ] Auto-scroll with manual override
- [ ] Render markdown content

### 13. Iteration Loop
- [ ] Implement main build loop
- [ ] Check session_complete signal
- [ ] Handle iteration limits
- [ ] Continue from last iteration

### 14. CLI Commands
- [ ] Implement `build` command with all flags
- [ ] Implement `message` command
- [ ] Implement `gen-template` command
- [ ] Implement `doctor` command
- [ ] Implement `version` command

### 15. Polish
- [ ] Add headless mode (no TUI)
- [ ] Graceful shutdown cleanup
- [ ] Error handling and recovery
- [ ] Debug logging

### 16. Testing
- [ ] Unit tests for session store operations
- [ ] Unit tests for template rendering
- [ ] Integration test for iteration loop
- [ ] TUI component tests

### 17. Documentation
- [ ] Write README with usage examples
- [ ] Document environment variables
- [ ] Add AGENTS.md for AI agent guidance

## UI Mockup

```
+------------------------------------------------------------------+
|  iteratr v0.1.0 | my-feature | Iteration #3           [1][2][3] |
+------------------------------------------------------------------+
|                                                                  |
|  DASHBOARD                                                       |
|  ─────────────────────────────────────────                       |
|                                                                  |
|  Session:    my-feature                                          |
|  Iteration:  3 of unlimited                                      |
|  Progress:   [████████░░░░░░░░░░░░] 4/10 tasks                   |
|                                                                  |
|  Current Task:                                                   |
|  ┌──────────────────────────────────────────────────────────┐   |
|  │ [a1b2c3d4] Implement user authentication endpoint        │   |
|  └──────────────────────────────────────────────────────────┘   |
|                                                                  |
|  Agent Output:                                                   |
|  ────────────────────────────────────────                        |
|  │ Looking at the authentication requirements...                 |
|  │ I'll start by creating the auth handler in                    |
|  │ internal/api/auth.go                                          |
|  │ ▌                                                             |
|                                                                  |
+------------------------------------------------------------------+
|  [1] Dashboard [2] Tasks [3] Logs [4] Notes [5] Inbox    q=quit |
+------------------------------------------------------------------+
```

## Out of Scope

- Remote NATS server support (embedded only)
- Multiple concurrent sessions
- Model/provider configuration (delegated to opencode)
- ngrok/tunnel support
- Web UI
- Session export/import

## Open Questions

1. Should we support session history browsing (view past sessions)?
2. Should inbox messages persist across session restarts?
3. What's the right behavior when opencode isn't installed?
4. Should we support custom tool definitions beyond the built-in ones?

## Dependencies

```go
require (
    github.com/charmbracelet/bubbletea/v2 v2.0.0-rc.2
    github.com/charmbracelet/lipgloss v0.13.0
    github.com/charmbracelet/bubbles v0.20.0
    github.com/coder/acp-go-sdk v0.6.3
    github.com/nats-io/nats-server/v2 v2.10.0
    github.com/nats-io/nats.go v1.36.0
    github.com/spf13/cobra v1.8.0
)
```

## Resources

### btca Queries

Use `btca ask` for up-to-date implementation patterns:

```bash
# Bubbletea v2 patterns
btca ask -r bubbleteaV2 -q "How do I use tea.View with AltScreen and MouseMode?"
btca ask -r bubbleteaV2 -q "How do I send messages from external goroutines?"
btca ask -r bubbleteaV2 -q "How do I use tea.Batch and create custom Cmds?"

# NATS embedded server + JetStream
btca ask -r natsGo -q "How do I use InProcessServer for embedded NATS?"
btca ask -r natsGo -q "How do I create streams and consumers with jetstream.New?"
btca ask -r natsGo -q "How do I use FilterSubject with wildcards?"

# ACP Go SDK
btca ask -r acpGoSdk -q "How do I create a ClientSideConnection to control an agent?"
btca ask -r acpGoSdk -q "How do I handle SessionUpdate notifications?"
btca ask -r acpGoSdk -q "How do I define and expose custom tools via ACP?"
```

### Key API Patterns

| Component | Pattern | Notes |
|-----------|---------|-------|
| Embedded NATS | `nats.Connect("", nats.InProcessServer(srv))` | No network ports |
| JetStream | `jetstream.New(nc)` then `js.CreateOrUpdateStream()` | Modern API |
| Stream read | `consumer.Fetch(n)` + range over `msgs.Messages()` | Pull consumer |
| Bubbletea v2 | `View()` returns `tea.View` with `.AltScreen`, `.MouseMode` | Display options |
| External msgs | `program.Send(msg)` from goroutines | Thread-safe |
| Concurrent cmds | `tea.Batch(cmd1, cmd2, cmd3)` | No ordering |
| ACP connection | `acp.NewClientSideConnection(client, stdin, stdout)` | Stdio pipes |
| ACP init | `conn.Initialize()` → `conn.NewSession()` → `conn.Prompt()` | Sequence |
| ACP tools | `acp.StartToolCall()` / `acp.UpdateToolCall()` | Via SessionUpdate |
| ACP streaming | Implement `SessionUpdate(ctx, params)` on Client | Notifications |

### ACP Client Interface

Required methods for `acp.Client`:

| Method | Purpose |
|--------|---------|
| `ReadTextFile` | Agent requests file read |
| `WriteTextFile` | Agent requests file write |
| `SessionUpdate` | Receive streaming updates (messages, tools, plans) |
| `RequestPermission` | Agent requests user approval |

### SessionUpdate Variants

| Variant | When Sent |
|---------|-----------|
| `AgentMessageChunk` | Agent streaming text response |
| `AgentThoughtChunk` | Agent internal reasoning |
| `ToolCall` | Tool execution started |
| `ToolCallUpdate` | Tool status/result update |
| `Plan` | Execution plan update |
| `CurrentModeUpdate` | Session mode changed |

### Tool Call Lifecycle

```
ToolCallStatusPending → ToolCallStatusInProgress → ToolCallStatusCompleted
                                                 → ToolCallStatusFailed
```

### Important Notes

- **Bubbletea v2 breaking changes**: `Init()` returns `tea.Cmd` only, `View()` returns `tea.View` struct
- **NATS InProcessServer**: Pass empty string as URL with `InProcessServer` option
- **JetStream consumers**: Use `AckExplicitPolicy` and call `msg.Ack()` after processing
- **Subject wildcards**: `*` = single token, `>` = multi-level (must be last)
- **ACP tools**: No registration needed - tools exposed dynamically via SessionUpdate
- **ACP streaming**: Continue accepting updates after `Cancel()` until stop reason received
