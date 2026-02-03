package specwizard

import (
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// WizardModel is the main BubbleTea model for the spec wizard.
// For the tracer bullet, this is a hardcoded 1-step flow with just title input.
type WizardModel struct {
	title      string
	titleInput textinput.Model
	width      int
	height     int
	err        error
	done       bool
	cfg        *config.Config
}

// Run is the entry point for the spec wizard.
func Run(cfg *config.Config) error {
	m := &WizardModel{
		cfg:        cfg,
		titleInput: textinput.New(),
	}
	m.titleInput.Placeholder = "e.g., 'User Authentication' or 'API Rate Limiting'"
	m.titleInput.Focus()

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	wizModel, ok := finalModel.(*WizardModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	if wizModel.err != nil {
		return wizModel.err
	}

	return nil
}

func (m *WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.err = fmt.Errorf("cancelled by user")
			return m, tea.Quit
		case "enter":
			m.title = m.titleInput.Value()
			if m.title == "" {
				return m, nil
			}
			m.done = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	var cmd tea.Cmd
	m.titleInput, cmd = m.titleInput.Update(msg)
	return m, cmd
}

func (m *WizardModel) View() tea.View {
	var view tea.View
	view.AltScreen = true

	// Get the current theme
	currentTheme := theme.Current()

	// Simple centered modal
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(currentTheme.Primary)).
		MarginBottom(1)

	inputStyle := lipgloss.NewStyle().
		Width(60).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(currentTheme.BorderDefault))

	title := titleStyle.Render("Spec Wizard - Step 1: Title")
	input := inputStyle.Render(m.titleInput.View())

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.FgMuted)).
		Render("enter to continue • esc to cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		input,
		hint,
	)

	// Center on screen
	centered := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	// Draw to canvas using ultraviolet
	canvas := uv.NewScreenBuffer(m.width, m.height)
	uv.NewStyledString(centered).Draw(canvas, uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: m.width, Y: m.height},
	})

	view.Content = lipgloss.NewLayer(canvas.Render())
	return view
}
