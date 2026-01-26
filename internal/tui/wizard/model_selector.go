package wizard

import (
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui"
)

// ModelInfo represents a model that can be selected.
type ModelInfo struct {
	id   string // Model ID (e.g. "anthropic/claude-sonnet-4-5")
	name string // Display name (same as ID for now)
}

// ID returns the unique identifier for this model (required by ScrollItem interface).
func (m *ModelInfo) ID() string {
	return m.id
}

// Render returns the rendered string representation (required by ScrollItem interface).
func (m *ModelInfo) Render(width int) string {
	display := m.name

	// Truncate if too long
	if len(display) > width-2 {
		display = display[:width-5] + "..."
	}

	return display
}

// Height returns the number of lines this item occupies (required by ScrollItem interface).
func (m *ModelInfo) Height() int {
	return 1
}

// ModelSelectorStep manages the model selector UI step.
type ModelSelectorStep struct {
	allModels      []*ModelInfo    // Full list from opencode
	filtered       []*ModelInfo    // Filtered by search
	scrollList     *tui.ScrollList // Lazy-rendering scroll list for filtered models
	selectedIdx    int             // Index in filtered list
	searchInput    textinput.Model // Fuzzy search input
	loading        bool            // Whether models are being fetched
	error          string          // Error message if fetch failed
	isNotInstalled bool            // True if opencode is not installed
	spinner        spinner.Model   // Loading spinner
	width          int             // Available width
	height         int             // Available height
}

// NewModelSelectorStep creates a new model selector step.
func NewModelSelectorStep() *ModelSelectorStep {
	// Initialize search input
	input := textinput.New()
	input.Placeholder = "Type to filter models..."
	input.Prompt = "Search: "

	// Configure styles for textinput (using lipgloss v2)
	styles := textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("#b4befe")),
		},
		Blurred: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")),
		},
		Cursor: textinput.CursorStyle{
			Color: lipgloss.Color("#cba6f7"),
			Shape: tea.CursorBar,
			Blink: true,
		},
	}
	input.SetStyles(styles)
	input.SetWidth(50)

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))

	scrollList := tui.NewScrollList(60, 10)
	scrollList.SetAutoScroll(false) // Manual navigation
	scrollList.SetFocused(true)

	return &ModelSelectorStep{
		searchInput: input,
		scrollList:  scrollList,
		spinner:     s,
		loading:     true,
		selectedIdx: 0,
		width:       60,
		height:      10,
	}
}

// Init initializes the model selector and starts fetching models.
func (m *ModelSelectorStep) Init() tea.Cmd {
	return tea.Batch(
		m.fetchModels(),
		m.spinner.Tick,
		m.searchInput.Focus(),
	)
}

// fetchModels executes "opencode models" and parses the output.
func (m *ModelSelectorStep) fetchModels() tea.Cmd {
	return func() tea.Msg {
		// Check if opencode is installed
		if _, err := exec.LookPath("opencode"); err != nil {
			return ModelsErrorMsg{
				err:            err,
				isNotInstalled: true,
			}
		}

		cmd := exec.Command("opencode", "models")
		output, err := cmd.Output()
		if err != nil {
			return ModelsErrorMsg{
				err:            err,
				isNotInstalled: false,
			}
		}

		// Parse output
		models := parseModelsOutput(output)
		return ModelsLoadedMsg{models: models}
	}
}

// parseModelsOutput parses the newline-separated model IDs from opencode output.
// Skips lines starting with "INFO" and empty lines.
func parseModelsOutput(output []byte) []*ModelInfo {
	var models []*ModelInfo

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "INFO") {
			continue
		}

		models = append(models, &ModelInfo{
			id:   line,
			name: line,
		})
	}

	return models
}

// selectDefaultModel finds and selects the default model in the filtered list.
func (m *ModelSelectorStep) selectDefaultModel() {
	for i, model := range m.filtered {
		if model.id == config.DefaultModel {
			m.selectedIdx = i
			m.scrollList.SetSelected(i)
			m.scrollList.ScrollToItem(i)
			return
		}
	}
	// Default model not found, keep current selection (index 0)
}

// filterModels filters allModels by search query using case-insensitive substring match.
func (m *ModelSelectorStep) filterModels() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	if query == "" {
		// No filter, show all models
		m.filtered = m.allModels
	} else {
		// Filter by case-insensitive substring match
		m.filtered = make([]*ModelInfo, 0)
		for _, model := range m.allModels {
			if strings.Contains(strings.ToLower(model.id), query) {
				m.filtered = append(m.filtered, model)
			}
		}
	}

	// Reset selection if out of bounds
	if m.selectedIdx >= len(m.filtered) {
		m.selectedIdx = 0
	}

	// Update scroll list with filtered items
	scrollItems := make([]tui.ScrollItem, len(m.filtered))
	for i, model := range m.filtered {
		scrollItems[i] = model
	}
	m.scrollList.SetItems(scrollItems)
	m.scrollList.SetSelected(m.selectedIdx)
}

// SetSize updates the dimensions for the model selector.
func (m *ModelSelectorStep) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.searchInput.SetWidth(width - 10)
	// Reserve space for search input, spacing, and hint bar (about 6 lines)
	listHeight := height - 6
	if listHeight < 3 {
		listHeight = 3
	}
	m.scrollList.SetWidth(width)
	m.scrollList.SetHeight(listHeight)
}

// Update handles messages for the model selector step.
func (m *ModelSelectorStep) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ModelsLoadedMsg:
		// Models fetched successfully
		m.loading = false
		m.allModels = msg.models
		m.filterModels()
		// Pre-select default model if available
		m.selectDefaultModel()
		return nil

	case ModelsErrorMsg:
		// Error fetching models
		m.loading = false
		m.error = msg.err.Error()
		m.isNotInstalled = msg.isNotInstalled
		return nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return cmd
		}
		return nil
	}

	// If still loading, update spinner and return
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return cmd
	}

	// Handle retry on error
	if m.error != "" && !m.isNotInstalled {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "r" {
			// Retry fetching models
			m.loading = true
			m.error = ""
			return tea.Batch(
				m.fetchModels(),
				m.spinner.Tick,
			)
		}
	}

	// Handle keyboard input
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.scrollList.SetSelected(m.selectedIdx)
				m.scrollList.ScrollToItem(m.selectedIdx)
			}
			return nil

		case "down", "j":
			if m.selectedIdx < len(m.filtered)-1 {
				m.selectedIdx++
				m.scrollList.SetSelected(m.selectedIdx)
				m.scrollList.ScrollToItem(m.selectedIdx)
			}
			return nil

		case "enter":
			// Model selected
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.filtered) {
				model := m.filtered[m.selectedIdx]
				return func() tea.Msg {
					return ModelSelectedMsg{ModelID: model.id}
				}
			}
			return nil
		}
	}

	// Update search input (this will handle typing)
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	cmds = append(cmds, cmd)

	// Re-filter on every input change
	m.filterModels()

	return tea.Batch(cmds...)
}

// View renders the model selector step.
func (m *ModelSelectorStep) View() string {
	var b strings.Builder

	// Show loading state
	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading models...\n")
		return b.String()
	}

	// Show error state
	if m.error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))

		if m.isNotInstalled {
			// Special message for opencode not installed
			b.WriteString(errorStyle.Render("✗ opencode is not installed"))
			b.WriteString("\n\n")
			b.WriteString(hintStyle.Render("opencode is required to fetch available models."))
			b.WriteString("\n")
			b.WriteString(hintStyle.Render("Install it from: https://github.com/opencode-ai/opencode"))
			b.WriteString("\n\n")
			// Hint bar for not installed case
			hintBar := renderHintBar("tab", "buttons", "esc", "back")
			b.WriteString(hintBar)
		} else {
			// Generic error message
			b.WriteString(errorStyle.Render("Error: " + m.error))
			b.WriteString("\n\n")
			// Hint bar for retry case
			hintBar := renderHintBar("r", "retry", "tab", "buttons", "esc", "back")
			b.WriteString(hintBar)
		}
		return b.String()
	}

	// Show search input
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	// Show filtered models
	if len(m.filtered) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")).Render("No models match your search"))
		b.WriteString("\n\n")
		// Hint bar for empty search results
		hintBar := renderHintBar("type", "filter", "tab", "buttons", "esc", "back")
		b.WriteString(hintBar)
		return b.String()
	}

	// Render model list using ScrollList for lazy rendering
	b.WriteString(m.scrollList.View())

	// Add spacing before hint bar
	b.WriteString("\n")

	// Hint bar for normal state
	hintBar := renderHintBar(
		"type", "filter",
		"↑↓/j/k", "navigate",
		"enter", "select",
		"tab", "buttons",
		"esc", "back",
	)
	b.WriteString(hintBar)

	return b.String()
}

// SelectedModel returns the currently selected model ID (empty if none selected).
func (m *ModelSelectorStep) SelectedModel() string {
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.filtered) {
		return m.filtered[m.selectedIdx].id
	}
	return ""
}

// ModelsLoadedMsg is sent when models are successfully fetched.
type ModelsLoadedMsg struct {
	models []*ModelInfo
}

// ModelsErrorMsg is sent when model fetching fails.
type ModelsErrorMsg struct {
	err            error
	isNotInstalled bool // True if opencode is not installed
}

// ModelSelectedMsg is sent when a model is selected.
type ModelSelectedMsg struct {
	ModelID string
}
