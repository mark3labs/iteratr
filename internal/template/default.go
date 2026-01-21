package template

// DefaultTemplate is the embedded default prompt template.
// It uses {{variable}} placeholders for dynamic content injection.
const DefaultTemplate = `## Context
Session: {{session}} | Iteration: #{{iteration}}
Spec: {{spec}}
{{inbox}}{{notes}}
## Task State
{{tasks}}

## Tools - all require session_name="{{session}}"
- inbox_list / inbox_mark_read(id) - check/ack messages
- task_add(content, status?) / task_status(id, status) / task_list - manage tasks
- note_add(content, type) / note_list(type?) - record learnings/tips/blockers/decisions
- session_complete - call when ALL tasks done

## Workflow
1. Check inbox, mark read after processing
2. Ensure all spec tasks exist in task list
3. Pick ONE task, mark in_progress, do work, mark completed
4. Run tests, commit with clear message
5. If stuck/learned something: note_add
6. When ALL done: session_complete

Rules: ONE task/iteration. Test before commit. Call session_complete to end.
{{extra}}`
