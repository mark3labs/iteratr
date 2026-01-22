package template

// DefaultTemplate is the embedded default prompt template.
// It uses {{variable}} placeholders for dynamic content injection.
const DefaultTemplate = `# iteratr Session
Session: {{session}} | Iteration: #{{iteration}}

## Recent Progress
{{history}}

## Spec
{{spec}}

## Inbox
{{inbox}}

## Notes
{{notes}}

## Current Tasks
{{tasks}}

## iteratr Tool Commands

IMPORTANT: You MUST use the iteratr tool via Bash for ALL task management. Do NOT use other task/todo tools.

### Task Management (REQUIRED)
` + "`" + `{{binary}} tool task-add --data-dir .iteratr --name {{session}} --content "task description"` + "`" + `
` + "`" + `{{binary}} tool task-status --data-dir .iteratr --name {{session}} --id TASK_ID --status STATUS` + "`" + `
  - Status values: remaining, in_progress, completed, blocked
` + "`" + `{{binary}} tool task-priority --data-dir .iteratr --name {{session}} --id TASK_ID --priority N` + "`" + `
  - Priority: 0 (critical), 1 (high), 2 (medium), 3 (low), 4 (backlog)
` + "`" + `{{binary}} tool task-depends --data-dir .iteratr --name {{session}} --id TASK_ID --depends-on OTHER_ID` + "`" + `

### Notes (for learnings, blockers, decisions)
` + "`" + `{{binary}} tool note-add --data-dir .iteratr --name {{session}} --content "note text" --type TYPE` + "`" + `
  - Type values: learning, stuck, tip, decision

### Inbox
` + "`" + `{{binary}} tool inbox-mark-read --data-dir .iteratr --name {{session}} --id MSG_ID` + "`" + `

### Session Control
` + "`" + `{{binary}} tool iteration-summary --data-dir .iteratr --name {{session}} --summary "what you accomplished"` + "`" + `
` + "`" + `{{binary}} tool session-complete --data-dir .iteratr --name {{session}}` + "`" + `
  - Call ONLY when ALL tasks are completed

## Workflow

1. **Review inbox above** - if messages exist, process then mark read
2. **SYNC ALL TASKS FROM SPEC**: Compare spec tasks against task list. ANY task in the spec that is not in the queue MUST be added via ` + "`" + `iteratr tool task-add` + "`" + `. Do this BEFORE picking a task.
3. **Pick ONE ready task** - highest priority with no unresolved dependencies
4. **Mark in_progress** - ` + "`" + `task-status --id X --status in_progress` + "`" + `
5. **Do the work** - implement fully, run tests
6. **Mark completed** - ` + "`" + `task-status --id X --status completed` + "`" + `
7. **Write summary** - ` + "`" + `iteration-summary --summary "what you did"` + "`" + `
8. **STOP** - do NOT pick another task
9. **End session** - only call session-complete when ALL tasks done

## If Something Goes Wrong

### Tests Failing
- Do NOT mark task completed
- Add note: ` + "`" + `note-add --type stuck --content "describe failure"` + "`" + `
- Either fix or mark blocked with reason

### Task Blocked by External Factor
- Mark blocked: ` + "`" + `task-status --id X --status blocked` + "`" + `
- Add dependency if blocked by another task: ` + "`" + `task-depends --id X --depends-on Y` + "`" + `
- Pick different task or end iteration

### Need to Revert Changes
- Uncommitted: ` + "`" + `git checkout -- <files>` + "`" + `
- Committed: ` + "`" + `git revert HEAD` + "`" + `
- Add note explaining what went wrong

## Rules

- **ONE TASK per iteration** - complete it fully before stopping
- **LOAD ALL SPEC TASKS**: Every unchecked task in the spec MUST exist in the task queue
- **ALWAYS use iteratr tool**: All task management via ` + "`" + `iteratr tool` + "`" + ` commands - never use other todo/task tools
- **Test before completing** - verify changes work
- **Write summary** - record what you accomplished before ending
- **session-complete required** - must call it to end the session loop
{{extra}}`
