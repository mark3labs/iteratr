package template

// DefaultTemplate is the embedded default prompt template.
// It uses {{variable}} placeholders for dynamic content injection.
const DefaultTemplate = `# iteratr Session
Session: {{session}} | Iteration: #{{iteration}}

{{history}}
## Spec
{{spec}}

{{notes}}
{{tasks}}

NOTE: The "Current Tasks" section above is auto-injected at the start of each iteration. Do NOT call ` + "`" + `task-list` + "`" + ` - you already have the current state.

## iteratr Tool Commands

IMPORTANT: You MUST use the iteratr tool via Bash for ALL task management. Do NOT use other task/todo tools.

### Task Management (REQUIRED)
` + "`" + `{{binary}} tool task-add --data-dir .iteratr --name {{session}} --content "task description"` + "`" + `
` + "`" + `{{binary}} tool task-batch-add --data-dir .iteratr --name {{session}} --tasks '[{"content":"task 1"},{"content":"task 2"}]'` + "`" + `
` + "`" + `{{binary}} tool task-status --data-dir .iteratr --name {{session}} --id TASK_ID --status STATUS` + "`" + `
  - Status values: remaining, in_progress, completed, blocked
` + "`" + `{{binary}} tool task-priority --data-dir .iteratr --name {{session}} --id TASK_ID --priority N` + "`" + `
  - Priority: 0 (critical), 1 (high), 2 (medium), 3 (low), 4 (backlog)
` + "`" + `{{binary}} tool task-depends --data-dir .iteratr --name {{session}} --id TASK_ID --depends-on OTHER_ID` + "`" + `

### Notes (for learnings, blockers, decisions)
` + "`" + `{{binary}} tool note-add --data-dir .iteratr --name {{session}} --content "note text" --type TYPE` + "`" + `
  - Type values: learning, stuck, tip, decision

### Session Control
` + "`" + `{{binary}} tool iteration-summary --data-dir .iteratr --name {{session}} --summary "what you accomplished"` + "`" + `
` + "`" + `{{binary}} tool session-complete --data-dir .iteratr --name {{session}}` + "`" + `
  - Call ONLY when ALL tasks are completed

## Workflow

1. **SYNC ALL TASKS FROM SPEC**: Compare spec tasks against the "Current Tasks" section above. ANY task in the spec that is not already listed MUST be added via ` + "`" + `task-batch-add` + "`" + ` (preferred for multiple) or ` + "`" + `task-add` + "`" + `. Do this BEFORE picking a task.
2. **Pick ONE ready task** - highest priority with no unresolved dependencies
3. **Mark in_progress** - ` + "`" + `task-status --id X --status in_progress` + "`" + `
4. **Do the work** - implement fully, run tests
5. **Mark completed** - ` + "`" + `task-status --id X --status completed` + "`" + `
6. **Write summary** - ` + "`" + `iteration-summary --summary "what you did"` + "`" + `
7. **STOP** - do NOT pick another task
8. **End session** - only call session-complete when ALL tasks done

## If Something Goes Wrong

### Tests Failing
- Do NOT mark task completed
- Add note: ` + "`" + `note-add --type stuck --content "describe failure"` + "`" + `
- Either fix or mark blocked with reason

### Task Blocked by External Factor
- Mark blocked: ` + "`" + `task-status --id X --status blocked` + "`" + `
- Add dependency if blocked by another task: ` + "`" + `task-depends --id X --depends-on Y` + "`" + `
- Pick different task or end iteration

## Rules

- **ONE TASK per iteration** - complete it fully before stopping
- **LOAD ALL SPEC TASKS**: Every unchecked task in the spec MUST exist in the task queue
- **RESPECT USER-ADDED TASKS**: Tasks added by the user during the session are valid work items - do NOT cancel or remove them just because they aren't in the spec
- **ALWAYS use iteratr tool**: All task management via ` + "`" + `iteratr tool` + "`" + ` commands - never use other todo/task tools
- **Test before completing** - verify changes work
- **Write summary** - record what you accomplished before ending
- **session-complete required** - must call it to end the session loop
{{extra}}`
