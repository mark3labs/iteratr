## Architecture

See `component-tree.md` for the full TUI component tree, message flow, keyboard routing, layout management, and rendering pipeline.

## Feature Specifications

Feature specs are stored in the `specs/` directory. See `specs/README.md` for the index.

### When to Create a Spec

- New features that require design decisions
- Features with multiple components or integration points
- Work that benefits from upfront planning before implementation

### Spec Format

Each spec should include:
- **Overview** - What the feature does
- **User Story** - Who benefits and why
- **Requirements** - Detailed requirements gathered from stakeholders
- **Technical Implementation** - Routes, components, data flow
- **Tasks** - Byte-sized implementation tasks (see below)
- **UI Mockup** - ASCII or description of the interface
- **Out of Scope** - What's explicitly not included in v1
- **Open Questions** - Unresolved decisions for future discussion

### Tasks Section

Break implementation into small, sequential tasks an AI agent can complete one per iteration:
- Each task should be completable in a single focused session
- Tasks should be ordered by dependency (earlier tasks unblock later ones)
- Use checkbox format: `- [ ] Task description`
- Group related subtasks under numbered headings
- Each task should have clear success criteria implicit in description
- Aim for 5-15 tasks depending on feature complexity

Example:
```markdown
## Tasks

### 1. Create basic skeleton
- [ ] Create file with main function signature
- [ ] Add CLI argument parsing

### 2. Implement core feature
- [ ] Add helper function X
- [ ] Add helper function Y
- [ ] Wire helpers into main
```

### Spec Guidelines
- Make specs extremely concise. Sacrifice grammar for the sake of concision.

### Workflow

1. Create spec via interview process (gather requirements interactively)
2. Save to `specs/<feature-name>.md`
3. Update `specs/README.md` index table

## btca

When you need up-to-date information about technologies used in this project, use btca to query source repositories directly.

**Available resources**: opencode, bubbleteaV2, natsGo, acpGoSdk, bubbles, crush, ultraviolet, lipgloss

### Usage

```bash
btca ask -r <resource> -q "<question>"
```

Use multiple `-r` flags to query multiple resources at once:

```bash
btca ask -r opencode -r bubbleteaV2 -q "How do I build a TUI with opencode?"
```

### Using Bubbles Components

When building TUI components, prefer using Bubbles v2 pre-built components whenever possible instead of building from scratch. Bubbles provides production-ready components like:
- List (interactive scrollable lists with filtering)
- Viewport (scrollable text containers)
- TextInput (text entry fields)
- Progress (progress bars)
- Spinner (loading indicators)
- Table (interactive tables)
- Paginator (page navigation)

Query bubbles resource for component usage: `btca ask -r bubbles -q "How do I use the viewport component?"`

### Using Ultraviolet for Layouts

Use Ultraviolet for rectangle-based layout management instead of manual dimension calculations:
- `uv.SplitVertical()` - Split area into rows (top-to-bottom)
- `uv.SplitHorizontal()` - Split area into columns (left-to-right)
- `uv.Fixed(size)` - Fixed pixel/character size constraint
- `uv.Flex()` - Takes remaining space

Query ultraviolet resource for layout patterns: `btca ask -r ultraviolet -q "How do I create a responsive layout with header, content, and footer?"`

### Using Lipgloss for Styling

Use Lipgloss for styling and flexbox-like content composition:
- `lipgloss.JoinVertical()` - Stack content vertically
- `lipgloss.JoinHorizontal()` - Place content side-by-side
- `lipgloss.NewStyle()` - Create styled text with colors, borders, padding

Query lipgloss resource for styling: `btca ask -r lipgloss -q "How do I create a styled box with borders and padding?"`
