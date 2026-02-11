package setup

import (
	"encoding/json"
	"os/exec"
	"sort"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui"
)

// ModelInfo represents a model that can be selected.
type ModelInfo struct {
	id          string // Full model ID (e.g. "opencode/claude-opus-4-6")
	displayName string // Human-readable name (e.g. "Claude Opus 4.6")
	provider    string // Provider display name (e.g. "OpenCode Zen", "Anthropic")
	providerID  string // Provider ID (e.g. "opencode", "anthropic")
	isFree      bool   // True when input+output cost are both 0
	isHeader    bool   // True for section header items (not selectable)
}

// ID returns the unique identifier for this model (required by ScrollItem interface).
func (m *ModelInfo) ID() string {
	if m.isHeader {
		return "__header__" + m.provider
	}
	return m.id
}

// Render returns the rendered string representation (required by ScrollItem interface).
func (m *ModelInfo) Render(width int) string {
	if m.isHeader {
		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#b4befe")).Bold(true)
		return headerStyle.Render(m.provider)
	}

	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	providerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	freeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))

	var left strings.Builder
	left.WriteString("  ")
	left.WriteString(nameStyle.Render(m.displayName))
	left.WriteString(" ")
	left.WriteString(providerStyle.Render(m.provider))

	right := ""
	if m.isFree {
		right = freeStyle.Render("Free")
	}

	leftLen := lipgloss.Width(left.String())
	rightLen := lipgloss.Width(right)
	availWidth := width - 2

	if right != "" && leftLen+rightLen < availWidth {
		padding := availWidth - leftLen - rightLen
		if padding < 1 {
			padding = 1
		}
		return left.String() + strings.Repeat(" ", padding) + right
	}

	display := left.String()
	if lipgloss.Width(display) > availWidth {
		return display[:availWidth-3] + "..."
	}

	return display
}

// Height returns the number of lines this item occupies (required by ScrollItem interface).
func (m *ModelInfo) Height() int {
	return 1
}

// providerDisplayName maps providerID to human-readable name.
func providerDisplayName(providerID string) string {
	switch providerID {
	case "opencode":
		return "OpenCode Zen"
	case "anthropic":
		return "Anthropic"
	case "openai":
		return "OpenAI"
	case "google":
		return "Google"
	case "deepseek":
		return "DeepSeek"
	case "xai":
		return "xAI"
	case "mistral":
		return "Mistral"
	default:
		if len(providerID) > 0 {
			return strings.ToUpper(providerID[:1]) + providerID[1:]
		}
		return providerID
	}
}

// verboseModelJSON is the JSON structure from "opencode models --verbose".
type verboseModelJSON struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Name       string `json:"name"`
	Cost       struct {
		Input  float64 `json:"input"`
		Output float64 `json:"output"`
	} `json:"cost"`
}

// ModelStep manages the model selector UI step for setup wizard.
type ModelStep struct {
	allModels      []*ModelInfo    // Full list from opencode (no headers)
	filtered       []*ModelInfo    // Filtered by search (includes headers when no search)
	scrollList     *tui.ScrollList // Lazy-rendering scroll list for filtered models
	selectedIdx    int             // Index in filtered list (skips headers)
	searchInput    textinput.Model // Fuzzy search input
	customInput    textinput.Model // Custom model entry input
	loading        bool            // Whether models are being fetched
	error          string          // Error message if fetch failed
	isNotInstalled bool            // True if opencode is not installed
	isCustomMode   bool            // True when in custom model entry mode
	spinner        spinner.Model   // Loading spinner
	width          int             // Available width
	height         int             // Available height
}

// NewModelStep creates a new model selector step.
func NewModelStep() *ModelStep {
	// Initialize search input
	input := textinput.New()
	input.Placeholder = "Search"
	input.Prompt = ""

	// Configure styles for textinput (using lipgloss v2)
	styles := textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#585b70")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("#b4befe")),
		},
		Blurred: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#585b70")),
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

	// Initialize custom model input
	customInput := textinput.New()
	customInput.Placeholder = "e.g., anthropic/claude-opus-4"
	customInput.Prompt = "Model ID: "
	customInput.SetStyles(styles)
	customInput.SetWidth(50)

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))

	scrollList := tui.NewScrollList(60, 10)
	scrollList.SetAutoScroll(false) // Manual navigation
	scrollList.SetFocused(true)

	return &ModelStep{
		searchInput: input,
		customInput: customInput,
		scrollList:  scrollList,
		spinner:     s,
		loading:     true,
		selectedIdx: 0,
		width:       60,
		height:      10,
	}
}

// Init initializes the model selector and starts fetching models.
func (m *ModelStep) Init() tea.Cmd {
	return tea.Batch(
		m.fetchModels(),
		m.spinner.Tick,
		m.searchInput.Focus(),
	)
}

// fetchModels executes "opencode models --verbose" and parses the output.
// Falls back to plain "opencode models" if verbose fails.
func (m *ModelStep) fetchModels() tea.Cmd {
	return func() tea.Msg {
		// Check if opencode is installed
		if _, err := exec.LookPath("opencode"); err != nil {
			return ModelsErrorMsg{
				err:            err,
				isNotInstalled: true,
			}
		}

		// Try verbose mode first for rich metadata
		cmd := exec.Command("opencode", "models", "--verbose")
		output, err := cmd.Output()
		if err == nil {
			models := parseVerboseModelsOutput(output)
			if len(models) > 0 {
				return ModelsLoadedMsg{models: models}
			}
		}

		// Fallback to plain mode
		cmd = exec.Command("opencode", "models")
		output, err = cmd.Output()
		if err != nil {
			return ModelsErrorMsg{
				err:            err,
				isNotInstalled: false,
			}
		}

		models := parsePlainModelsOutput(output)
		return ModelsLoadedMsg{models: models}
	}
}

// parseVerboseModelsOutput parses the verbose JSON output from "opencode models --verbose".
func parseVerboseModelsOutput(output []byte) []*ModelInfo {
	var models []*ModelInfo

	lines := strings.Split(string(output), "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		i++

		if line == "" || strings.HasPrefix(line, "INFO") {
			continue
		}

		if strings.Contains(line, "/") && !strings.Contains(line, "{") {
			fullID := line

			var jsonLines []string
			braceCount := 0
			for i < len(lines) {
				jline := lines[i]
				i++
				trimmed := strings.TrimSpace(jline)

				if trimmed == "" && braceCount == 0 {
					continue
				}

				jsonLines = append(jsonLines, jline)

				for _, ch := range trimmed {
					switch ch {
					case '{':
						braceCount++
					case '}':
						braceCount--
					}
				}

				if braceCount <= 0 && len(jsonLines) > 0 {
					break
				}
			}

			if len(jsonLines) > 0 {
				jsonStr := strings.Join(jsonLines, "\n")
				var parsed verboseModelJSON
				if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
					displayName := parsed.Name
					if displayName == "" {
						displayName = parsed.ID
					}

					provID := parsed.ProviderID
					if provID == "" {
						if parts := strings.SplitN(fullID, "/", 2); len(parts) == 2 {
							provID = parts[0]
						}
					}

					// Only mark as free if the name/ID explicitly contains "free"
					// Cost=0 is unreliable: many providers (anthropic, github-copilot, etc.)
					// report 0 because billing is external, not because the model is free.
					isFree := strings.Contains(strings.ToLower(fullID), "free") ||
						strings.Contains(strings.ToLower(displayName), "free")

					models = append(models, &ModelInfo{
						id:          fullID,
						displayName: displayName,
						provider:    providerDisplayName(provID),
						providerID:  provID,
						isFree:      isFree,
					})
					continue
				}
			}

			parts := strings.SplitN(fullID, "/", 2)
			provID := ""
			name := fullID
			if len(parts) == 2 {
				provID = parts[0]
				name = parts[1]
			}
			models = append(models, &ModelInfo{
				id:          fullID,
				displayName: name,
				provider:    providerDisplayName(provID),
				providerID:  provID,
			})
		}
	}

	return models
}

// parsePlainModelsOutput parses the newline-separated model IDs from opencode output.
func parsePlainModelsOutput(output []byte) []*ModelInfo {
	var models []*ModelInfo

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "INFO") {
			continue
		}

		parts := strings.SplitN(line, "/", 2)
		provID := ""
		name := line
		if len(parts) == 2 {
			provID = parts[0]
			name = parts[1]
		}

		models = append(models, &ModelInfo{
			id:          line,
			displayName: name,
			provider:    providerDisplayName(provID),
			providerID:  provID,
		})
	}

	return models
}

// buildGroupedList creates the filtered list with provider section headers.
func (m *ModelStep) buildGroupedList() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	if query != "" {
		m.filtered = make([]*ModelInfo, 0)
		for _, model := range m.allModels {
			if strings.Contains(strings.ToLower(model.id), query) ||
				strings.Contains(strings.ToLower(model.displayName), query) ||
				strings.Contains(strings.ToLower(model.provider), query) {
				m.filtered = append(m.filtered, model)
			}
		}
	} else {
		groups := make(map[string][]*ModelInfo)
		var providerOrder []string
		for _, model := range m.allModels {
			if _, exists := groups[model.providerID]; !exists {
				providerOrder = append(providerOrder, model.providerID)
			}
			groups[model.providerID] = append(groups[model.providerID], model)
		}

		sort.SliceStable(providerOrder, func(i, j int) bool {
			if providerOrder[i] == "opencode" {
				return true
			}
			if providerOrder[j] == "opencode" {
				return false
			}
			return providerOrder[i] < providerOrder[j]
		})

		m.filtered = make([]*ModelInfo, 0)
		for _, provID := range providerOrder {
			models := groups[provID]
			if len(models) == 0 {
				continue
			}
			m.filtered = append(m.filtered, &ModelInfo{
				isHeader:   true,
				provider:   providerDisplayName(provID),
				providerID: provID,
			})
			m.filtered = append(m.filtered, models...)
		}
	}

	if m.selectedIdx >= len(m.filtered) {
		m.selectedIdx = 0
	}
	m.skipToSelectable(1)

	scrollItems := make([]tui.ScrollItem, len(m.filtered))
	for i, model := range m.filtered {
		scrollItems[i] = model
	}
	m.scrollList.SetItems(scrollItems)
	m.scrollList.SetSelected(m.selectedIdx)
}

// skipToSelectable moves selectedIdx to the next selectable (non-header) item.
func (m *ModelStep) skipToSelectable(dir int) {
	if len(m.filtered) == 0 {
		return
	}
	for m.selectedIdx >= 0 && m.selectedIdx < len(m.filtered) {
		if !m.filtered[m.selectedIdx].isHeader {
			return
		}
		m.selectedIdx += dir
	}
	if m.selectedIdx < 0 {
		m.selectedIdx = 0
	}
	if m.selectedIdx >= len(m.filtered) {
		m.selectedIdx = len(m.filtered) - 1
	}
}

// moveSelection moves the selection up or down, skipping headers.
func (m *ModelStep) moveSelection(dir int) {
	if len(m.filtered) == 0 {
		return
	}
	newIdx := m.selectedIdx + dir
	for newIdx >= 0 && newIdx < len(m.filtered) {
		if !m.filtered[newIdx].isHeader {
			m.selectedIdx = newIdx
			m.scrollList.SetSelected(m.selectedIdx)
			m.scrollList.ScrollToItem(m.selectedIdx)
			return
		}
		newIdx += dir
	}
}

// SetSize updates the dimensions for the model selector.
func (m *ModelStep) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.searchInput.SetWidth(width - 4)
	m.scrollList.SetWidth(width)
	m.scrollList.SetHeight(height)
}

// Update handles messages for the model selector step.
func (m *ModelStep) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ModelsLoadedMsg:
		m.loading = false
		m.allModels = msg.models
		m.buildGroupedList()
		return func() tea.Msg { return ContentChangedMsg{} }

	case ModelsErrorMsg:
		m.loading = false
		m.error = msg.err.Error()
		m.isNotInstalled = msg.isNotInstalled
		return func() tea.Msg { return ContentChangedMsg{} }

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return cmd
		}
		return nil
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return cmd
	}

	// Handle retry on error
	if m.error != "" && !m.isNotInstalled {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "r" {
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
		// In custom mode, handle differently
		if m.isCustomMode {
			switch keyMsg.String() {
			case "enter":
				customModel := strings.TrimSpace(m.customInput.Value())
				if customModel != "" {
					return func() tea.Msg {
						return ModelSelectedMsg{ModelID: customModel}
					}
				}
				return nil
			case "esc":
				m.isCustomMode = false
				m.customInput.SetValue("")
				m.customInput.Blur()
				m.searchInput.Focus()
				return func() tea.Msg { return ContentChangedMsg{} }
			}

			var cmd tea.Cmd
			m.customInput, cmd = m.customInput.Update(msg)
			cmds = append(cmds, cmd)
			return tea.Batch(cmds...)
		}

		// Normal mode
		switch keyMsg.String() {
		case "c":
			m.isCustomMode = true
			m.searchInput.Blur()
			cmds = append(cmds, m.customInput.Focus())
			cmds = append(cmds, func() tea.Msg { return ContentChangedMsg{} })
			return tea.Batch(cmds...)

		case "up", "k":
			m.moveSelection(-1)
			return nil

		case "down", "j":
			m.moveSelection(1)
			return nil

		case "enter":
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.filtered) {
				model := m.filtered[m.selectedIdx]
				if !model.isHeader {
					return func() tea.Msg {
						return ModelSelectedMsg{ModelID: model.id}
					}
				}
			}
			return nil
		}
	}

	// Update search input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	cmds = append(cmds, cmd)

	m.buildGroupedList()

	return tea.Batch(cmds...)
}

// View renders the model selector step.
func (m *ModelStep) View() string {
	var b strings.Builder

	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading models...\n")
		return b.String()
	}

	if m.error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))

		if m.isNotInstalled {
			b.WriteString(errorStyle.Render("✗ opencode is not installed"))
			b.WriteString("\n\n")
			b.WriteString(hintStyle.Render("opencode is required to fetch available models."))
			b.WriteString("\n")
			b.WriteString(hintStyle.Render("Install it from: https://github.com/opencode-ai/opencode"))
			b.WriteString("\n\n")
			b.WriteString(hintStyle.Render("Press 'c' for custom model or ESC to exit"))
		} else {
			b.WriteString(errorStyle.Render("Error: " + m.error))
			b.WriteString("\n\n")
			b.WriteString(hintStyle.Render("Press 'r' to retry, 'c' for custom model, or ESC to exit"))
		}
		return b.String()
	}

	if m.isCustomMode {
		titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#b4befe")).Bold(true)
		b.WriteString(titleStyle.Render("Enter Custom Model"))
		b.WriteString("\n\n")
		b.WriteString(m.customInput.View())
		b.WriteString("\n\n")
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
		b.WriteString(hintStyle.Render("Enter confirm • ESC cancel"))
		return b.String()
	}

	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	selectableCount := 0
	for _, model := range m.filtered {
		if !model.isHeader {
			selectableCount++
		}
	}

	if selectableCount == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")).Render("No models match your search"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")).Render("Type to filter • 'c' for custom • ESC to exit"))
		return b.String()
	}

	b.WriteString(m.scrollList.View())
	b.WriteString("\n")

	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
	b.WriteString(hintStyle.Render("↑↓/j/k navigate • Enter select • 'c' custom • ESC exit"))

	return b.String()
}

// PreferredHeight returns the preferred height for this step's content.
func (m *ModelStep) PreferredHeight() int {
	if m.loading {
		return 1
	}

	if m.error != "" {
		if m.isNotInstalled {
			return 6
		}
		return 3
	}

	if m.isCustomMode {
		return 5
	}

	overhead := 4
	listItems := len(m.filtered)
	if listItems > 20 {
		listItems = 20
	}

	return listItems + overhead
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
