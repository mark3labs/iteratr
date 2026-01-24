# User Note Creation Modal

Ctrl+N opens a modal with text input and type selector, allowing users to create notes persisted via the same NATS event mechanism the agent uses.

## Overview

Add an interactive modal triggered by ctrl+n that contains a textarea for note content and a type selector (learning/stuck/tip/decision). On submit, the note is published to NATS JetStream via `Store.NoteAdd()`, associating it with the current iteration. The modal closes on submit; state propagation handles sidebar updates.

## User Story

**As a** developer observing an agent session  
**I want** to quickly add my own notes to the session  
**So that** I can record observations, decisions, or tips that persist alongside agent notes

## Requirements

### Functional

1. **Trigger**: Ctrl+N opens modal (blocked when another modal/dialog is visible)
2. **Type Selector**: Cycle through learning/stuck/tip/decision (left/right arrows when focused)
3. **Content Input**: Multi-line textarea with word wrap
4. **Submit Button**: Clickable button; also activated via Enter/Space when focused
5. **Submit Shortcut**: Ctrl+Enter submits from any focus zone
6. **Focus Cycling**: Tab/Shift+Tab cycles focus: type selector → textarea → submit button
7. **Cancel**: ESC closes modal without saving, clears input state
8. **Validation**: Submit blocked if content is empty (button visually dimmed)

### Non-Functional

1. Modal blocks all other keyboard input while open
2. No flicker on open/close
3. Textarea supports at least 500 characters

## Technical Implementation

### Architecture

```
Ctrl+N keypress
└── App.handleGlobalKeys()
    └── Opens NoteInputModal (if no other modal visible)
        ├── Tab/Shift+Tab cycles focus: TypeSelector → TextArea → SubmitButton
        ├── TypeSelector (left/right cycles: learning → stuck → tip → decision)
        ├── TextArea (Bubbles textarea component)
        └── Submit (ctrl+enter from any zone, Enter/Space on button, or click button)
            └── CreateNoteMsg → App.Update()
                └── Store.NoteAdd(content, type, iteration)
                    └── NATS event published
                        └── State rebuilt → sidebar updates
```

### Reference: Bubbles Textarea API

```go
import (
    "charm.land/bubbles/v2/textarea"
    tea "charm.land/bubbletea/v2"
)

// Creation and configuration
ta := textarea.New()
ta.SetWidth(60)
ta.SetHeight(6)
ta.Placeholder = "Enter your note..."
ta.CharLimit = 500
ta.ShowLineNumbers = false
ta.Prompt = ""  // No prompt character
cmd := ta.Focus()

// In Update - forward messages to textarea
var cmd tea.Cmd
m.textarea, cmd = m.textarea.Update(msg)

// Get content
content := m.textarea.Value()

// Reset
m.textarea.SetValue("")

// Render
output := m.textarea.View()
```

**Key bindings of note**: Enter inserts newline by default (`KeyMap.InsertNewline`).
To use ctrl+enter for submit, intercept `"ctrl+enter"` before forwarding to textarea.

### Reference: Textarea Styling (from Crush)

```go
import "charm.land/bubbles/v2/textarea"

s.TextArea = textarea.Styles{
    Focused: textarea.StyleState{
        Base:        base,
        Text:        base,
        Placeholder: base.Foreground(fgSubtle),
        Prompt:      base.Foreground(tertiary),
    },
    Blurred: textarea.StyleState{
        Base:        base,
        Text:        base.Foreground(fgMuted),
        Placeholder: base.Foreground(fgSubtle),
        Prompt:      base.Foreground(fgMuted),
    },
    Cursor: textarea.CursorStyle{
        Color: secondary,
        Shape: tea.CursorBlock,
        Blink: true,
    },
}
```

Apply styles: `ta.SetStyles(styles)`

### New Component

```go
type focusZone int

const (
    focusTypeSelector focusZone = iota
    focusTextarea
    focusSubmitButton
)

type NoteInputModal struct {
    visible   bool
    textarea  textarea.Model  // Bubbles v2 textarea
    noteType  string          // Current selected type
    types     []string        // ["learning","stuck","tip","decision"]
    typeIndex int             // Current type index
    focus     focusZone       // Which zone has keyboard focus
    width     int
    height    int
    buttonArea uv.Rectangle  // Hit area for mouse click on submit button
}
```

### Messages

```go
type CreateNoteMsg struct {
    Content   string
    NoteType  string
    Iteration int
}
```

### Key Bindings (inside modal)

| Key | Context | Action |
|-----|---------|--------|
| Tab | Any | Cycle focus: type selector → textarea → submit button |
| Shift+Tab | Any | Cycle focus backward |
| Left / Right | Type selector focused | Cycle note type |
| Enter / Space | Submit button focused | Submit note |
| Ctrl+Enter | Any focus | Submit note (shortcut) |
| ESC | Any | Cancel and close |
| All other keys | Textarea focused | Forwarded to textarea |

### App Integration

- Add `noteInputModal *NoteInputModal` field to App
- Add `"ctrl+n"` case in `handleGlobalKeys()` (guard: no modal/dialog visible)
- Add priority check in `handleKeyPress()` between NoteModal and LogViewer
- Draw in `App.Draw()` after NoteModal, before Dialog
- On `CreateNoteMsg`: call `Store.NoteAdd()`, close modal

## Tasks

### 1. Tracer bullet: minimal end-to-end

- [ ] Enable `KeyboardEnhancements` in `App.View()` (required for ctrl+enter to work)
- [ ] Add `iteration int` field to App, set it in `IterationStartMsg` handler
- [ ] Create `internal/tui/note_input_modal.go` with struct, `New()`, `IsVisible()`, `Show()`, `Close()`
- [ ] Add Bubbles textarea, hardcode type to "learning", focus starts on textarea
- [ ] Add submit button rendering (static text for now, no focus/click yet)
- [ ] Wire ctrl+n in `handleGlobalKeys()` to call `Show()` and return `textarea.Focus()` cmd
- [ ] Handle ctrl+enter to emit `CreateNoteMsg` and close modal
- [ ] Handle ESC to close without saving
- [ ] Add `CreateNoteMsg` handler in `App.Update()` that calls `Store.NoteAdd()`
- [ ] Add `Draw()` method with `styleModalContainer`, render in `App.Draw()`
- [ ] Add priority routing in `handleKeyPress()` to forward keys to modal when visible

### 2. Focus cycling and submit button

- [ ] Add `focusZone` type and `focus` field to struct
- [ ] Implement Tab/Shift+Tab to cycle focus: type selector → textarea → button
- [ ] Blur textarea when focus leaves it, re-focus when it returns
- [ ] Render button with focused/unfocused/disabled styles
- [ ] Handle Enter/Space on button focus to submit
- [ ] Store `buttonArea uv.Rectangle` during Draw for click hit detection
- [ ] Handle mouse click on button area to submit

### 3. Type selector

- [ ] Add `types []string`, `typeIndex int` fields
- [ ] Render type badges row above textarea (highlight active type with badge style)
- [ ] Handle Left/Right arrows when type selector is focused to cycle `typeIndex`
- [ ] Pass selected type into `CreateNoteMsg`

### 4. Polish textarea

- [ ] Configure textarea: multi-line, word wrap, character limit, placeholder text
- [ ] Override textarea KeyMap: remove ctrl+n from LineNext (use only down arrow)
- [ ] Size textarea to fill modal content area (responsive to terminal size)
- [ ] Style textarea to match modal theme (background, cursor color)

### 5. Validation and UX

- [ ] Block submit when content is whitespace-only (dim button, ignore ctrl+enter)
- [ ] Clear textarea and reset type/focus on close (both cancel and submit)
- [ ] Show hint bar at modal bottom: tab/ctrl+enter/esc

### 6. Guard and edge cases

- [ ] Guard ctrl+n: no-op if dialog, task modal, note modal, or log viewer visible
- [ ] Guard ctrl+n: no-op if session has no active iteration (no iteration to tag)
- [ ] Handle terminal resize while modal is open (recalculate dimensions)

## UI Mockup

```
╭─ New Note ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╮
│                                                  │
│  Type: [learning]  stuck   tip   decision        │
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │ The agent keeps retrying the same approach │  │
│  │ for parsing. Should try regex instead of   │  │
│  │ manual string splitting.                   │  │
│  │                                            │  │
│  │                                            │  │
│  └────────────────────────────────────────────┘  │
│                                                  │
│                                    [ Save Note ] │
│                                                  │
│  tab cycle focus · ctrl+enter submit · esc close │
╰──────────────────────────────────────────────────╯
```

Button states:
- **Focused**: `[ Save Note ]` with highlighted border/background
- **Unfocused**: `  Save Note  ` dimmed
- **Disabled** (empty content): `  Save Note  ` muted, non-interactive

## Gotchas

### 1. ctrl+enter requires keyboard enhancements

The app does NOT currently enable `KeyboardEnhancements` in the `View()` return. Without this, many terminals cannot distinguish `ctrl+enter` from `enter`. To fix, enable in `View()`:

```go
view.KeyboardEnhancements = tea.KeyboardEnhancements{
    ReportEventTypes: true,
}
```

Without this, ctrl+enter will not work as a distinct key event. This must be done in the tracer bullet task.

### 2. No iteration getter on App

`Dashboard.iteration` is unexported with only a setter. The App needs the current iteration to pass to `Store.NoteAdd()`. Either:
- Add `Iteration() int` getter to Dashboard, or
- Track iteration directly on App (simpler: set it in the `IterationStartMsg` handler)

### 3. handleGlobalKeys() return value pattern

`handleGlobalKeys()` currently returns `nil` for ctrl+l/ctrl+s (toggle booleans). The caller only early-returns if cmd is non-nil (line 250). For ctrl+n, the handler **must return a non-nil cmd** (e.g., the `textarea.Focus()` cmd) to prevent the keypress from falling through to `dashboard.Update()`.

### 4. Sidebar updates are automatic

After `Store.NoteAdd()` publishes to NATS, the existing event subscription (`subscribeToEvents`) picks it up → triggers `loadInitialState()` → sends `StateUpdateMsg` → calls `sidebar.SetState()`. No manual sidebar refresh needed.

### 5. Textarea default KeyMap conflict

The Bubbles textarea binds `ctrl+n` to `LineNext` (move cursor down). This is harmless: once the modal is open, ctrl+n goes to the modal's Update which can forward it to the textarea (acts as down-arrow). But users may find it unexpected. Consider disabling it:

```go
ta.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))
```

## Out of Scope

- Edit existing notes
- Delete notes
- Note search/filter from modal
- Rich text / markdown preview in textarea
- File attachments
- Note templates
