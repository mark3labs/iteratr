package wizard

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui"
)

// collapseNewlines replaces all newline sequences with a single space,
// then collapses consecutive whitespace into single space.
// Used for single-line text inputs to collapse multi-line paste content.
func collapseNewlines(content string) string {
	// Replace all newline sequences (\n, \r\n, \r) with a single space
	content = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(content)

	// Collapse consecutive spaces into single space
	return strings.Join(strings.Fields(content), " ")
}

// ModelInfo represents a model that can be selected.
type ModelInfo struct {
	id          string // Full model ID (e.g. "opencode/claude-opus-4-6")
	displayName string // Human-readable name (e.g. "Claude Opus 4.6")
	provider    string // Provider display name (e.g. "OpenCode Zen", "Anthropic")
	providerID  string // Provider ID (e.g. "opencode", "anthropic")
	isFree      bool   // True when input+output cost are both 0
	isHeader    bool   // True for section header items (not selectable)
	isActive    bool   // True if this is the currently configured model
}

// ID returns the unique identifier for this item (required by ScrollItem interface).
func (m *ModelInfo) ID() string {
	if m.isHeader {
		return "__header__" + m.provider
	}
	return m.id
}

// Render returns the rendered string representation (required by ScrollItem interface).
func (m *ModelInfo) Render(width int) string {
	if m.isHeader {
		// Section header: provider name in accent color
		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#b4befe")).Bold(true)
		return headerStyle.Render(m.provider)
	}

	// Model item: "  displayName  Provider          Free"
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	providerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	freeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa"))

	// Build left part: active indicator + display name + provider
	var left strings.Builder
	if m.isActive {
		left.WriteString(activeStyle.Render("● "))
	} else {
		left.WriteString("  ")
	}
	left.WriteString(nameStyle.Render(m.displayName))
	left.WriteString(" ")
	left.WriteString(providerStyle.Render(m.provider))

	// Build right part: "Free" badge
	right := ""
	if m.isFree {
		right = freeStyle.Render("Free")
	}

	// Calculate spacing for right-alignment
	leftLen := lipgloss.Width(left.String())
	rightLen := lipgloss.Width(right)
	availWidth := width - 2 // padding

	if right != "" && leftLen+rightLen < availWidth {
		padding := availWidth - leftLen - rightLen
		if padding < 1 {
			padding = 1
		}
		return left.String() + strings.Repeat(" ", padding) + right
	}

	// Truncate if too long
	display := left.String()
	if lipgloss.Width(display) > availWidth {
		// Simple truncation
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
		// Capitalize first letter
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

// ModelSelectorStep manages the model selector UI step.
type ModelSelectorStep struct {
	allModels       []*ModelInfo    // Full list from opencode (no headers)
	filtered        []*ModelInfo    // Filtered by search (includes headers when no search)
	scrollList      *tui.ScrollList // Lazy-rendering scroll list for filtered models
	selectedIdx     int             // Index in filtered list (skips headers)
	searchInput     textinput.Model // Fuzzy search input
	loading         bool            // Whether models are being fetched
	error           string          // Error message if fetch failed
	isNotInstalled  bool            // True if opencode is not installed
	spinner         spinner.Model   // Loading spinner
	width           int             // Available width
	height          int             // Available height
	overrideDefault string          // If set, overrides config model as the default selection
	activeModelID   string          // Currently configured model (for active indicator)
}

// NewModelSelectorStep creates a new model selector step.
func NewModelSelectorStep() *ModelSelectorStep {
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

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))

	scrollList := tui.NewScrollList(60, 10)
	scrollList.SetAutoScroll(false) // Manual navigation
	scrollList.SetFocused(true)

	// Load active model from config
	activeModel := ""
	if cfg, err := config.Load(); err == nil {
		activeModel = cfg.Model
	}

	return &ModelSelectorStep{
		searchInput:   input,
		scrollList:    scrollList,
		spinner:       s,
		loading:       true,
		selectedIdx:   0,
		width:         60,
		height:        10,
		activeModelID: activeModel,
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

// fetchModels executes "opencode models --verbose" and parses the output.
// Falls back to plain "opencode models" if verbose fails.
func (m *ModelSelectorStep) fetchModels() tea.Cmd {
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
// Format: lines of "provider/model-id" followed by JSON object blocks.
func parseVerboseModelsOutput(output []byte) []*ModelInfo {
	var models []*ModelInfo

	lines := strings.Split(string(output), "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		i++

		// Skip empty lines, INFO lines
		if line == "" || strings.HasPrefix(line, "INFO") {
			continue
		}

		// Check if this is a model ID line (contains "/" and no "{")
		if strings.Contains(line, "/") && !strings.Contains(line, "{") {
			fullID := line

			// Collect the JSON block that follows
			var jsonLines []string
			braceCount := 0
			for i < len(lines) {
				jline := lines[i]
				i++
				trimmed := strings.TrimSpace(jline)

				if trimmed == "" && braceCount == 0 {
					continue // skip empty lines before JSON
				}

				jsonLines = append(jsonLines, jline)

				for _, ch := range trimmed {
					if ch == '{' {
						braceCount++
					} else if ch == '}' {
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
						// Extract from full ID
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

			// JSON parsing failed, add as plain model
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
// Fallback when verbose mode is unavailable.
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

// SetDefaultModel sets an override for the default model selection.
// When set, this model is pre-selected instead of the config model.
// Used when resuming a session to default to the previously-used model.
func (m *ModelSelectorStep) SetDefaultModel(modelID string) {
	m.overrideDefault = modelID
}

// selectDefaultModel finds and selects the configured model in the filtered list.
// Priority: overrideDefault > config model > first selectable item.
func (m *ModelSelectorStep) selectDefaultModel() {
	// Try override first (e.g. session's previous model)
	if m.overrideDefault != "" {
		for i, model := range m.filtered {
			if !model.isHeader && model.id == m.overrideDefault {
				m.selectedIdx = i
				m.scrollList.SetSelected(i)
				m.scrollList.ScrollToItem(i)
				return
			}
		}
	}

	// Try to load model from config
	cfg, err := config.Load()
	if err == nil && cfg.Model != "" {
		for i, model := range m.filtered {
			if !model.isHeader && model.id == cfg.Model {
				m.selectedIdx = i
				m.scrollList.SetSelected(i)
				m.scrollList.ScrollToItem(i)
				return
			}
		}
	}

	// Default to first selectable item
	for i, model := range m.filtered {
		if !model.isHeader {
			m.selectedIdx = i
			m.scrollList.SetSelected(i)
			return
		}
	}
}

// buildGroupedList creates the filtered list with provider section headers.
// When searching, headers are omitted for a flat filtered list.
func (m *ModelSelectorStep) buildGroupedList() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	// Mark active model
	for _, model := range m.allModels {
		model.isActive = (model.id == m.activeModelID)
	}

	if query != "" {
		// Searching: flat list, no headers
		m.filtered = make([]*ModelInfo, 0)
		for _, model := range m.allModels {
			if strings.Contains(strings.ToLower(model.id), query) ||
				strings.Contains(strings.ToLower(model.displayName), query) ||
				strings.Contains(strings.ToLower(model.provider), query) {
				m.filtered = append(m.filtered, model)
			}
		}
	} else {
		// No search: group by provider with headers
		// Group models by providerID
		groups := make(map[string][]*ModelInfo)
		var providerOrder []string
		for _, model := range m.allModels {
			if _, exists := groups[model.providerID]; !exists {
				providerOrder = append(providerOrder, model.providerID)
			}
			groups[model.providerID] = append(groups[model.providerID], model)
		}

		// Sort providers: "opencode" first, then alphabetical
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
			// Add header
			m.filtered = append(m.filtered, &ModelInfo{
				isHeader:   true,
				provider:   providerDisplayName(provID),
				providerID: provID,
			})
			// Add models
			m.filtered = append(m.filtered, models...)
		}
	}

	// Reset selection if out of bounds or on header
	if m.selectedIdx >= len(m.filtered) {
		m.selectedIdx = 0
	}
	// Ensure selection is not on a header
	m.skipToSelectable(1)

	// Update scroll list
	scrollItems := make([]tui.ScrollItem, len(m.filtered))
	for i, model := range m.filtered {
		scrollItems[i] = model
	}
	m.scrollList.SetItems(scrollItems)
	m.scrollList.SetSelected(m.selectedIdx)
}

// skipToSelectable moves selectedIdx to the next selectable (non-header) item
// in the given direction (1 for forward, -1 for backward).
func (m *ModelSelectorStep) skipToSelectable(dir int) {
	if len(m.filtered) == 0 {
		return
	}

	for m.selectedIdx >= 0 && m.selectedIdx < len(m.filtered) {
		if !m.filtered[m.selectedIdx].isHeader {
			return
		}
		m.selectedIdx += dir
	}

	// Clamp
	if m.selectedIdx < 0 {
		m.selectedIdx = 0
	}
	if m.selectedIdx >= len(m.filtered) {
		m.selectedIdx = len(m.filtered) - 1
	}
}

// moveSelection moves the selection up or down, skipping headers.
func (m *ModelSelectorStep) moveSelection(dir int) {
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
	// Can't move further - stay in place
}

// SetSize updates the dimensions for the model selector.
func (m *ModelSelectorStep) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.searchInput.SetWidth(width - 4)
	m.scrollList.SetWidth(width)
	// Reserve space for search input (2 lines) + hint bar (2 lines)
	listHeight := height - 4
	if listHeight < 5 {
		listHeight = 5
	}
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
		m.buildGroupedList()
		// Pre-select default model if available
		m.selectDefaultModel()
		// Notify wizard that content changed (for modal resizing)
		return func() tea.Msg { return ContentChangedMsg{} }

	case ModelsErrorMsg:
		// Error fetching models
		m.loading = false
		m.error = msg.err.Error()
		m.isNotInstalled = msg.isNotInstalled
		// Notify wizard that content changed (for modal resizing)
		return func() tea.Msg { return ContentChangedMsg{} }

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

	// Handle paste messages - intercept and collapse newlines for single-line input
	if pasteMsg, ok := msg.(tea.PasteMsg); ok {
		if !m.loading && m.error == "" {
			// Sanitize and collapse newlines
			sanitized := tui.SanitizePaste(pasteMsg.Content)
			collapsed := collapseNewlines(sanitized)

			// Create modified paste message with collapsed content
			modifiedMsg := tea.PasteMsg{Content: collapsed}

			// Forward to search input
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(modifiedMsg)

			// Re-filter models with new search value
			m.buildGroupedList()

			return cmd
		}
		return nil
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
			m.moveSelection(-1)
			return nil

		case "down", "j":
			m.moveSelection(1)
			return nil

		case "enter":
			// Model selected
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

	// Update search input (this will handle typing)
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	cmds = append(cmds, cmd)

	// Re-filter on every input change
	m.buildGroupedList()

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
	selectableCount := 0
	for _, model := range m.filtered {
		if !model.isHeader {
			selectableCount++
		}
	}

	if selectableCount == 0 {
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
		model := m.filtered[m.selectedIdx]
		if !model.isHeader {
			return model.id
		}
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

// Cursor returns the cursor from the search input.
// The search input is the first line rendered in View, so no Y offset needed.
func (m *ModelSelectorStep) Cursor() *tea.Cursor {
	if m.loading || m.error != "" {
		return nil
	}
	return m.searchInput.Cursor()
}

// PreferredHeight returns the preferred height for this step's content.
// This allows the modal to size dynamically based on content.
func (m *ModelSelectorStep) PreferredHeight() int {
	// For loading state
	if m.loading {
		// "Loading models..." = 1 line
		return 1
	}

	// For error state
	if m.error != "" {
		if m.isNotInstalled {
			// Error + blank + 2 help lines + blank + hint bar = 6 lines
			return 6
		}
		// Error + blank + hint bar = 3 lines
		return 3
	}

	// For normal state:
	// - Search input: 1
	// - Blank line: 1
	// - Model list (cap at 20 for reasonable modal size)
	// - Blank line: 1
	// - Hint bar: 1
	// Total overhead: 4
	overhead := 4

	listItems := len(m.filtered)
	if listItems > 20 {
		listItems = 20
	}

	return listItems + overhead
}

// formatProviderID converts a full model ID "provider/model" to a display-friendly name.
// Used as a fallback when verbose metadata is unavailable.
func formatProviderID(fullID string) (providerID, modelName string) {
	parts := strings.SplitN(fullID, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", fullID
}

// sortModelsByName sorts models alphabetically by display name within each provider group.
func sortModelsByName(models []*ModelInfo) {
	sort.SliceStable(models, func(i, j int) bool {
		// Group by provider first
		if models[i].providerID != models[j].providerID {
			// opencode first
			if models[i].providerID == "opencode" {
				return true
			}
			if models[j].providerID == "opencode" {
				return false
			}
			return models[i].providerID < models[j].providerID
		}
		return strings.ToLower(models[i].displayName) < strings.ToLower(models[j].displayName)
	})
}

// uniqueID generates a unique scroll item ID for duplicate model IDs across providers.
func uniqueID(providerID, modelID string) string {
	return fmt.Sprintf("%s/%s", providerID, modelID)
}
