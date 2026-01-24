# Iteratr Component Tree

## Overview
Iteratr is a Go TUI application built with BubbleTea v2 that manages iterative development sessions with an AI agent. The application features a multi-pane interface with real-time updates via NATS messaging, agent output streaming, task/note management, and modal overlays.

## Architecture Pattern
- **Screen/Draw pattern**: Components render directly to screen buffers using Ultraviolet
- **Message-based updates**: State changes propagate via typed messages
- **Lazy rendering**: ScrollList components only render visible items
- **Hierarchical focus management**: Priority-based keyboard routing (Dialog > Modal > Global > View > Focus > Component)

---

## Component Tree

```
App (internal/tui/app.go:17-656)
├── Root BubbleTea Model
├── Implements: tea.Model (Init, Update, View)
├── State Management: session.Store, NATS event subscription
├── Channels: eventChan (NATS events), sendChan (user input to orchestrator)
│
├─── Dashboard (internal/tui/dashboard.go:26-364)
│    ├── Main content area component
│    ├── Implements: FocusableComponent
│    ├── Focus Management: FocusPane enum (FocusAgent, FocusTasks, FocusNotes, FocusInput)
│    ├── Renders: "Agent Output" panel with title bar
│    ├── Child Components:
│    │   └── AgentOutput (shared reference, rendered by Dashboard)
│    └── Message Handling:
│        ├── KeyPress: Tab (cycle focus), i (focus input), Enter/Esc (input control)
│        ├── UserInputMsg → emitted when user submits text
│        └── Focus delegation to child components
│
├─── AgentOutput (internal/tui/agent.go:14-871)
│    ├── Streaming agent conversation display
│    ├── Implements: Component (Draw, Update)
│    ├── Child Components:
│    │   ├── ScrollList (messages viewport)
│    │   ├── textinput.Model (bubbles v2 - user input field)
│    │   └── GradientSpinner (streaming animation)
│    ├── Message Types (internal/tui/messages.go):
│    │   ├── TextMessageItem (assistant text with markdown rendering)
│    │   ├── ThinkingMessageItem (reasoning content, collapsible)
│    │   ├── ToolMessageItem (tool calls with status, expandable)
│    │   ├── InfoMessageItem (model/provider/duration metadata)
│    │   └── DividerMessageItem (iteration separator)
│    ├── Layout: Vertical split (viewport: height-5, input area: 5 lines)
│    ├── Renders:
│    │   ├── ScrollList viewport with message items
│    │   ├── Separator line
│    │   ├── Input field ("> " prompt + text input)
│    │   └── Help text ("Press i to type" or "Enter to send · Esc to cancel")
│    ├── Mouse Interaction:
│    │   ├── Click-to-expand: Toggles expandable messages (ToolMessageItem, ThinkingMessageItem)
│    │   └── Input area click: Focuses text input
│    └── Message Handling:
│        ├── AgentOutputMsg → AppendText()
│        ├── AgentToolCallMsg → AppendToolCall()
│        ├── AgentThinkingMsg → AppendThinking()
│        ├── AgentFinishMsg → AppendFinish()
│        ├── KeyPress: up/down (focus expand/collapse), j/k (vim scroll), space/enter (toggle expand)
│        └── GradientSpinnerMsg → spinner animation updates
│
├─── Sidebar (internal/tui/sidebar.go:162-718)
│    ├── Tasks and notes list display
│    ├── Implements: FocusableComponent
│    ├── Child Components:
│    │   ├── tasksScrollList (task items)
│    │   ├── notesScrollList (note items)
│    │   └── Pulse (animation effect for status changes)
│    ├── Layout: Vertical split (Tasks: 60%, Notes: 40%)
│    ├── Renders:
│    │   ├── Tasks panel: "Tasks" title + ScrollList of taskScrollItem
│    │   └── Notes panel: "Notes" title + ScrollList of noteScrollItem
│    ├── Task Item Format: " [icon] content" (icons: ►=in_progress, ○=remaining, ✓=completed, ⊘=blocked)
│    ├── Note Item Format: " [emoji] content" (emojis: 💡=learning, 🚫=stuck, 💬=tip, ⚡=decision)
│    ├── Mouse Interaction:
│    │   ├── TaskAtPosition() → opens TaskModal
│    │   └── NoteAtPosition() → opens NoteModal
│    ├── Message Handling:
│    │   ├── KeyPress: j/down (cursor down), k/up (cursor up), enter (open modal)
│    │   ├── PulseMsg → pulse animation updates
│    │   ├── StateUpdateMsg → detects task status changes, triggers pulse
│    │   └── OpenTaskModalMsg → emitted when task selected
│    └── State Tracking:
│        ├── taskIndex (ID → position lookup)
│        ├── noteIndex (ID → position lookup)
│        └── pulsedTaskIDs (track status changes)
│
├─── StatusBar (internal/tui/status.go:14-246)
│    ├── Session info and keybinding hints
│    ├── Implements: FullComponent
│    ├── Child Components:
│    │   └── Spinner (bubbles v2 - activity indicator)
│    ├── Layout: Single row at top of screen
│    ├── Renders: "iteratr | session | Iteration #N [spinner]     ctrl+l logs  ctrl+c quit"
│    ├── Left Side: title, session name, iteration number, task stats (✓3 ●1 ○5 ✗1)
│    ├── Right Side: keybinding hints
│    └── Message Handling:
│        ├── StateUpdateMsg → updates task stats, starts/stops spinner
│        ├── SpinnerTickMsg → spinner animation
│        └── ConnectionStatusMsg → updates connection indicator
│
├─── LogViewer (internal/tui/logs.go:14-223) [Modal Overlay]
│    ├── Event history modal
│    ├── Implements: FocusableComponent
│    ├── Child Components:
│    │   └── viewport.Model (bubbles v2)
│    ├── Visibility: Toggled by logsVisible flag in App
│    ├── Renders: Centered modal (80% screen size) with event log
│    ├── Event Format: "HH:MM:SS [TYPE] action data"
│    └── Message Handling:
│        ├── EventMsg → AddEvent() (appends to log, auto-scrolls to bottom)
│        └── KeyPress: esc/ctrl+l (close), up/down (scroll)
│
├─── TaskModal (internal/tui/modal.go:14-280) [Modal Overlay]
│    ├── Task detail view
│    ├── Visibility: Controlled by App.taskModal.visible
│    ├── Renders: Centered modal (60x20) with task details
│    ├── Content: ID, Status badge, Priority badge, Content, Dependencies, Timestamps
│    ├── Mouse Interaction:
│    │   ├── Click outside → closes modal
│    │   └── Click on different task → switches task
│    └── Message Handling:
│        └── KeyPress: esc (close)
│
├─── NoteModal (internal/tui/note_modal.go:12-217) [Modal Overlay]
│    ├── Note detail view
│    ├── Visibility: Controlled by App.noteModal.visible
│    ├── Renders: Centered modal (60x14) with note details
│    ├── Content: ID, Type badge, Content, Timestamp
│    ├── Mouse Interaction:
│    │   ├── Click outside → closes modal
│    │   └── Click on different note → switches note
│    └── Message Handling:
│        └── KeyPress: esc (close)
│
└─── Dialog (internal/tui/dialog.go:10-171) [Modal Overlay]
     ├── Simple confirmation dialog
     ├── Visibility: Controlled by App.dialog.visible
     ├── Renders: Centered rounded border dialog with title, message, OK button
     ├── Used for: Session completion notification
     ├── Mouse Interaction: Click anywhere → dismisses dialog
     └── Message Handling:
         ├── KeyPress: enter/space/esc (close, execute onClose callback)
         └── SessionCompleteMsg → shown when all tasks completed
```

---

## Supporting Components (Non-BubbleTea Models)

### ScrollList (internal/tui/scrolllist.go:21-470)
- **Purpose**: Lazy-rendering scrollable list (only renders visible items)
- **Interface**: ScrollItem (ID(), Render(width), Height())
- **Used By**: AgentOutput, Sidebar (tasks/notes)
- **Features**: Offset-based scrolling, auto-scroll to bottom, keyboard navigation (pgup/pgdown/home/end, j/k)

### Message Items (internal/tui/messages.go)
All implement ScrollItem interface:

| Item | Lines | Purpose |
|------|-------|---------|
| TextMessageItem | 44-101 | Assistant text with markdown rendering via glamour |
| ThinkingMessageItem | 104-204 | Reasoning content, collapsible (last 10 lines when collapsed) |
| ToolMessageItem | 206-453 | Tool execution: header, code output, diffs, expandable |
| InfoMessageItem | 456-528 | Model/provider/duration metadata |
| DividerMessageItem | 531-593 | Iteration separator |

### Animation Components (internal/tui/anim.go)

| Component | Lines | Purpose | Used By |
|-----------|-------|---------|---------|
| Spinner | 13-51 | MiniDot activity indicator | StatusBar |
| Pulse | 54-151 | 5-frame fade in/out effect | Sidebar (task status changes) |
| GradientSpinner | 154-256 | Animated gradient text | AgentOutput ("Generating..."/"Thinking...") |

---

## Message Flow

### Initialization
```
main → Orchestrator.Start()
  → NewApp(ctx, store, sessionName, nc, sendChan)
    → App.Init() → tea.Batch(
        subscribeToEvents(),      // NATS subscription
        waitForEvents(),          // Event channel listener
        loadInitialState(),       // Load session from store
        agent.Init(),             // Initialize AgentOutput
        checkConnectionHealth()   // Periodic health checks
      )
```

### NATS Event Flow
```
NATS Message (iteratr.{session}.>)
  → subscribeToEvents() → eventChan
    → waitForEvents() → EventMsg
      → App.Update(EventMsg)
        ├→ logs.AddEvent(event)
        ├→ loadInitialState()
        └→ waitForEvents()  // Continue listening
```

### State Update Flow
```
loadInitialState()
  → StateUpdateMsg{state}
    → App.Update(StateUpdateMsg)
      ├→ status.SetState(state)
      ├→ sidebar.SetState(state)  // Detects changes → pulse
      ├→ dashboard.UpdateState(state)
      └→ logs.SetState(state)
```

### User Input Flow
```
'i' → Dashboard.Update → focusPane = FocusInput → agent.SetInputFocused(true) → textinput.Focus()
typing → agent.Update → textinput.Update()
Enter → Dashboard.Update → UserInputMsg{text} → App.Update → sendChan <- text → orchestrator → agent
```

### Agent Output Flow
```
Agent runner → orchestrator → NATS/direct
  → App.Update receives:
    ├─ AgentOutputMsg → agent.AppendText()
    ├─ AgentToolCallMsg → agent.AppendToolCall()
    ├─ AgentThinkingMsg → agent.AppendThinking()
    └─ AgentFinishMsg → agent.AppendFinish()
      → ScrollList.SetItems() → auto-scroll
```

---

## Keyboard Routing Priority

```
App.handleKeyPress(KeyPressMsg)
  Priority 0: Dialog visible → Dialog.Update()
  Priority 1: TaskModal visible → ESC closes
  Priority 2: NoteModal visible → ESC closes
  Priority 3: LogViewer visible → ESC/ctrl+l closes, else logs.Update()
  Priority 4: Global keys (ctrl+c quit, ctrl+l logs, ctrl+s sidebar toggle)
  Priority 5: dashboard.Update()
    → 'i' focus input
    → Tab cycle focus
    → Forward to agent (FocusAgent) or sidebar (FocusTasks/FocusNotes)
```

---

## Layout Management

### CalculateLayout() (internal/tui/layout.go)
- **Desktop Mode** (width >= 120): 3-column layout (Status, Main, Sidebar)
- **Compact Mode** (width < 120): 2-row layout (Status, Main), sidebar overlays on toggle

### Resize Flow
```
WindowSizeMsg → App.Update
  → CalculateLayout(width, height) → Layout{Mode, Status, Main, Sidebar}
    → propagateSizes()
      ├→ status.SetSize()
      ├→ dashboard.SetSize() → agent.UpdateSize()
      ├→ logs.SetSize()
      └→ sidebar.SetSize()
```

---

## Rendering Pipeline

```
App.View()
  1. Recalculate layout if dirty
  2. Create screen buffer: uv.NewScreenBuffer(width, height)
  3. Draw in order (back to front):
     ├─ dashboard.Draw(scr, layout.Main)
     ├─ status.Draw(scr, layout.Status)
     ├─ sidebar.Draw(scr, layout.Sidebar)  [desktop mode]
     ├─ logs.Draw(scr, area)               [if visible]
     ├─ taskModal.Draw(scr, area)          [if visible]
     ├─ noteModal.Draw(scr, area)          [if visible]
     └─ dialog.Draw(scr, area)            [if visible]
  4. canvas.Render() → string
  5. Return tea.View{Content, AltScreen, MouseMode}
```

---

## Key Files Reference

| File | Purpose | Lines |
|------|---------|-------|
| `internal/tui/app.go` | Root BubbleTea model, message routing, layout | 656 |
| `internal/tui/dashboard.go` | Main content area, focus management | 364 |
| `internal/tui/agent.go` | Agent conversation display, user input | 871 |
| `internal/tui/sidebar.go` | Tasks/notes lists with pulse animation | 718 |
| `internal/tui/status.go` | Status bar with session info | 246 |
| `internal/tui/logs.go` | Event log modal overlay | 223 |
| `internal/tui/modal.go` | Task detail modal | 280 |
| `internal/tui/note_modal.go` | Note detail modal | 217 |
| `internal/tui/dialog.go` | Simple confirmation dialog | 171 |
| `internal/tui/scrolllist.go` | Lazy-rendering scroll container | 470 |
| `internal/tui/messages.go` | Message item types | 1162 |
| `internal/tui/anim.go` | Animation components | 256 |
| `internal/tui/draw.go` | Drawing utilities | 104 |
| `internal/tui/markdown.go` | Markdown rendering via glamour | 37 |
| `internal/tui/interfaces.go` | Component interfaces | 63 |
| `internal/tui/layout.go` | Layout calculation logic | — |
| `internal/orchestrator/orchestrator.go` | Application orchestrator | — |
