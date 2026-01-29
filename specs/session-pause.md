# Session Pause/Play

## Overview

Toggle pause/resume during active session via `ctrl+x p`. Paused sessions complete current iteration, then block until resumed. Status bar shows pause state.

## User Story

As a developer, I want to pause a running session so I can step away or review changes before continuing.

## Requirements

- `ctrl+x p` toggles pause/resume (same shortcut for all states)
- Pause takes effect after current iteration completes (no mid-iteration abort)
- Toggle while PAUSING cancels pause request, agent continues to next iteration
- Status bar shows derived states:
  - `[spinner] PAUSING...` when paused but iteration still running
  - `⏸ PAUSED` when paused and orchestrator blocked
- Pause state is ephemeral (in-memory only)

## Technical Implementation

### Orchestrator Pause State

Simple in-memory state with channel for blocking:

```go
// In internal/tui/messages.go (orchestrator imports tui, so type lives here)
type PauseStateMsg struct{ Paused bool }

// In internal/orchestrator/orchestrator.go
type Orchestrator struct {
    // ... existing
    paused      atomic.Bool
    resumeChan  chan struct{}  // Buffered(1), signals resume
}

func (o *Orchestrator) RequestPause()    // Sets paused=true
func (o *Orchestrator) CancelPause()     // Sets paused=false (only effective before waitIfPaused blocks)
func (o *Orchestrator) Resume()          // Sets paused=false, sends to resumeChan
func (o *Orchestrator) IsPaused() bool   // TUI reads for display
```

TUI derives display state:
- `paused && working` → `[spinner] PAUSING...` (iteration still running)
- `paused && !working` → `⏸ PAUSED` (blocked between iterations)

### Orchestrator Loop

`internal/orchestrator/orchestrator.go` - Add pause check after iteration completes:
```go
// After processUserMessages(), before iterationCount++
if err := o.waitIfPaused(); err != nil {
    return err
}
```

New `waitIfPaused()` method:
1. If `paused == false`, return immediately
2. Send `PauseStateMsg{true}` to TUI (signals "now blocking")
3. Block on `resumeChan` (or ctx.Done())
4. On resume signal: return nil
5. On ctx.Done(): return ctx.Err()

### TUI Integration

| File | Change |
|------|--------|
| `internal/tui/app.go` | Add `p` case in ctrl+x prefix handler, calls `togglePause()` |
| `internal/tui/status.go` | Handle `PauseStateMsg`, render pause state alongside/instead of spinner |
| `internal/tui/hints.go` | Add `ctrl+x p pause` to status hints |

### Keyboard Flow

```
User presses ctrl+x -> awaitingPrefixKey = true
User presses p -> togglePause() called
  -> If !paused: call RequestPause(), send PauseStateMsg{true} to status bar
  -> If paused && working: call CancelPause(), send PauseStateMsg{false}
  -> If paused && !working: call Resume(), send PauseStateMsg{false}

Display states (TUI derives from paused + working):
  !paused            --> [spinner] (normal)
  paused && working  --> [spinner] PAUSING...
  paused && !working --> ⏸ PAUSED
```

## Tasks

### 1. Orchestrator pause state (tracer bullet foundation)

- [ ] Add `PauseStateMsg` to `internal/tui/messages.go`
- [ ] Add `paused atomic.Bool` field to Orchestrator
- [ ] Add `RequestPause()`, `CancelPause()`, `Resume()` methods
- [ ] Add `IsPaused() bool` method for TUI to read

### 2. Orchestrator pause loop

- [ ] Add `resumeChan chan struct{}` field, initialize as `make(chan struct{}, 1)` in constructor
- [ ] Add `waitIfPaused()` method to orchestrator
- [ ] If `!paused`: return immediately
- [ ] If `paused`: send `PauseStateMsg{true}` to TUI, block on resumeChan or ctx.Done()
- [ ] On resume signal: drain channel, return nil
- [ ] On ctx.Done(): return ctx.Err()
- [ ] Call `waitIfPaused()` after `processUserMessages()` in loop
- [ ] Add debug logging for pause/resume

### 3. TUI keyboard binding

- [ ] Add `p` case to ctrl+x prefix handler in `app.go`
- [ ] Implement `togglePause()`:
  - Guard: if orchestrator is nil, return early
  - If `!paused`: call RequestPause()
  - If `paused && working`: call CancelPause()
  - If `paused && !working`: call Resume()
- [ ] Send `PauseStateMsg` to status bar after state change
- [ ] Verify keyboard routing works with existing prefix keys

### 4. Status bar indicator

- [ ] Add `StatusPausing` and `StatusPaused` styles to theme
- [ ] Add `paused bool` field to StatusBar, updated via `PauseStateMsg`
- [ ] In `buildLeft()`: derive display from `paused` + `working` fields
- [ ] If `paused && working`: show `[spinner] PAUSING...` (spinner stays visible)
- [ ] If `paused && !working`: show `⏸ PAUSED` (static icon, no spinner)

### 5. Hints update

- [ ] Add `KeyCtrlXP` constant to hints.go
- [ ] Update `HintStatus()` to include pause shortcut

## UI Mockup

**Normal state (spinner at end):**
```
iteratr | my-session | 00:12:34 | Iteration #3 | ✓2 ●1 ○3 | 5 files modified ◐
```

**Pausing state (iteration still running, spinner active):**
```
iteratr | my-session | 00:12:34 | Iteration #3 | ✓2 ●1 ○3 | 5 files modified ◐ PAUSING...
```

**Paused state (iteration complete, loop blocked):**
```
iteratr | my-session | 00:12:34 | Iteration #3 | ✓2 ●1 ○3 | 5 files modified ⏸ PAUSED
```

## Out of Scope (v1)

- CLI tool commands (`iteratr tool session-pause`)
- Pause timeout/expiry
- Pause during human-in-the-loop prompts (no special handling)
- Pause confirmation dialog

## Open Questions

None - all resolved during interview.
