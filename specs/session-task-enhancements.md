# Session & Task Management Enhancements

Improvements to session/task management based on Anthropic's long-running agent best practices and ralph-tui patterns.

## Overview

Enhance iteratr's session and task management to better support long-running agents across multiple iterations. Key improvements: task dependencies/priorities, iteration history tracking, state snapshots for performance, and prompt refinements.

## User Story

**As a** developer running multi-iteration AI coding sessions  
**I want** better task organization, progress visibility, and faster state loading  
**So that** agents can make consistent progress across many context windows without re-doing work or getting stuck on blocked tasks

## Requirements

### Functional

1. **Task Dependencies**
   - Tasks can declare dependencies on other tasks
   - `depends_on` field: array of task IDs this task is blocked by
   - Tasks with unresolved dependencies shown as blocked in prompt
   - New command: `task-depends --id X --depends-on Y`

2. **Task Priority**
   - Priority levels: 0 (critical), 1 (high), 2 (medium), 3 (low), 4 (backlog)
   - Default priority: 2 (medium)
   - New command: `task-priority --id X --priority N`
   - Prompt shows priority in task list

3. **Iteration History**
   - Track summary of what happened each iteration
   - Store tasks worked on per iteration
   - New event action: `iteration.summary`
   - Prompt includes recent iteration summaries (last 5)

4. **State Snapshots**
   - Snapshot state to JetStream KV on iteration end
   - LoadState reads snapshot first, replays only newer events
   - Eliminates O(n) replay for every state load

5. **Prompt Improvements**
   - Remove redundant read-only tool commands (task-list, note-list, inbox-list)
   - Change "Check inbox" to "Review inbox above"
   - Add iteration history section
   - Add error recovery instructions
   - Show task dependencies and priorities in task list

### Non-Functional

1. Backward compatible with existing sessions (snapshots optional optimization)
2. No breaking changes to existing tool commands
3. Snapshot overhead < 10ms per write

## Technical Implementation

### Task Model Changes

```go
type Task struct {
    ID        string    `json:"id"`
    Content   string    `json:"content"`
    Status    string    `json:"status"`    // remaining, in_progress, completed, blocked
    Priority  int       `json:"priority"`  // 0-4, default 2
    DependsOn []string  `json:"depends_on"` // Task IDs this depends on
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Iteration int       `json:"iteration"`
}
```

### Iteration Model Changes

```go
type Iteration struct {
    Number      int       `json:"number"`
    StartedAt   time.Time `json:"started_at"`
    EndedAt     time.Time `json:"ended_at,omitempty"`
    Complete    bool      `json:"complete"`
    Summary     string    `json:"summary,omitempty"`      // What was accomplished
    TasksWorked []string  `json:"tasks_worked,omitempty"` // Task IDs touched
}
```

### New Event Actions

| Type | Action | Meta Fields |
|------|--------|-------------|
| task | priority | task_id, priority, iteration |
| task | depends | task_id, depends_on, iteration |
| iteration | summary | number, summary, tasks_worked |

### KV Snapshot Schema

```
Bucket: iteratr_snapshots
Key: {session}
Value: {
    "state": <State>,
    "after_sequence": 12345,
    "created_at": "2025-01-22T..."
}
```

### Optimized LoadState

```go
func (s *Store) LoadState(ctx context.Context, session string) (*State, error) {
    // Try snapshot first
    snapshot, err := s.kv.Get(ctx, session)
    if err == nil {
        state := decodeSnapshot(snapshot)
        // Replay only events after snapshot sequence
        return s.replayEventsAfter(ctx, session, state, snapshot.AfterSequence)
    }
    // Fall back to full replay
    return s.loadStateFull(ctx, session)
}
```

### New Tool Commands

```
iteratr tool task-priority --name SESSION --id TASK_ID --priority N
iteratr tool task-depends --name SESSION --id TASK_ID --depends-on OTHER_ID
iteratr tool task-next --name SESSION  # Returns highest priority unblocked task
iteratr tool iteration-summary --name SESSION --summary "text"
```

### Updated Prompt Template

```markdown
# iteratr Session
Session: {{session}} | Iteration: #{{iteration}}
Tasks: {{ready_count}} ready | {{blocked_count}} blocked | {{completed_count}} done

## Recent Progress
{{history}}

## Spec
{{spec}}

{{inbox}}
{{notes}}

## Current Tasks
{{tasks}}

## iteratr Tool Commands

### Task Management (REQUIRED - use via Bash)
`{{binary}} tool task-add --name {{session}} --content "description"`
`{{binary}} tool task-status --name {{session}} --id TASK_ID --status STATUS`
`{{binary}} tool task-priority --name {{session}} --id TASK_ID --priority N`
`{{binary}} tool task-depends --name {{session}} --id TASK_ID --depends-on OTHER_ID`

### Notes
`{{binary}} tool note-add --name {{session}} --content "text" --type TYPE`

### Inbox
`{{binary}} tool inbox-mark-read --name {{session}} --id MSG_ID`

### Session
`{{binary}} tool iteration-summary --name {{session}} --summary "what you accomplished"`
`{{binary}} tool session-complete --name {{session}}`

## Workflow

1. **Review inbox above** - if messages exist, process then mark read
2. **Review tasks above** - add any missing spec tasks via task-add
3. **Pick ONE ready task** - highest priority with no unresolved dependencies
4. **Mark in_progress** - `task-status --id X --status in_progress`
5. **Do the work** - implement fully, run tests
6. **Mark completed** - `task-status --id X --status completed`
7. **Write summary** - `iteration-summary --summary "what you did"`
8. **STOP** - do NOT pick another task
9. **End session** - only call session-complete when ALL tasks done

## If Something Goes Wrong

### Tests Failing
- Do NOT mark task completed
- Add note: `note-add --type stuck --content "describe failure"`
- Either fix or mark blocked with reason

### Task Blocked by External Factor
- Mark blocked: `task-status --id X --status blocked`
- Add dependency if blocked by another task: `task-depends --id X --depends-on Y`
- Pick different task or end iteration

### Need to Revert Changes
- Uncommitted: `git checkout -- <files>`
- Committed: `git revert HEAD`
- Add note explaining what went wrong

## Rules

- **ONE TASK per iteration** - complete it fully before stopping
- **Use tools above** - all task management via iteratr tool commands
- **Test before completing** - verify changes work
- **Write summary** - record what you accomplished before ending
- **session-complete required** - must call it to end the session loop
{{extra}}
```

### New Template Variables

| Variable | Type | Content |
|----------|------|---------|
| `{{history}}` | string | Formatted iteration summaries (last 5) |
| `{{ready_count}}` | int | Tasks with status=remaining and no unresolved deps |
| `{{blocked_count}}` | int | Tasks with status=blocked or unresolved deps |
| `{{completed_count}}` | int | Tasks with status=completed |

### Format Functions

```go
func formatIterationHistory(state *State) string
func formatTasksWithDeps(state *State) string
func countReadyTasks(state *State) int
func countBlockedTasks(state *State) int
```

## Tasks

### 1. Task Priority Support
- [ ] Add `Priority` field to Task struct in session.go
- [ ] Update `applyTaskEvent` to handle priority in "add" action metadata
- [ ] Create `task-priority` event action in applyTaskEvent
- [ ] Add `TaskPriority` method to Store (task.go)
- [ ] Add `task-priority` CLI command (tool_task.go)
- [ ] Update `formatTasks` to show priority as [P0]-[P4] prefix

### 2. Task Dependencies Support
- [ ] Add `DependsOn` field to Task struct in session.go
- [ ] Create `task-depends` event action in applyTaskEvent
- [ ] Add `TaskDepends` method to Store (task.go)
- [ ] Add `task-depends` CLI command (tool_task.go)
- [ ] Add `resolveBlockedByDeps` helper to identify tasks blocked by dependencies
- [ ] Update `formatTasks` to show dependency info

### 3. Task Next Command
- [ ] Add `TaskNext` method to Store - returns highest priority unblocked task
- [ ] Add `task-next` CLI command (tool_task.go)
- [ ] Consider priority (lower number = higher priority)
- [ ] Skip tasks with unresolved dependencies

### 4. Iteration Summary Support
- [ ] Add `Summary` and `TasksWorked` fields to Iteration struct
- [ ] Create `iteration.summary` event action in applyIterationEvent
- [ ] Add `IterationSummary` method to Store (iteration.go)
- [ ] Add `iteration-summary` CLI command (tool_session.go)

### 5. Iteration History in Prompt
- [ ] Add `formatIterationHistory` function (template.go)
- [ ] Add `History` field to template Variables struct
- [ ] Update `BuildPrompt` to populate history
- [ ] Update default template with `{{history}}` section

### 6. Task Count Variables
- [ ] Add `countReadyTasks` helper (template.go)
- [ ] Add `countBlockedTasks` helper (template.go)
- [ ] Add `ReadyCount`, `BlockedCount`, `CompletedCount` to Variables
- [ ] Update `BuildPrompt` to populate counts
- [ ] Update default template header with counts

### 7. KV Store Setup
- [ ] Add KV bucket creation in nats/store.go: `iteratr_snapshots`
- [ ] Add `CreateSnapshotBucket` function
- [ ] Call bucket creation in orchestrator startup

### 8. Snapshot Write
- [ ] Add `Snapshot` struct with State, AfterSequence, CreatedAt
- [ ] Add `WriteSnapshot` method to Store
- [ ] Serialize state to JSON, store in KV with session as key

### 9. Snapshot Read
- [ ] Add `ReadSnapshot` method to Store
- [ ] Return state, afterSequence, or error if not found

### 10. Optimized LoadState
- [ ] Modify `LoadState` to check snapshot first
- [ ] Add `replayEventsAfter` helper for partial replay
- [ ] Fall back to full replay if no snapshot

### 11. Snapshot on Iteration End
- [ ] Call `WriteSnapshot` after `IterationComplete` in orchestrator
- [ ] Pass current sequence number from latest event

### 12. Update Default Template
- [ ] Remove task-list, note-list, inbox-list commands from template
- [ ] Change "Check inbox" to "Review inbox above"
- [ ] Add error recovery section
- [ ] Add iteration-summary to workflow
- [ ] Update task format to show [P#] and dependency info

### 13. Enhanced Task Formatting
- [ ] Update `formatTasks` to show priority prefix [P0]-[P4]
- [ ] Show "blocked by: X, Y" for tasks with unresolved deps
- [ ] Group blocked-by-deps separately or mark clearly

### 14. Tests for Priority
- [ ] Test TaskAdd with priority
- [ ] Test TaskPriority update
- [ ] Test priority in formatTasks output

### 15. Tests for Dependencies
- [ ] Test TaskDepends add dependency
- [ ] Test task shows as blocked when dependency incomplete
- [ ] Test task unblocks when dependency completed
- [ ] Test formatTasks shows dependency info

### 16. Tests for Snapshots
- [ ] Test WriteSnapshot stores state correctly
- [ ] Test ReadSnapshot retrieves state
- [ ] Test LoadState uses snapshot when available
- [ ] Test LoadState replays events after snapshot
- [ ] Test LoadState falls back when no snapshot

### 17. Tests for Iteration Summary
- [ ] Test IterationSummary stores summary
- [ ] Test formatIterationHistory output format

## UI Mockup

### Task List with Priority/Dependencies
```
REMAINING:
  - [P1] [cp4k2m8x] Setup API routes
  - [P2] [dp5l3n9y] Add validation (blocked by: cp4k2m8x)

IN_PROGRESS:
  - [P0] [ap2i0k6w] Fix auth bug

BLOCKED:
  - [P2] [bp3j1l7v] Deploy to staging (depends on: ap2i0k6w)

COMPLETED:
  - [P1] [ep6m4o0z] Database schema [iteration #2]
```

### Iteration History
```
## Recent Progress
- #5 (3min ago): Completed "Add auth middleware", "Fix login bug"
- #4 (18min ago): Completed "Setup database models"
- #3 (1hr ago): Added 12 tasks from spec, marked 2 blocked
```

## Out of Scope

- Git commit tracking per iteration (future enhancement)
- Inbox priority levels (not needed currently)
- First-iteration initializer pattern (too complex for v1)
- Test status tracking separate from task status (can use notes for now)

## Open Questions

1. Should task-depends support removing dependencies, or only adding?
2. Should snapshots be written on every task edit, or only iteration end?
3. Max number of iteration summaries to show in prompt (5? 10?)?
4. Should blocked-by-deps tasks be in BLOCKED group or separate WAITING group?

## Dependencies

No new external dependencies. Uses existing:
- NATS JetStream (Streams + KV)
- Cobra for CLI
