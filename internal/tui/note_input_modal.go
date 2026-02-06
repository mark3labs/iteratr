package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// focusZone represents which UI element has keyboard focus within the modal.
type focusZone int

const (
	focusTypeSelector focusZone = iota // Type selector (learning/stuck/tip/decision badges)
	focusTextarea                      // Multi-line textarea for note content
	focusSubmitButton                  // Submit button
)

// NoteInputModal is an interactive modal for creating new notes.
// It displays a textarea for content input and allows the user to submit notes.
type NoteInputModal struct {
	visible    bool
	textarea   textarea.Model
	noteType   string    // Current selected type (derived from types[typeIndex])
	types      []string  // Available note types: ["learning", "stuck", "tip", "decision"]
	typeIndex  int       // Current index in types array
	focus      focusZone // Which UI element currently has keyboard focus
	width      int
	height     int
	buttonArea uv.Rectangle // Hit area for mouse click on submit button
}

// NewNoteInputModal creates a new NoteInputModal component.
func NewNoteInputModal() *NoteInputModal {
	// Create and configure textarea
	ta := textarea.New()
	ta.Placeholder = "Enter your note..."
	ta.CharLimit = 500
	ta.ShowLineNumbers = false
	ta.Prompt = "" // No prompt character
	ta.SetWidth(50)
	ta.SetHeight(6)

	// Override textarea KeyMap to remove ctrl+n from LineNext
	// By default, textarea binds ctrl+n to move cursor down (LineNext)
	// We only want the down arrow key for this action
	ta.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))

	// Style textarea to match modal theme using default dark styles
	// and customizing the cursor color to match our secondary brand color
	t := theme.Current()
	styles := textarea.DefaultDarkStyles()
	styles.Cursor.Color = lipgloss.Color(t.Secondary)
	styles.Cursor.Shape = tea.CursorBlock
	styles.Cursor.Blink = true
	ta.SetStyles(styles)

	// Define available note types
	types := []string{"learning", "stuck", "tip", "decision"}

	return &NoteInputModal{
		visible:   false,
		textarea:  ta,
		types:     types,
		typeIndex: 0,             // Start with first type (learning)
		noteType:  types[0],      // Initialize to "learning"
		focus:     focusTextarea, // Start with textarea focused
		width:     60,
		height:    16,
	}
}

// IsVisible returns whether the modal is currently visible.
func (m *NoteInputModal) IsVisible() bool {
	return m.visible
}

// Show makes the modal visible and focuses the textarea.
func (m *NoteInputModal) Show() tea.Cmd {
	m.visible = true
	return m.textarea.Focus()
}

// Close hides the modal and resets its state.
func (m *NoteInputModal) Close() {
	m.visible = false
	m.reset()
}

// reset clears the textarea and resets the modal to initial state.
// Called on both cancel (ESC) and submit to ensure clean state on next open.
func (m *NoteInputModal) reset() {
	// Clear textarea content
	m.textarea.SetValue("")

	// Reset type selector to default (first type: learning)
	m.typeIndex = 0
	m.noteType = m.types[0]

	// Reset focus to textarea (default starting position)
	m.focus = focusTextarea

	// Blur the textarea to reset its internal state
	m.textarea.Blur()
}

// cycleFocusForward moves focus to the next element in the cycle:
// type selector → textarea → submit button → type selector (wraps)
// Returns a command to focus the textarea if it becomes the active element.
func (m *NoteInputModal) cycleFocusForward() tea.Cmd {
	oldFocus := m.focus

	switch m.focus {
	case focusTypeSelector:
		m.focus = focusTextarea
	case focusTextarea:
		m.focus = focusSubmitButton
	case focusSubmitButton:
		m.focus = focusTypeSelector
	}

	return m.updateTextareaFocus(oldFocus)
}

// cycleFocusBackward moves focus to the previous element in the cycle:
// button → textarea → type selector → button (wraps)
// Returns a command to focus the textarea if it becomes the active element.
func (m *NoteInputModal) cycleFocusBackward() tea.Cmd {
	oldFocus := m.focus

	switch m.focus {
	case focusTypeSelector:
		m.focus = focusSubmitButton
	case focusTextarea:
		m.focus = focusTypeSelector
	case focusSubmitButton:
		m.focus = focusTextarea
	}

	return m.updateTextareaFocus(oldFocus)
}

// updateTextareaFocus manages the textarea's focus/blur state based on focus changes.
// If focus moved TO textarea, it calls Focus(). If focus moved AWAY from textarea, it calls Blur().
// Returns the Focus() command if textarea should be focused, nil otherwise.
func (m *NoteInputModal) updateTextareaFocus(oldFocus focusZone) tea.Cmd {
	// Focus moved TO textarea
	if m.focus == focusTextarea && oldFocus != focusTextarea {
		return m.textarea.Focus()
	}

	// Focus moved AWAY from textarea
	if m.focus != focusTextarea && oldFocus == focusTextarea {
		m.textarea.Blur()
	}

	return nil
}

// cycleTypeForward cycles to the next note type in the types array.
// Wraps around from the last type back to the first.
func (m *NoteInputModal) cycleTypeForward() {
	m.typeIndex = (m.typeIndex + 1) % len(m.types)
	m.noteType = m.types[m.typeIndex]
}

// cycleTypeBackward cycles to the previous note type in the types array.
// Wraps around from the first type back to the last.
func (m *NoteInputModal) cycleTypeBackward() {
	m.typeIndex = (m.typeIndex - 1 + len(m.types)) % len(m.types)
	m.noteType = m.types[m.typeIndex]
}

// Update handles keyboard input for the modal.
// For now, this is a minimal implementation that will be expanded in later tasks.
func (m *NoteInputModal) Update(msg tea.Msg) tea.Cmd {
	if !m.visible {
		return nil
	}

	// Handle paste messages when textarea is focused
	if pasteMsg, ok := msg.(tea.PasteMsg); ok {
		if m.focus == focusTextarea {
			return m.handlePaste(pasteMsg)
		}
		// Paste when textarea not focused is a no-op
		return nil
	}

	// Handle key presses
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "esc":
			// ESC closes the modal
			m.Close()
			return nil
		case "ctrl+enter":
			// Ctrl+Enter submits the note
			// Get the content from textarea
			content := strings.TrimSpace(m.textarea.Value())

			// Don't submit if empty (validation)
			if content == "" {
				return nil
			}

			// Return a function that creates the CreateNoteMsg
			// The iteration will be set by App when it receives this
			return m.submit(content)
		case "tab":
			// Tab cycles focus forward: type selector → textarea → button
			return m.cycleFocusForward()
		case "shift+tab":
			// Shift+Tab cycles focus backward: button → textarea → type selector
			return m.cycleFocusBackward()
		case "left", "right":
			// Left/Right arrows when type selector is focused cycles through note types
			if m.focus == focusTypeSelector {
				if keyMsg.String() == "right" {
					m.cycleTypeForward()
				} else {
					m.cycleTypeBackward()
				}
				return nil
			}
		case "enter", "space":
			// Enter or Space when button is focused submits the note
			if m.focus == focusSubmitButton {
				content := strings.TrimSpace(m.textarea.Value())

				// Don't submit if empty (validation)
				if content == "" {
					return nil
				}

				return m.submit(content)
			}
		}
	}

	// Forward messages to textarea only when it's focused
	// This ensures other focus zones don't accidentally trigger textarea input
	if m.focus == focusTextarea {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return cmd
	}

	return nil
}

// handlePaste processes a paste message for the textarea with char limit enforcement.
// If the paste would exceed the 500 char limit, it truncates the content and shows a toast.
func (m *NoteInputModal) handlePaste(msg tea.PasteMsg) tea.Cmd {
	currentValue := m.textarea.Value()
	currentLen := len([]rune(currentValue))
	pasteContent := msg.Content
	pasteLen := len([]rune(pasteContent))
	charLimit := m.textarea.CharLimit

	// Calculate how much we can fit
	remainingSpace := charLimit - currentLen
	if remainingSpace <= 0 {
		// No space left - show toast and don't paste
		return func() tea.Msg {
			return ShowToastMsg{Text: fmt.Sprintf("%d chars truncated", pasteLen)}
		}
	}

	if pasteLen > remainingSpace {
		// Truncate the paste content
		truncatedRunes := []rune(pasteContent)[:remainingSpace]
		truncatedContent := string(truncatedRunes)
		truncatedCount := pasteLen - remainingSpace

		// Forward truncated paste to textarea
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(tea.PasteMsg{Content: truncatedContent})

		// Return batch command with textarea update + toast
		return tea.Batch(cmd, func() tea.Msg {
			return ShowToastMsg{Text: fmt.Sprintf("%d chars truncated", truncatedCount)}
		})
	}

	// No truncation needed - forward original paste to textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(tea.PasteMsg{Content: pasteContent})
	return cmd
}

// submit returns a command that creates a CreateNoteMsg.
// The App will receive this message and fill in the iteration number.
func (m *NoteInputModal) submit(content string) tea.Cmd {
	noteType := m.noteType
	return func() tea.Msg {
		return CreateNoteMsg{
			Content:   content,
			NoteType:  noteType,
			Iteration: 0, // Will be filled in by App
		}
	}
}

// HandleClick processes mouse clicks on the modal.
// Returns a command if the submit button was clicked and note is valid.
// Returns nil if click was outside button area (caller should close modal).
func (m *NoteInputModal) HandleClick(x, y int) tea.Cmd {
	// Check if click is within button area
	if x >= m.buttonArea.Min.X && x < m.buttonArea.Max.X &&
		y >= m.buttonArea.Min.Y && y < m.buttonArea.Max.Y {
		// Button was clicked - submit the note
		content := strings.TrimSpace(m.textarea.Value())

		// Don't submit if empty (validation)
		if content == "" {
			return nil
		}

		return m.submit(content)
	}

	// Click was outside button area
	return nil
}

// View renders the modal content (for testing and integration).
func (m *NoteInputModal) View() string {
	if !m.visible {
		return ""
	}

	var sections []string

	// Title - width accounts for border (2) + padding (4)
	title := renderModalTitle("New Note", m.width-6)
	sections = append(sections, title)
	sections = append(sections, "")

	// Type selector badges row
	typeBadges := m.renderTypeBadges()
	sections = append(sections, typeBadges)
	sections = append(sections, "")

	// Textarea
	sections = append(sections, m.textarea.View())
	sections = append(sections, "")

	// Submit button (static, unfocused state for now)
	button := m.renderButton()
	buttonLine := lipgloss.NewStyle().Width(m.width - 4).Align(lipgloss.Right).Render(button)
	sections = append(sections, buttonLine)
	sections = append(sections, "")

	// Hint bar at bottom with keyboard shortcuts
	hintBar := HintModal()
	hintText := lipgloss.NewStyle().Width(m.width - 4).Align(lipgloss.Center).Render(hintBar)
	sections = append(sections, hintText)

	return strings.Join(sections, "\n")
}

// renderTypeBadges renders the row of note type badges with the active type highlighted.
// When the type selector is focused, the active badge is highlighted with primary color.
// When unfocused, the active badge uses the standard type-specific color.
// Inactive badges show colored text without background.
func (m *NoteInputModal) renderTypeBadges() string {
	var badges []string
	t := theme.Current()
	s := t.S()

	for i, noteType := range m.types {
		isActive := i == m.typeIndex
		var badge lipgloss.Style
		var text string
		var typeBadge lipgloss.Style
		var typeColor string

		// Determine badge style, color, and text based on type
		switch noteType {
		case "learning":
			text = "* learning"
			typeBadge = s.BadgeSuccess
			typeColor = t.Success
		case "stuck":
			text = "! stuck"
			typeBadge = s.BadgeError
			typeColor = t.Error
		case "tip":
			text = "› tip"
			typeBadge = s.BadgeWarning
			typeColor = t.Warning
		case "decision":
			text = "◇ decision"
			typeBadge = s.BadgeInfo
			typeColor = t.Secondary
		default:
			text = "≡ " + noteType
			typeBadge = s.Badge
			typeColor = t.FgBase
		}

		if isActive {
			if m.focus == focusTypeSelector {
				// Active and focused: use primary color
				badge = s.Badge.
					Foreground(lipgloss.Color(t.FgBright)).
					Background(lipgloss.Color(t.Primary))
			} else {
				// Active but not focused: use type-specific color
				badge = typeBadge
			}
		} else {
			// Inactive: colored text without background
			badge = s.Badge.Foreground(lipgloss.Color(typeColor))
		}

		badges = append(badges, badge.Render(text))
	}

	// Join badges with spaces
	return strings.Join(badges, " ")
}

// renderButton renders the submit button in its current state with appropriate styling.
// Three states:
// - Focused: highlighted with primary color background
// - Unfocused: muted with dim background
// - Disabled: muted and visually dimmed (when content is empty)
func (m *NoteInputModal) renderButton() string {
	content := strings.TrimSpace(m.textarea.Value())
	isEmpty := content == ""
	t := theme.Current()
	s := t.S()

	var buttonStyle lipgloss.Style

	// Disabled state: content is empty
	if isEmpty {
		buttonStyle = s.BadgeMuted.
			Foreground(lipgloss.Color(t.FgSubtle)).  // Dimmed text
			Background(lipgloss.Color(t.BgSurface0)) // Very subtle background
	} else if m.focus == focusSubmitButton {
		// Focused state: highlighted with primary color
		buttonStyle = s.Badge.
			Foreground(lipgloss.Color(t.FgBright)). // Bright text
			Background(lipgloss.Color(t.Primary))   // Primary brand color
	} else {
		// Unfocused state: standard muted style
		buttonStyle = s.BadgeMuted
	}

	return buttonStyle.Render("  Save Note  ")
}

// Draw renders the modal centered on the screen buffer.
func (m *NoteInputModal) Draw(scr uv.Screen, area uv.Rectangle) {
	if !m.visible {
		return
	}

	modalWidth := m.width
	modalHeight := m.height

	// Ensure modal fits on screen with margins
	if modalWidth > area.Dx()-4 {
		modalWidth = area.Dx() - 4
	}
	if modalHeight > area.Dy()-4 {
		modalHeight = area.Dy() - 4
	}

	// Ensure minimum dimensions
	if modalWidth < 30 {
		modalWidth = 30
	}
	if modalHeight < 8 {
		modalHeight = 8
	}

	// Dynamically size textarea to fill available modal content area
	// Modal layout: title (1) + empty (1) + type badges (1) + empty (1) +
	//               textarea (N) + empty (1) + button (1) + empty (1) + hint (1)
	// Container padding: 4 (border + padding on top/bottom)
	contentPadding := 4
	fixedLines := 8 // title, 2 empty, type badges, button, hint, and separators
	availableHeight := modalHeight - contentPadding - fixedLines

	// Set minimum textarea height
	if availableHeight < 3 {
		availableHeight = 3
	}

	// Calculate textarea width accounting for modal padding and borders
	// Modal width - container padding (4) - content margin (0)
	textareaWidth := modalWidth - 4
	if textareaWidth < 20 {
		textareaWidth = 20
	}

	// Update textarea dimensions to be responsive to terminal size
	m.textarea.SetWidth(textareaWidth)
	m.textarea.SetHeight(availableHeight)

	// Build modal content using View()
	content := m.View()

	// Style the modal with border and background
	s := theme.Current().S()
	modalStyle := s.ModalContainer.
		Width(modalWidth).
		Height(modalHeight)

	modalContent := modalStyle.Render(content)

	// Calculate center position
	renderedWidth := lipgloss.Width(modalContent)
	renderedHeight := lipgloss.Height(modalContent)
	x := (area.Dx() - renderedWidth) / 2
	y := (area.Dy() - renderedHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// Calculate button area for mouse click detection
	// Button is on line: title(1) + empty(1) + badges(1) + empty(1) + textarea(N) + empty(1) = N+5
	// Within the modal content area (accounting for container padding)
	buttonLineY := 5 + availableHeight // 5 fixed lines before button + textarea height

	// Get button text to calculate its width
	buttonText := m.renderButton()
	buttonWidth := lipgloss.Width(buttonText)

	// Button is right-aligned within modal content width
	// Modal content has padding, so actual content width is modalWidth - 4
	contentWidth := modalWidth - 4
	buttonX := contentWidth - buttonWidth

	// Store button area in screen coordinates (relative to modal top-left)
	// Add 2 for modal container padding (border + padding)
	m.buttonArea = uv.Rectangle{
		Min: uv.Position{X: area.Min.X + x + 2 + buttonX, Y: area.Min.Y + y + 2 + buttonLineY},
		Max: uv.Position{X: area.Min.X + x + 2 + buttonX + buttonWidth, Y: area.Min.Y + y + 2 + buttonLineY + 1},
	}

	// Draw modal centered on screen
	modalArea := uv.Rectangle{
		Min: uv.Position{X: area.Min.X + x, Y: area.Min.Y + y},
		Max: uv.Position{X: area.Min.X + x + renderedWidth, Y: area.Min.Y + y + renderedHeight},
	}
	uv.NewStyledString(modalContent).Draw(scr, modalArea)
}
