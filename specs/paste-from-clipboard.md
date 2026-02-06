## Overview

Enable paste-from-clipboard in every text input/textarea across the entire app. Fix message routing so `tea.PasteMsg` reaches focused inputs in modals and wizards. Add paste sanitization, single-line newline stripping, char limit truncation with toast notification.

## User Story

As a user, I want to paste text from my OS clipboard into any input field (task modal, note modal, chat input, wizard fields) so I can quickly enter pre-composed content without retyping.

## Requirements

- **R1**: `tea.PasteMsg` must reach the focused textarea/textinput in ALL contexts: dashboard, modals (task, note, subagent), wizards (setup, spec)
- **R2**: Pasting multi-line text into single-line inputs → collapse all newlines to single space
- **R3**: Pasted text exceeding char limits → truncate to fit, show toast notification ("N chars truncated")
- **R4**: Sanitize pasted content: strip ANSI escapes, null bytes, non-printable control chars (keep newlines/tabs), normalize CRLF→LF, trim trailing whitespace
- **R5**: Paste is no-op when non-text element (button, selector) is focused
- **R6**: Minimal toast component for truncation feedback: bottom-right overlay, auto-dismiss 3s
- **R7**: Routing fix only (bracketed paste) — no ctrl+v/tea.ReadClipboard() fallback
- **R8**: No copy-to-clipboard support (separate feature)

## Technical Implementation

### Root Cause

`App.Update` (app.go:131) switches on `msg.(type)`. `tea.KeyPressMsg` routes to `handleKeyPress()` which has the 9-level priority hierarchy including modal forwarding (lines 496-503). But `tea.PasteMsg` is a different type — it falls to the default case (line 390) which only forwards to `dashboard.Update()`, **skipping modals and wizards entirely**.

### Fix: Add PasteMsg Routing in App.Update

Add a `tea.PasteMsg` case in `App.Update` that mirrors the modal/component priority from `handleKeyPress`:

```
case tea.PasteMsg:
    1. Sanitize content (strip control chars, normalize CRLF, trim)
    2. If noteInputModal visible → forward to noteInputModal.Update(sanitized)
    3. If taskInputModal visible → forward to taskInputModal.Update(sanitized)
    4. If subagentModal visible → forward to subagentModal.Update(sanitized)
    5. If logsVisible → no-op (no text input in logs)
    6. Else → forward to dashboard.Update(sanitized)
```

### Paste Sanitization (new helper)

`internal/tui/paste.go` — `SanitizePaste(content string) string`:
- Strip ANSI escape sequences (`\x1b\[...m` etc.)
- Remove null bytes and non-printable control chars (except `\n`, `\t`, `\r`)
- Normalize `\r\n` → `\n`
- Trim trailing whitespace

### Single-line Newline Collapsing

In each `textinput.Model` wrapper's Update handler, intercept `tea.PasteMsg` and collapse newlines before forwarding to the Bubbles component:
- Replace `\n+` with single space
- Applies to: agent chat input, model selector search, config step inputs, title step input

### Char Limit Truncation + Toast

In each textarea/textinput wrapper's Update handler:
- Before forwarding PasteMsg to Bubbles component, check if `len(existing) + len(paste) > charLimit`
- If exceeding: truncate paste content, emit `ShowToastMsg{Text: "N chars truncated"}`
- Forward truncated content to Bubbles component

### Toast Component

`internal/tui/toast.go` — minimal overlay:
- Single message string, auto-dismiss after 3s via `tea.Tick`
- Rendered bottom-right of screen, above status bar
- `Show(msg string)` / `Update(msg)` / `View(width, height) string`
- Composed into `App.View()` as overlay

### Affected Components

| Component | File | Type | Changes needed |
|-----------|------|------|----------------|
| App | app.go | Router | Add PasteMsg case with modal priority routing |
| TaskInputModal | task_input_modal.go | textarea | Handle PasteMsg when focusTextarea, truncate+toast |
| NoteInputModal | note_input_modal.go | textarea | Handle PasteMsg when focusTextarea, truncate+toast |
| AgentOutput | agent.go | textinput | Collapse newlines in PasteMsg before forwarding |
| Dashboard | dashboard.go | Router | Forward PasteMsg to focused pane |
| DescriptionStep | specwizard/description_step.go | textarea | Truncate+toast |
| TitleStep | specwizard/title_step.go | textinput | Collapse newlines |
| ModelSelectorStep | wizard/model_selector.go | textinput | Collapse newlines |
| ConfigStep | wizard/config_step.go | textinput | Collapse newlines |

## Tasks

### 1. Paste sanitization helper
- [ ] [P0] Create `internal/tui/paste.go` with `SanitizePaste(string) string` — strip ANSI escapes, null bytes, non-printable control chars, normalize CRLF→LF, trim trailing whitespace
- [ ] [P0] Create `internal/tui/paste_test.go` — unit tests for ANSI stripping, null byte removal, CRLF normalization, trailing whitespace trimming, preserving valid newlines/tabs

### 2. Add PasteMsg routing in App.Update
- [ ] [P0] Add `tea.PasteMsg` case in `App.Update` switch (app.go) — sanitize content, route to visible modal first, then dashboard (mirror handleKeyPress priority order)
- [ ] [P1] Add unit test verifying PasteMsg reaches noteInputModal when visible
- [ ] [P1] Add unit test verifying PasteMsg reaches taskInputModal when visible
- [ ] [P1] Add unit test verifying PasteMsg reaches dashboard when no modal visible

### 3. Single-line newline collapsing
- [ ] [P1] In `AgentOutput.Update` (agent.go) — intercept PasteMsg, collapse `\n+` to single space before forwarding to textinput
- [ ] [P2] In `ModelSelectorStep.Update` (wizard/model_selector.go) — same newline collapsing
- [ ] [P2] In `ConfigStep.Update` (wizard/config_step.go) — same newline collapsing for both inputs
- [ ] [P2] In `TitleStep.Update` (specwizard/title_step.go) — same newline collapsing
- [ ] [P1] Unit tests for newline collapsing in single-line inputs (agent input + one wizard input)

### 4. Minimal toast component
- [ ] [P1] Create `internal/tui/toast.go` — toast model with Show(msg), Update(tick), View(w,h), auto-dismiss after 3s
- [ ] [P1] Create `internal/tui/toast_test.go` — unit tests for show/dismiss timing, rendering
- [ ] [P1] Integrate toast into App: add field, compose in View() as bottom-right overlay, forward tick msgs

### 5. Char limit truncation with toast
- [ ] [P1] In `TaskInputModal.Update` — intercept PasteMsg when focusTextarea, check char limit (500), truncate + emit ShowToastMsg if exceeded, forward to textarea
- [ ] [P1] In `NoteInputModal.Update` — same truncation logic (500 char limit)
- [ ] [P2] In `DescriptionStep.Update` — same truncation logic (5000 char limit)
- [ ] [P1] Unit tests for truncation in task modal (paste within limit, paste exceeding limit, paste into partially filled textarea)

### 6. Integration tests
- [ ] [P2] teatest integration test: paste into task modal textarea, verify content appears
- [ ] [P2] teatest integration test: paste multi-line into agent chat input, verify newlines collapsed
- [ ] [P2] teatest integration test: paste exceeding char limit, verify truncation + toast appears

## UI Mockup

Normal paste — no visible change, text appears at cursor in focused input.

Truncation toast (bottom-right, above status bar):
```
┌─────────────────────────────────────────────────┐
│                                                 │
│  [main app content]                             │
│                                                 │
│                                                 │
│                         ┌──────────────────────┐│
│                         │ 142 chars truncated  ││
│                         └──────────────────────┘│
│ [status bar]                                    │
└─────────────────────────────────────────────────┘
```
Toast auto-dismisses after 3 seconds.

## Out of Scope

- Copy-to-clipboard (ctrl+c / tea.SetClipboard) — separate feature
- ctrl+v / tea.ReadClipboard() fallback for non-bracketed-paste terminals
- Paste history or undo-paste
- Rich text / image paste handling
- Paste confirmation dialog for large content

## Open Questions

- Should the toast component support severity levels (info/warn/error) for reuse by other features (queue overflow, task creation errors)? Keeping minimal for now but worth considering.
- Should paste into the agent chat input auto-submit if content ends with a newline? (Current spec: no, just insert)
- Maximum paste size cap? Currently only char limits per field. Very large clipboard content (e.g., entire file) could cause performance issues in textarea rendering.
