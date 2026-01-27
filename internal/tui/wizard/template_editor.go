package wizard

import (
	"os"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/editor"
	"github.com/mark3labs/iteratr/internal/template"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// getTemplateVarStyle returns the style for highlighting {{variables}} in templates.
func getTemplateVarStyle() lipglossv2.Style {
	t := theme.Current()
	return lipglossv2.NewStyle().
		Foreground(lipglossv2.Color(t.Primary)).
		Bold(true)
}

// getTemplateHeaderStyle returns the style for markdown headers.
func getTemplateHeaderStyle() lipglossv2.Style {
	t := theme.Current()
	return lipglossv2.NewStyle().
		Foreground(lipglossv2.Color(t.Secondary)).
		Bold(true)
}

// TemplateEditorStep manages the template viewer UI step with syntax highlighting.
type TemplateEditorStep struct {
	viewport viewport.Model // Scrollable viewport for template display
	content  string         // Raw template content
	width    int            // Available width
	height   int            // Available height
	tmpFile  string         // Path to temp file for editing
}

// NewTemplateEditorStep creates a new template editor step.
func NewTemplateEditorStep() *TemplateEditorStep {
	// Create viewport
	vp := viewport.New(
		viewport.WithWidth(60),
		viewport.WithHeight(10),
	)

	// Enable mouse wheel scrolling
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	// Get default template content
	content := template.DefaultTemplate

	// Set highlighted content
	vp.SetContent(highlightTemplate(content))

	return &TemplateEditorStep{
		viewport: vp,
		content:  content,
		width:    60,
		height:   20,
	}
}

// highlightTemplate applies syntax highlighting to template content.
// Highlights {{variables}} and markdown headers.
func highlightTemplate(content string) string {
	styleVar := getTemplateVarStyle()
	styleHeader := getTemplateHeaderStyle()

	// First highlight {{variables}}
	varRegex := regexp.MustCompile(`\{\{[^}]+\}\}`)
	result := varRegex.ReplaceAllStringFunc(content, func(match string) string {
		return styleVar.Render(match)
	})

	// Then highlight markdown headers (lines starting with #)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if strings.HasPrefix(trimmed, "#") {
			// Find where the # starts
			prefix := line[:len(line)-len(trimmed)]
			lines[i] = prefix + styleHeader.Render(trimmed)
		}
	}

	return strings.Join(lines, "\n")
}

// Init initializes the template editor.
func (t *TemplateEditorStep) Init() tea.Cmd {
	return nil
}

// SetSize updates the dimensions for the template editor.
func (t *TemplateEditorStep) SetSize(width, height int) {
	t.width = width
	t.height = height

	// Update viewport dimensions (full width)
	t.viewport.SetWidth(width)

	// Reserve space for content below viewport:
	// - 1 line for hint bar
	viewportHeight := height - 1
	if viewportHeight < 5 {
		viewportHeight = 5
	}
	t.viewport.SetHeight(viewportHeight)
}

// Update handles messages for the template editor step.
func (t *TemplateEditorStep) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "e":
			// Open external editor
			return t.openEditor()
		}
	case TemplateEditedMsg:
		// Editor returned with new content
		t.content = msg.Content
		t.viewport.SetContent(highlightTemplate(t.content))
		t.viewport.GotoTop()
		// Clean up temp file
		if t.tmpFile != "" {
			_ = os.Remove(t.tmpFile)
			t.tmpFile = ""
		}
		return nil
	}

	var cmd tea.Cmd
	t.viewport, cmd = t.viewport.Update(msg)
	return cmd
}

// openEditor launches the user's $EDITOR with the template content.
func (t *TemplateEditorStep) openEditor() tea.Cmd {
	// Create temp file with template content
	tmpfile, err := os.CreateTemp("", "iteratr_template_*.md")
	if err != nil {
		return nil // Silently fail - editor not available
	}

	// Write current content to temp file
	if _, err := tmpfile.WriteString(t.content); err != nil {
		_ = tmpfile.Close()
		_ = os.Remove(tmpfile.Name())
		return nil
	}
	_ = tmpfile.Close()

	// Store temp file path for cleanup
	t.tmpFile = tmpfile.Name()

	// Create editor command
	cmd, err := editor.Command("iteratr", tmpfile.Name())
	if err != nil {
		_ = os.Remove(tmpfile.Name())
		return nil
	}

	// Execute editor and read result
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return nil
		}

		// Read modified content
		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			return nil
		}

		return TemplateEditedMsg{
			Content: string(content),
		}
	})
}

// View renders the template editor step.
func (t *TemplateEditorStep) View() string {
	var b strings.Builder

	// Render viewport
	b.WriteString(t.viewport.View())
	b.WriteString("\n")

	// Hint bar - only show edit option if $EDITOR is set
	var hintBar string
	if os.Getenv("EDITOR") != "" {
		hintBar = renderHintBar(
			"↑↓", "scroll",
			"e", "edit",
			"tab", "buttons",
			"esc", "back",
		)
	} else {
		hintBar = renderHintBar(
			"↑↓", "scroll",
			"tab", "buttons",
			"esc", "back",
		)
	}
	b.WriteString(hintBar)

	return b.String()
}

// Content returns the current template content.
func (t *TemplateEditorStep) Content() string {
	return t.content
}

// TemplateEditedMsg is sent when the external editor returns with new content.
type TemplateEditedMsg struct {
	Content string
}

// PreferredHeight returns the preferred height for this step's content.
// Template editor prefers maximum available height for comfortable viewing.
func (t *TemplateEditorStep) PreferredHeight() int {
	// Template editor wants all available space - return a large value
	// to indicate it should use max height. The modal will clamp this.
	return 100
}
