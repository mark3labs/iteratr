# Iteratr Component Tree

## Overview
Iteratr is a Go TUI application built with BubbleTea v2 that manages iterative development sessions with an AI agent. The application features a multi-pane interface with real-time updates via NATS messaging, agent output streaming, task/note management, modal overlays, and subagent session replay.

## Architecture Pattern
- **Screen/Draw pattern**: Components render directly to screen buffers using Ultraviolet
- **Message-based updates**: State changes propagate via typed messages
- **Lazy rendering**: ScrollList components only render visible items
- **Hierarchical focus management**: Priority-based keyboard routing (Dialog > Prefix Mode > Modal > View > Focus > Component)
- **Prefix key sequences**: ctrl+x initiates a two-key command sequence

---

## Component Tree

```
App (internal/tui/app.go:29-1045)
├── Root BubbleTea Model
├── Implements: tea.Model (Init, Update, View)
├── State Management: session.Store, NATS event subscription, Orchestrator control
├── Channels: eventChan (NATS events), sendChan (user input to orchestrator)
├── Prefix Mode: awaitingPrefixKey for ctrl+x sequences
│
├─── Dashboard (internal/tui/dashboard.go:27-325)
│    ├── Main content area component
│    ├── Implements: FocusableComponent
│    ├── Focus Management: FocusPane enum (FocusAgent, FocusTasks, FocusNotes, FocusInput)
│    ├── Renders: "Agent Output" panel with title bar
│    ├── Child Components:
│    │   ├── AgentOutput (shared reference, rendered by Dashboard)
│    │   └── Sidebar (shared reference for focus delegation)
│    └── Message Handling:
│        ├── KeyPress: Tab (cycle focus), i (focus input), Enter/Esc (input control)
│        ├── UserInputMsg → emitted when user submits text
│        └── Focus delegation to child components
│
├─── AgentOutput (internal/tui/agent.go:15-903)
│    ├── Streaming agent conversation display
│    ├── Implements: Component (Draw, Update)
│    ├── Child Components:
│    │   ├── ScrollList (messages viewport)
│    │   ├── textinput.Model (bubbles v2 - user input field)
│    │   └── GradientSpinner (streaming animation)
│    ├── Message Types (internal/tui/messages.go):
│    │   ├── TextMessageItem (assistant text with markdown rendering)
│    │   ├── UserMessageItem (user text, right-aligned)
│    │   ├── ThinkingMessageItem (reasoning content, collapsible)
│    │   ├── ToolMessageItem (tool calls with status, expandable)
│    │   ├── SubagentMessageItem (subagent tasks with session viewer)
│    │   ├── InfoMessageItem (model/provider/duration metadata)
│    │   └── DividerMessageItem (iteration separator)
│    ├── Layout: Vertical split (viewport: height-5, input area: 5 lines)
│    ├── Renders:
│    │   ├── ScrollList viewport with message items
│    │   ├── Separator line
│    │   ├── Input field ("> " prompt + text input + queue indicator)
│    │   └── Help text (context-sensitive hints)
│    ├── Mouse Interaction:
│    │   ├── Click-to-expand: Toggles expandable messages (ToolMessageItem, ThinkingMessageItem)
│    │   ├── Click SubagentMessageItem → opens SubagentModal
│    │   └── Input area click: Focuses text input
│    └── Message Handling:
│        ├── AgentOutputMsg → AppendText()
│        ├── AgentToolCallMsg → AppendToolCall()
│        ├── AgentThinkingMsg → AppendThinking()
│        ├── AgentFinishMsg → AppendFinish()
│        ├── KeyPress: up/down (scroll), j/k (vim scroll), space/enter (toggle expand)
│        └── GradientSpinnerMsg → spinner animation updates
│
├─── Sidebar (internal/tui/sidebar.go:174-878)
│    ├── Tasks and notes list display with logo
│    ├── Implements: FocusableComponent
│    ├── Child Components:
│    │   ├── tasksScrollList (task items)
│    │   ├── notesScrollList (note items)
│    │   └── Pulse (animation effect for status changes)
│    ├── Layout: Vertical split (Logo: 6, Tasks: 55%, Notes: 45% of remainder)
│    ├── Renders:
│    │   ├── Logo panel: gradient-colored "iteratr" ASCII art
│    │   ├── Tasks panel: "Tasks" title + ScrollList of taskScrollItem
│    │   └── Notes panel: "Notes" title + ScrollList of noteScrollItem
│    ├── Task Item Format: " [icon] content" (icons: ►=in_progress, ○=remaining, ✓=completed, ⊘=blocked, ⊗=cancelled)
│    ├── Note Item Format: " [icon] content" (icons: *=learning, !=stuck, ›=tip, ◇=decision)
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
├─── StatusBar (internal/tui/status.go:18-374)
│    ├── Session info, git status, and keybinding hints
│    ├── Implements: FullComponent
│    ├── Child Components:
│    │   └── Spinner (bubbles v2 - activity indicator)
│    ├── Layout: Single row at top of screen
│    ├── Renders: "iteratr | session | branch* hash | H:MM:SS | Iteration #N | stats | [spinner] | PAUSED/PAUSING | hints"
│    ├── Left Side: title, session name, git info, duration, iteration number, task stats, file count, spinner, pause state
│    ├── Right Side: keybinding hints (ctrl+x p pause, ctrl+x l logs, ctrl+c quit)
│    ├── Prefix Mode: Shows "(awaiting key...)" when waiting for second key
│    └── Message Handling:
│        ├── StateUpdateMsg → updates task stats, starts/stops spinner
│        ├── DurationTickMsg → updates elapsed time display
│        ├── PauseStateMsg → updates pause indicator
│        ├── AgentBusyMsg → determines PAUSING vs PAUSED display
│        ├── GitInfoMsg → updates git status display
│        └── ConnectionStatusMsg → updates connection indicator
│
├─── LogViewer (internal/tui/logs.go:15-223) [Modal Overlay]
│    ├── Event history modal
│    ├── Implements: FocusableComponent
│    ├── Child Components:
│    │   └── viewport.Model (bubbles v2)
│    ├── Visibility: Toggled by ctrl+x l
│    ├── Renders: Centered modal (80% screen size) with event log
│    ├── Event Format: "HH:MM:SS [TYPE] action data"
│    └── Message Handling:
│        ├── EventMsg → AddEvent() (appends to log, auto-scrolls to bottom)
│        └── KeyPress: esc (close), up/down (scroll)
│
├─── TaskModal (internal/tui/modal.go:15-303) [Modal Overlay]
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
├─── NoteModal (internal/tui/note_modal.go:12-220) [Modal Overlay]
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
├─── NoteInputModal (internal/tui/note_input_modal.go:14-512) [Modal Overlay]
│    ├── Interactive note creation modal
│    ├── Visibility: Controlled by ctrl+x n
│    ├── Child Components:
│    │   └── textarea.Model (bubbles v2 - multi-line input)
│    ├── Focus Zones: focusTypeSelector, focusTextarea, focusSubmitButton
│    ├── Renders: Centered modal with type badges, textarea, submit button
│    ├── Type Selection: learning, stuck, tip, decision (cycle with left/right)
│    ├── Mouse Interaction:
│    │   ├── Button click → submits note
│    │   └── Click outside → closes modal
│    └── Message Handling:
│        ├── KeyPress: Tab/Shift+Tab (cycle focus), left/right (cycle type)
│        ├── KeyPress: ctrl+enter (submit), esc (close)
│        └── CreateNoteMsg → emitted on submit
│
├─── TaskInputModal (internal/tui/task_input_modal.go:14-443) [Modal Overlay]
│    ├── Interactive task creation modal
│    ├── Visibility: Controlled by ctrl+x t
│    ├── Child Components:
│    │   └── textarea.Model (bubbles v2 - multi-line input)
│    ├── Focus Zones: focusPrioritySelector, focusTextarea, focusSubmitButton
│    ├── Renders: Centered modal with priority badges, textarea, submit button
│    ├── Priority Selection: critical, high, medium, low, backlog (cycle with left/right)
│    ├── Mouse Interaction:
│    │   ├── Button click → submits task
│    │   └── Click outside → closes modal
│    └── Message Handling:
│        ├── KeyPress: Tab/Shift+Tab (cycle focus), left/right (cycle priority)
│        ├── KeyPress: ctrl+enter (submit), esc (close)
│        └── CreateTaskMsg → emitted on submit
│
├─── SubagentModal (internal/tui/subagent_modal.go:17-630) [Modal Overlay]
│    ├── Full-screen subagent session viewer
│    ├── Visibility: Opened by clicking SubagentMessageItem with sessionID
│    ├── Child Components:
│    │   ├── ScrollList (message replay viewport)
│    │   ├── GradientSpinner (loading state)
│    │   └── SessionLoader (ACP subprocess for session replay)
│    ├── States: loading (spinner), error (message), content (scroll list)
│    ├── Renders: Full-screen modal with subagent conversation replay
│    ├── Mouse Interaction:
│    │   └── Click-to-expand: Toggles expandable messages
│    └── Message Handling:
│        ├── SubagentTextMsg → appendText()
│        ├── SubagentToolCallMsg → appendToolCall()
│        ├── SubagentThinkingMsg → appendThinking()
│        ├── SubagentUserMsg → appendUserMessage()
│        ├── SubagentDoneMsg → session replay complete
│        ├── SubagentErrorMsg → displays error
│        └── KeyPress: esc (close), up/down (scroll)
│
└─── Dialog (internal/tui/dialog.go:10-172) [Modal Overlay]
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

### ScrollList (internal/tui/scrolllist.go:21-480)
- **Purpose**: Lazy-rendering scrollable list (only renders visible items)
- **Interface**: ScrollItem (ID(), Render(width), Height())
- **Used By**: AgentOutput, Sidebar (tasks/notes), SubagentModal
- **Features**: Offset-based scrolling, auto-scroll to bottom, keyboard navigation (pgup/pgdown/home/end, j/k), selection highlighting

### Message Items (internal/tui/messages.go)
All implement ScrollItem interface:

| Item | Lines | Purpose |
|------|-------|---------|
| TextMessageItem | 45-110 | Assistant text with markdown rendering via glamour |
| UserMessageItem | 53-163 | User text, right-aligned with border |
| ThinkingMessageItem | 165-266 | Reasoning content, collapsible (last 10 lines when collapsed) |
| ToolMessageItem | 267-552 | Tool execution: header, code output, diffs, expandable |
| SubagentMessageItem | 621-738 | Subagent task: spinner, status, click-to-view hint |
| InfoMessageItem | 553-619 | Model/provider/duration metadata |
| DividerMessageItem | 740-797 | Iteration separator |

### Animation Components (internal/tui/anim.go)

| Component | Lines | Purpose | Used By |
|-----------|-------|---------|---------|
| Spinner | 12-52 | MiniDot activity indicator | StatusBar, SubagentMessageItem |
| Pulse | 54-153 | 5-frame fade in/out effect | Sidebar (task status changes) |
| GradientSpinner | 155-229 | Animated gradient text | AgentOutput, SubagentModal ("Generating..."/"Thinking..."/"Loading...") |

### NotesPanel (internal/tui/notes.go:15-223)
- **Purpose**: Dedicated notes view (grouped by type)
- **Used By**: Potential future view switch
- **Features**: Color-coded type headers, word wrapping, viewport scrolling

### Footer (internal/tui/footer.go:12-239)
- **Purpose**: Navigation footer bar (not currently used in main app)
- **Features**: View navigation hints, clickable buttons, condensed mode for narrow terminals

---

## Message Flow

### Initialization
```
main → Orchestrator.Start()
  → NewApp(ctx, store, sessionName, workDir, nc, sendChan, orchestrator)
    → App.Init() → tea.Batch(
        subscribeToEvents(),      // NATS subscription
        waitForEvents(),          // Event channel listener
        loadInitialState(),       // Load session from store
        agent.Init(),             // Initialize AgentOutput
        checkConnectionHealth(),  // Periodic health checks
        status.StartDurationTick(), // Start elapsed time timer
        fetchGitInfo()            // Fetch git repository status
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

### Pause/Resume Flow
```
ctrl+x p → togglePause()
  → orchestrator.IsPaused() check
    ├→ Not paused: orchestrator.RequestPause() → PauseStateMsg{true}
    ├→ Paused + busy: orchestrator.CancelPause() → PauseStateMsg{false}
    └→ Paused + idle: orchestrator.Resume() → PauseStateMsg{false}
  → StatusBar displays PAUSING... (agent busy) or PAUSED (agent idle)
```

### Subagent Modal Flow
```
Click SubagentMessageItem with sessionID
  → OpenSubagentModalMsg{sessionID, subagentType}
    → NewSubagentModal() → subagentModal.Start()
      → agent.NewSessionLoader() → loader.LoadAndStream()
        → streamNext() → SubagentTextMsg/SubagentToolCallMsg/...
          → HandleUpdate() → append to messages → refreshContent()
            → SubagentDoneMsg (on EOF)
ESC → subagentModal.Close() → subagentModal = nil
```

---

## Keyboard Routing Priority

```
App.handleKeyPress(KeyPressMsg)
  Priority 0: Global keys (ctrl+x prefix, ctrl+c quit) - work everywhere, even with modals
  Priority 1: Dialog visible → Dialog.Update() (enter/space/esc closes)
  Priority 2: Prefix mode (ctrl+x followed by l/b/n/t/p)
    → ctrl+x l: toggle logs
    → ctrl+x b: toggle sidebar
    → ctrl+x n: create note (opens NoteInputModal)
    → ctrl+x t: create task (opens TaskInputModal)
    → ctrl+x p: toggle pause/resume
  Priority 3: TaskModal visible → ESC closes
  Priority 4: NoteModal visible → ESC closes
  Priority 5: NoteInputModal visible → forward to modal Update()
  Priority 6: TaskInputModal visible → forward to modal Update()
  Priority 7: SubagentModal visible → ESC closes, else forward scroll keys
  Priority 8: LogViewer visible → ESC closes, else logs.Update()
  Priority 9: dashboard.Update()
    → 'i' focus input
    → Tab cycle focus
    → Forward to agent (FocusAgent) or sidebar (FocusTasks/FocusNotes)
```

---

## Layout Management

### CalculateLayout() (internal/tui/layout.go:43-85)
- **Desktop Mode** (width >= 100, height >= 25): 3-column layout (Status, Main, Sidebar)
- **Compact Mode** (width < 100 or height < 25): 2-row layout (Status, Main), sidebar overlays on toggle

### Layout Constants
- `CompactWidthBreakpoint`: 100 chars
- `CompactHeightBreakpoint`: 25 rows
- `SidebarWidthDesktop`: 45 chars
- `StatusHeight`: 1 row

### Resize Flow
```
WindowSizeMsg → App.Update
  → CalculateLayout(width, height) → Layout{Mode, Status, Main, Sidebar}
    → propagateSizes()
      ├→ status.SetSize() + status.SetLayoutMode()
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
     ├─ sidebar.Draw(scr, layout.Sidebar)  [desktop mode or sidebarVisible]
     ├─ logs.Draw(scr, area)               [if logsVisible]
     ├─ subagentModal.Draw(scr, area)      [if subagentModal != nil]
     ├─ taskModal.Draw(scr, area)          [if visible]
     ├─ noteModal.Draw(scr, area)          [if visible]
     ├─ noteInputModal.Draw(scr, area)     [if visible]
     ├─ taskInputModal.Draw(scr, area)     [if visible]
     └─ dialog.Draw(scr, area)             [if visible]
  4. canvas.Render() → string
  5. Return tea.View{Content, AltScreen, MouseMode, BackgroundColor}
```

---

## Key Files Reference

| File | Purpose | Lines |
|------|---------|-------|
| `internal/tui/app.go` | Root BubbleTea model, message routing, layout | 1045 |
| `internal/tui/dashboard.go` | Main content area, focus management | 325 |
| `internal/tui/agent.go` | Agent conversation display, user input | 903 |
| `internal/tui/sidebar.go` | Tasks/notes lists with logo and pulse animation | 878 |
| `internal/tui/status.go` | Status bar with session/git info, pause state | 374 |
| `internal/tui/logs.go` | Event log modal overlay | 223 |
| `internal/tui/modal.go` | Task detail modal | 303 |
| `internal/tui/note_modal.go` | Note detail modal | 220 |
| `internal/tui/note_input_modal.go` | Note creation modal with textarea | 512 |
| `internal/tui/task_input_modal.go` | Task creation modal with textarea | 443 |
| `internal/tui/subagent_modal.go` | Subagent session viewer modal | 630 |
| `internal/tui/dialog.go` | Simple confirmation dialog | 172 |
| `internal/tui/scrolllist.go` | Lazy-rendering scroll container | 480 |
| `internal/tui/messages.go` | Message item types for conversation display | 1488 |
| `internal/tui/anim.go` | Animation components (Spinner, Pulse, GradientSpinner) | 229 |
| `internal/tui/draw.go` | Drawing utilities (DrawText, DrawStyled, DrawPanel) | 122 |
| `internal/tui/hints.go` | Keybinding hint rendering utilities | 118 |
| `internal/tui/markdown.go` | Markdown rendering via glamour | 37 |
| `internal/tui/styles.go` | Modal title rendering with gradient | 35 |
| `internal/tui/notes.go` | NotesPanel component (grouped notes view) | 223 |
| `internal/tui/footer.go` | Footer navigation bar (unused in main app) | 239 |
| `internal/tui/interfaces.go` | Component interfaces | 63 |
| `internal/tui/layout.go` | Layout calculation logic | 85 |
| `internal/tui/theme/` | Theme system (manager, styles, catppuccin) | — |
| `internal/tui/wizard/` | Setup wizard components | — |
| `internal/tui/setup/` | Setup flow components | — |
