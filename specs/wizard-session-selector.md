# Wizard Session Selector

## Overview

Add a new first step to the build wizard showing existing sessions to resume, with "New Session" at the bottom to start fresh.

## User Story

As a user, I want to quickly resume a previous session without re-running the full wizard, so I can continue work without reconfiguring model/template.

## Requirements

### Session List Display
- Query JetStream for unique session names from `iteratr.*.*` subjects
- Show per session: name, completed status, task progress (X/Y completed), last activity timestamp
- "New Session" option at bottom of list
- Empty state if no sessions exist (just shows "New Session")

### Selection Behavior

**New Session selected:**
- Proceeds to existing wizard flow (FilePicker → ModelSelector → TemplateEditor → Config)
- Config step must validate session name is unique (error if exists)

**Existing session selected (incomplete):**
1. Prompt: "Reset all tasks and notes?" (default: No)
2. If yes: call `store.ResetSession()` (purges all events)
3. Skip remaining wizard steps → exit wizard with session name

**Existing session selected (completed):**
1. Prompt: "Session is complete. Continue anyway?" (default: Yes)
2. If no: return to session list
3. If yes: prompt reset (same as incomplete flow)
4. Call `store.SessionRestart()` to set `Complete = false`
5. Skip remaining wizard steps → exit wizard with session name

### Wizard Result Changes
- `WizardResult` needs way to indicate "resume mode" vs "new session mode"
- Resume mode: only `SessionName` populated, other fields empty
- Caller (orchestrator) must detect resume mode and skip spec/model/template setup

## Technical Implementation

### New Files
- `internal/tui/wizard/session_selector.go` - new step 0 component

### Modified Files
- `internal/nats/store.go` - add `ListSessions()` function
- `internal/session/session.go` - add `ListSessions()` wrapper + session info struct
- `internal/tui/wizard/wizard.go` - insert session selector as step 0, shift existing steps
- `internal/tui/wizard/wizard.go` - modify `WizardResult` for resume mode
- `internal/orchestrator/orchestrator.go` - handle resume mode (skip spec loading)

### Session Discovery
```go
// In nats/store.go
func ListSessions(ctx context.Context, stream jetstream.Stream) ([]string, error) {
    // Get stream info with subjects
    // Parse unique session names from iteratr.{session}.* pattern
}

// In session/session.go  
type SessionInfo struct {
    Name           string
    Complete       bool
    TasksTotal     int
    TasksCompleted int
    LastActivity   time.Time
}

func (s *Store) ListSessions(ctx context.Context) ([]SessionInfo, error) {
    // Get session names from nats.ListSessions()
    // For each: load state, extract info
    // Sort by LastActivity desc
}
```

### Session Selector Component
- Uses `ScrollList` (like FilePicker) for session list, max 10 visible before scroll
- Loading state with spinner while fetching session list
- Custom item renderer showing: `session-name  [Complete] 5/10 tasks  2h ago`
- Tracks selected index, handles Enter to select
- Emits `SessionSelectedMsg{Name: string, IsNew: bool}` on selection

### Confirmation Prompts
- Use simple y/n text prompts within the step (not separate modals)
- State machine: `listing` → `confirm_continue` → `confirm_reset` → done

### Wizard Flow Changes
```
Before: Step 0-3 (FilePicker, ModelSelector, TemplateEditor, Config)
After:  Step 0-4 (SessionSelector, FilePicker, ModelSelector, TemplateEditor, Config)
```

Resume flow exits after step 0. New session flow continues to step 1+.

## Tasks

### 1. Session listing infrastructure
- [ ] Add `ListSessions()` to `nats/store.go` - query stream subjects, extract unique session names
- [ ] Add `SessionInfo` struct to `session/session.go`
- [ ] Add `ListSessions()` to `session/session.go` - load each session state, build SessionInfo list

### 2. Session selector component
- [ ] Create `session_selector.go` with `SessionSelectorStep` struct
- [ ] Add loading state with spinner while fetching sessions
- [ ] Implement session list rendering with ScrollList (max 10 visible)
- [ ] Add custom item renderer (name, status, progress, timestamp)
- [ ] Handle empty state (no sessions - just show "New Session")

### 3. Confirmation prompts
- [ ] Add state machine for confirmation flow (listing → confirm_continue → confirm_reset)
- [ ] Implement "continue anyway?" prompt for completed sessions
- [ ] Implement "reset tasks and notes?" prompt (both flows)
- [ ] Call `ResetSession()` / `SessionRestart()` based on user choices

### 4. Wizard integration
- [ ] Add `SessionSelectorStep` field to `WizardModel`
- [ ] Shift step numbers (existing 0-3 become 1-4)
- [ ] Handle `SessionSelectedMsg` - branch to resume vs new session flow
- [ ] Update step titles and button labels

### 5. WizardResult and caller changes
- [ ] Add `ResumeMode bool` field to `WizardResult`
- [ ] Ensure resume mode only sets `SessionName`, leaves others empty
- [ ] Update orchestrator to detect resume mode and skip spec/model/template loading

### 6. Config step validation
- [ ] Add session name uniqueness check in config step validation
- [ ] Show error if name already exists (load session list for check)

## UI Mockup

```
┌─────────────────────────────────────────────────────────┐
│  Build Wizard - Step 1 of 5: Select Session             │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  > my-feature      [In Progress]  3/10 tasks   2h ago   │
│    api-refactor    [Complete]     8/8 tasks    1d ago   │
│    bug-fix-123     [In Progress]  0/5 tasks    3d ago   │
│    ─────────────────────────────────────────────────    │
│    + New Session                                        │
│                                                         │
├─────────────────────────────────────────────────────────┤
│                              [Cancel]  [Next]           │
└─────────────────────────────────────────────────────────┘
```

Confirmation prompt (inline):
```
┌─────────────────────────────────────────────────────────┐
│  Build Wizard - Step 1 of 5: Select Session             │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Selected: api-refactor                                 │
│                                                         │
│  Session is complete. Continue anyway? [Y/n]            │
│                                                         │
├─────────────────────────────────────────────────────────┤
│                              [Cancel]  [Back]           │
└─────────────────────────────────────────────────────────┘
```

## Out of Scope

- Session deletion from UI (use CLI or direct NATS)
- Session renaming
- Filtering/searching sessions (keep list simple for v1)
- Showing full session details before selection

## Design Decisions

- "New Session" appears at bottom of list (separator line above it)
- Max 10 sessions visible before scrolling
- Show spinner while loading session list
