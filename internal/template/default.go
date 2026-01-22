package template

// DefaultTemplate is the embedded default prompt template.
// It uses {{variable}} placeholders for dynamic content injection.
const DefaultTemplate = `# iteratr Session
Session: {{session}} | Iteration: #{{iteration}}

## Spec
{{spec}}
{{inbox}}{{notes}}

## Current Tasks (managed via iteratr tool)
{{tasks}}

## iteratr Tool Commands

IMPORTANT: You MUST use the iteratr tool via Bash for ALL task management. Do NOT use other task/todo tools.

### Task Management (REQUIRED)
` + "`" + `{{binary}} tool task-add --data-dir .iteratr --name {{session}} --content "task description"` + "`" + `
` + "`" + `{{binary}} tool task-status --data-dir .iteratr --name {{session}} --id TASK_ID --status STATUS` + "`" + `
  - Status values: remaining, in_progress, completed, blocked

### Notes (for learnings, blockers, decisions)
` + "`" + `{{binary}} tool note-add --data-dir .iteratr --name {{session}} --content "note text" --type TYPE` + "`" + `
  - Type values: learning, stuck, tip, decision

### Inbox
` + "`" + `{{binary}} tool inbox-list --data-dir .iteratr --name {{session}}` + "`" + `
` + "`" + `{{binary}} tool inbox-mark-read --data-dir .iteratr --name {{session}} --id MSG_ID` + "`" + `

### Session Control
` + "`" + `{{binary}} tool session-complete --data-dir .iteratr --name {{session}}` + "`" + `
  - Call ONLY when ALL tasks are completed

## Workflow

1. **Inbox**: Check for messages, mark read after processing
2. **SYNC ALL TASKS FROM SPEC**: Compare spec tasks against task list. ANY task in the spec that is not in the queue MUST be added via ` + "`" + `iteratr tool task-add` + "`" + `. Do this BEFORE picking a task.
3. **Pick THE ONE MOST IMPORTANT Task**: Select the highest priority unblocked task, mark it in_progress
4. **Do Work**: Implement that ONE task fully
5. **Verify**: Run tests, ensure they pass
6. **Complete**: Mark task completed, commit changes
7. **STOP**: Do NOT pick another task - your iteration is complete
8. **End Session**: Only call session-complete when ALL tasks are done

## Rules

- **LOAD ALL SPEC TASKS**: Every unchecked task in the spec MUST exist in the task queue. Add missing tasks via ` + "`" + `iteratr tool task-add` + "`" + ` before doing any work.
- **ONE TASK ONLY**: Pick the single most important task and complete it. Do NOT start another task after.
- **ALWAYS use iteratr tool**: All task management via ` + "`" + `iteratr tool` + "`" + ` commands - never use other todo/task tools
- **Test before commit**: Verify your changes work
- **session-complete is required**: You MUST call it to end the session - printing a message does nothing
{{extra}}`
