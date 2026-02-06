package specwizard

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui/theme"
	"github.com/mark3labs/iteratr/internal/tui/wizard"
)

// Question represents a single question from the ask-questions tool.
type Question struct {
	Question string   // Full question text
	Header   string   // Short label (max 30 chars)
	Options  []Option // Available options
	Multiple bool     // Allow multi-select
}

// Option represents a single option for a question.
type Option struct {
	Label       string // Display text (1-5 words)
	Description string // Optional description
}

// QuestionAnswer stores a single answer (handles both single and multi-select).
type QuestionAnswer struct {
	Value   interface{} // string for single-select, []string for multi-select
	IsMulti bool
}

// OptionItem represents a single question option with selection state.
type OptionItem struct {
	label       string
	description string
	selected    bool
}

// QuestionOptions manages option selection with keyboard navigation.
type QuestionOptions struct {
	items       []OptionItem
	cursor      int  // Currently focused option
	multiSelect bool // Allow multiple selections
	focused     bool
}

// NewQuestionOptions creates a new question options selector.
func NewQuestionOptions(opts []OptionItem, multiSelect bool) QuestionOptions {
	return QuestionOptions{
		items:       opts,
		cursor:      0,
		multiSelect: multiSelect,
		focused:     true,
	}
}

// CursorUp moves cursor up.
func (q *QuestionOptions) CursorUp() {
	if q.cursor > 0 {
		q.cursor--
	}
}

// CursorDown moves cursor down.
func (q *QuestionOptions) CursorDown() {
	if q.cursor < len(q.items)-1 {
		q.cursor++
	}
}

// Toggle toggles selection of the current option.
func (q *QuestionOptions) Toggle() {
	isCustomOption := q.cursor == len(q.items)-1 // Last item is "Type your own answer"

	if q.multiSelect {
		if isCustomOption {
			// "Type your own" is mutually exclusive in multi-select
			for i := range q.items {
				q.items[i].selected = false
			}
			q.items[q.cursor].selected = true
		} else {
			// Selecting normal option: deselect "Type your own" and toggle current
			q.items[len(q.items)-1].selected = false
			q.items[q.cursor].selected = !q.items[q.cursor].selected
		}
	} else {
		// Single-select: unselect all, select current
		for i := range q.items {
			q.items[i].selected = false
		}
		q.items[q.cursor].selected = true
	}
}

// SelectedIndices returns indices of all selected options.
func (q QuestionOptions) SelectedIndices() []int {
	var indices []int
	for i, item := range q.items {
		if item.selected {
			indices = append(indices, i)
		}
	}
	return indices
}

// SelectedLabels returns labels of all selected options.
func (q QuestionOptions) SelectedLabels() []string {
	var labels []string
	for _, item := range q.items {
		if item.selected {
			labels = append(labels, item.label)
		}
	}
	return labels
}

// Focus focuses the option selector.
func (q *QuestionOptions) Focus() { q.focused = true }

// Blur blurs the option selector.
func (q *QuestionOptions) Blur() { q.focused = false }

// Update handles messages for the option selector.
func (q QuestionOptions) Update(msg tea.Msg) (QuestionOptions, tea.Cmd) {
	if !q.focused {
		return q, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()
		switch key {
		case "up", "k":
			q.CursorUp()
		case "down", "j":
			q.CursorDown()
		case " ", "enter":
			q.Toggle()
		}
	}
	return q, nil
}

// View renders the option selector.
func (q QuestionOptions) View() string {
	var b strings.Builder
	currentTheme := theme.Current()

	for i, item := range q.items {
		// Determine indicator based on selection type
		var indicator string
		if q.multiSelect {
			indicator = "☐" // Checkbox
			if item.selected {
				indicator = "☑"
			}
		} else {
			indicator = "○" // Radio button
			if item.selected {
				indicator = "●"
			}
		}

		// Show cursor for focused item
		cursor := "  "
		if i == q.cursor && q.focused {
			cursor = "▶ "
		}

		// Style based on focus/selection
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(currentTheme.FgBase))
		if i == q.cursor && q.focused {
			style = style.Foreground(lipgloss.Color(currentTheme.Primary)).Bold(true)
		}

		line := fmt.Sprintf("%s%s %s", cursor, indicator, item.label)
		b.WriteString(style.Render(line))
		b.WriteString("\n")

		// Show description with indent
		if item.description != "" {
			descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(currentTheme.FgMuted))
			b.WriteString(descStyle.Render(fmt.Sprintf("      %s", item.description)))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// QuestionView manages a single question with options, custom input, and navigation.
type QuestionView struct {
	questions    []Question
	answers      []QuestionAnswer // Persistent answers (parallel to questions)
	currentIndex int

	// Option selection component
	optionSelector QuestionOptions

	// Custom text input
	customInput textinput.Model
	showCustom  bool

	// Focus management
	// 0=options, 1=custom (if visible), 2=back button (if not first Q), 3=next/submit button
	focusIndex int

	// Button bar
	buttonBar *wizard.ButtonBar

	// Error text displayed below buttons (set by ShowErrorMsg)
	errText string

	width  int
	height int
}

// NewQuestionView creates a new question view.
func NewQuestionView(questions []Question, answers []QuestionAnswer, currentIndex int) *QuestionView {
	q := questions[currentIndex]

	// Build option items from question
	optItems := make([]OptionItem, len(q.Options))
	for i, opt := range q.Options {
		optItems[i] = OptionItem{
			label:       opt.Label,
			description: opt.Description,
			selected:    false,
		}
	}
	// Add "Type your own answer" option
	optItems = append(optItems, OptionItem{
		label:       "Type your own answer",
		description: "Enter custom text",
		selected:    false,
	})

	ti := textinput.New()
	ti.Placeholder = "Type your answer..."
	ti.CharLimit = 500

	qv := &QuestionView{
		questions:      questions,
		answers:        answers,
		currentIndex:   currentIndex,
		optionSelector: NewQuestionOptions(optItems, q.Multiple),
		customInput:    ti,
		focusIndex:     0,
	}

	// Restore previous answer
	qv.restoreAnswer()
	qv.optionSelector.Focus()

	// Initialize button bar
	qv.rebuildButtonBar()

	return qv
}

// restoreAnswer restores the previously saved answer for the current question.
func (q *QuestionView) restoreAnswer() {
	answer := q.answers[q.currentIndex]
	currentQ := q.questions[q.currentIndex]

	if answer.IsMulti {
		// Multi-select answer
		if labels, ok := answer.Value.([]string); ok {
			if len(labels) == 0 {
				return
			}

			// Check if it's a custom answer
			if len(labels) == 1 && !q.isOptionLabel(labels[0]) {
				// Custom text
				q.customInput.SetValue(labels[0])
				q.showCustom = true
				lastIdx := len(q.optionSelector.items) - 1
				q.optionSelector.items[lastIdx].selected = true
			} else {
				// Regular options
				for _, label := range labels {
					for i, opt := range currentQ.Options {
						if opt.Label == label {
							q.optionSelector.items[i].selected = true
						}
					}
				}
			}
		}
	} else {
		// Single-select answer
		if label, ok := answer.Value.(string); ok && label != "" {
			// Try to match to option
			found := false
			for i, opt := range currentQ.Options {
				if opt.Label == label {
					q.optionSelector.items[i].selected = true
					found = true
					break
				}
			}

			if !found {
				// Must be custom text
				q.customInput.SetValue(label)
				q.showCustom = true
				lastIdx := len(q.optionSelector.items) - 1
				q.optionSelector.items[lastIdx].selected = true
			}
		}
	}
}

// isOptionLabel checks if a label matches any of the question's options.
func (q *QuestionView) isOptionLabel(label string) bool {
	for _, opt := range q.questions[q.currentIndex].Options {
		if opt.Label == label {
			return true
		}
	}
	return false
}

// saveCurrentAnswer saves the current answer to the answers array.
func (q *QuestionView) saveCurrentAnswer() {
	selected := q.optionSelector.SelectedLabels()
	currentQ := q.questions[q.currentIndex]

	if len(selected) == 0 {
		// No selection
		if currentQ.Multiple {
			q.answers[q.currentIndex] = QuestionAnswer{Value: []string{}, IsMulti: true}
		} else {
			q.answers[q.currentIndex] = QuestionAnswer{Value: "", IsMulti: false}
		}
		return
	}

	// Check if "Type your own" is selected
	isCustom := selected[0] == "Type your own answer"

	if currentQ.Multiple {
		// Multi-select answer
		if isCustom {
			customText := q.customInput.Value()
			q.answers[q.currentIndex] = QuestionAnswer{
				Value:   []string{customText},
				IsMulti: true,
			}
		} else {
			// Remove "Type your own answer" if somehow in list
			filtered := make([]string, 0, len(selected))
			for _, s := range selected {
				if s != "Type your own answer" {
					filtered = append(filtered, s)
				}
			}
			q.answers[q.currentIndex] = QuestionAnswer{
				Value:   filtered,
				IsMulti: true,
			}
		}
	} else {
		// Single-select answer
		if isCustom {
			q.answers[q.currentIndex] = QuestionAnswer{
				Value:   q.customInput.Value(),
				IsMulti: false,
			}
		} else {
			q.answers[q.currentIndex] = QuestionAnswer{
				Value:   selected[0],
				IsMulti: false,
			}
		}
	}
}

// maxFocusIndex calculates the maximum focus index based on current state.
func (q *QuestionView) maxFocusIndex() int {
	// 0 = options (always present)
	max := 0
	if q.showCustom {
		max++ // 1 = custom input
	}
	if q.currentIndex > 0 {
		max++ // Back button (only after first question)
	}
	max++ // Next/Submit button (always present)
	return max
}

// backButtonFocusIndex returns the focus index for the back button.
func (q *QuestionView) backButtonFocusIndex() int {
	idx := 1
	if q.showCustom {
		idx++
	}
	return idx
}

// nextButtonFocusIndex returns the focus index for the next/submit button.
func (q *QuestionView) nextButtonFocusIndex() int {
	idx := q.backButtonFocusIndex()
	if q.currentIndex > 0 {
		idx++
	}
	return idx
}

// updateFocus updates focus based on current focusIndex.
func (q *QuestionView) updateFocus() {
	// Blur all components
	q.optionSelector.Blur()
	q.customInput.Blur()
	if q.buttonBar != nil {
		q.buttonBar.Blur()
	}

	// Focus active component
	switch q.focusIndex {
	case 0:
		q.optionSelector.Focus()
	case 1:
		if q.showCustom {
			q.customInput.Focus()
		} else {
			// When custom is hidden, index 1 is a button position
			// Focus the appropriate button based on whether we have back button
			if q.buttonBar != nil {
				if q.currentIndex > 0 {
					// Has back button at index 1
					q.buttonBar.FocusFirst()
				} else {
					// No back button, next is at index 1
					q.buttonBar.FocusLast()
				}
			}
		}
	default:
		// Button bar is focused (indices 2+)
		if q.buttonBar != nil {
			if q.focusIndex == q.backButtonFocusIndex() && q.currentIndex > 0 {
				q.buttonBar.FocusFirst() // Focus back button
			} else if q.focusIndex == q.nextButtonFocusIndex() {
				q.buttonBar.FocusLast() // Focus next/submit button
			}
		}
	}
}

// validateAnswer validates the current answer.
func (q *QuestionView) validateAnswer() bool {
	selected := q.optionSelector.SelectedLabels()
	if len(selected) == 0 {
		return false
	}

	// If "Type your own" is selected, check that custom text is non-empty
	if selected[0] == "Type your own answer" {
		return strings.TrimSpace(q.customInput.Value()) != ""
	}

	return true
}

// Update handles messages for the question view.
func (q *QuestionView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		q.width = msg.Width
		q.height = msg.Height
		return nil

	case tea.KeyPressMsg:
		// Clear any visible error text on user interaction
		q.errText = ""
		switch msg.String() {
		case "esc":
			if q.currentIndex > 0 {
				// Not on first question: go back to previous question
				q.saveCurrentAnswer()
				return func() tea.Msg { return PrevQuestionMsg{} }
			}
			// On first question: return a message to show cancel confirmation
			// This will be handled by the parent (agent phase)
			return func() tea.Msg { return ShowCancelConfirmMsg{} }

		case "tab":
			q.focusIndex++
			// Wrap around
			if q.focusIndex > q.maxFocusIndex() {
				q.focusIndex = 0
			}
			q.updateFocus()
			return nil

		case "shift+tab":
			q.focusIndex--
			// Wrap around
			if q.focusIndex < 0 {
				q.focusIndex = q.maxFocusIndex()
			}
			q.updateFocus()
			return nil
		}
	}

	// Route to focused component
	if q.focusIndex == 0 {
		// Options focused
		var cmd tea.Cmd
		q.optionSelector, cmd = q.optionSelector.Update(msg)

		// Show/hide custom input based on selection
		selected := q.optionSelector.SelectedLabels()
		wasCustom := q.showCustom
		q.showCustom = len(selected) > 0 && selected[0] == "Type your own answer"

		// If custom input became visible, update button bar
		if !wasCustom && q.showCustom {
			q.rebuildButtonBar()
		} else if wasCustom && !q.showCustom {
			q.rebuildButtonBar()
		}

		return cmd
	} else if q.focusIndex == 1 && q.showCustom {
		// Custom input focused
		var cmd tea.Cmd
		q.customInput, cmd = q.customInput.Update(msg)
		return cmd
	} else if q.focusIndex == q.backButtonFocusIndex() && q.currentIndex > 0 {
		// Back button focused
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			if keyMsg.String() == "enter" || keyMsg.String() == " " {
				q.saveCurrentAnswer()
				return func() tea.Msg { return PrevQuestionMsg{} }
			}
		}
	} else if q.focusIndex == q.nextButtonFocusIndex() {
		// Next/Submit button focused
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			if keyMsg.String() == "enter" || keyMsg.String() == " " {
				if !q.validateAnswer() {
					return func() tea.Msg {
						return ShowErrorMsg{err: "Please select an answer or enter custom text"}
					}
				}
				q.saveCurrentAnswer()

				if q.currentIndex == len(q.questions)-1 {
					return func() tea.Msg { return SubmitAnswersMsg{} }
				}
				return func() tea.Msg { return NextQuestionMsg{} }
			}
		}
	}

	return nil
}

// rebuildButtonBar recreates the button bar based on current state.
func (q *QuestionView) rebuildButtonBar() {
	buttons := []wizard.Button{}

	// Back button (only show if not first question)
	if q.currentIndex > 0 {
		buttons = append(buttons, wizard.Button{
			Label: "← Back",
			State: wizard.ButtonNormal,
		})
	}

	// Next/Submit button
	nextLabel := "Next →"
	if q.currentIndex == len(q.questions)-1 {
		nextLabel = "Submit"
	}
	buttons = append(buttons, wizard.Button{
		Label: nextLabel,
		State: wizard.ButtonNormal,
	})

	q.buttonBar = wizard.NewButtonBar(buttons)
	q.buttonBar.SetWidth(q.width)

	// Restore focus to button bar if currently focused
	if q.focusIndex == q.backButtonFocusIndex() {
		q.buttonBar.FocusFirst()
	} else if q.focusIndex == q.nextButtonFocusIndex() {
		q.buttonBar.FocusLast()
	}
}

// View renders the question view.
func (q *QuestionView) View() string {
	var b strings.Builder
	currentTheme := theme.Current()

	// Question counter
	counterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(currentTheme.FgMuted))
	b.WriteString(counterStyle.Render(
		fmt.Sprintf("Question %d of %d", q.currentIndex+1, len(q.questions))))
	b.WriteString("\n\n")

	currentQ := q.questions[q.currentIndex]

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(currentTheme.Primary))
	b.WriteString(headerStyle.Render(currentQ.Header))
	b.WriteString("\n\n")

	// Question text
	b.WriteString(currentQ.Question)
	b.WriteString("\n\n")

	// Options (using QuestionOptions component)
	b.WriteString(q.optionSelector.View())

	// Custom input (if visible)
	if q.showCustom {
		b.WriteString("\n")
		b.WriteString(q.customInput.View())
		b.WriteString("\n")
	}

	// Navigation buttons
	b.WriteString("\n")
	b.WriteString(q.buttonBar.Render())

	// Error text (if any)
	if q.errText != "" {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(currentTheme.Error)).
			MarginTop(1)
		b.WriteString("\n")
		b.WriteString(errStyle.Render(q.errText))
	}

	return b.String()
}

// SetSize updates the dimensions of the question view.
func (q *QuestionView) SetSize(width, height int) {
	q.width = width
	q.height = height
	if q.buttonBar != nil {
		q.buttonBar.SetWidth(width)
	}
}

// PrevQuestionMsg is sent when the user navigates to the previous question.
type PrevQuestionMsg struct{}

// NextQuestionMsg is sent when the user navigates to the next question.
type NextQuestionMsg struct{}

// SubmitAnswersMsg is sent when the user submits all answers.
type SubmitAnswersMsg struct{}

// ShowErrorMsg is sent when a validation error occurs.
type ShowErrorMsg struct {
	err string
}
