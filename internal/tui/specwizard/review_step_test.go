package specwizard

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/config"
)

func TestNewReviewStep(t *testing.T) {
	content := "# Test Spec\n\nThis is a test spec."
	cfg := &config.Config{}

	step := NewReviewStep(content, cfg)

	if step == nil {
		t.Fatal("Expected step to be non-nil")
	}
	if step.content != content {
		t.Errorf("Expected content %q, got %q", content, step.content)
	}
	if step.cfg != cfg {
		t.Error("Expected config to match")
	}
	if step.edited {
		t.Error("Expected edited to be false initially")
	}
}

func TestReviewStep_SetSize(t *testing.T) {
	content := "# Test Spec"
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)

	// Set size
	step.SetSize(80, 30)

	if step.width != 80 {
		t.Errorf("Expected width 80, got %d", step.width)
	}
	if step.height != 30 {
		t.Errorf("Expected height 30, got %d", step.height)
	}

	// Verify SetSize doesn't panic and updates dimensions
	// (viewport internal dimensions are managed by viewport.Model)
}

func TestReviewStep_View(t *testing.T) {
	content := "# Test Spec\n\nSome content here."
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)
	step.SetSize(80, 20)

	view := step.View()

	// View should contain the hint bar
	if !strings.Contains(view, "scroll") {
		t.Error("Expected view to contain 'scroll' hint")
	}
	if !strings.Contains(view, "buttons") {
		t.Error("Expected view to contain 'buttons' hint")
	}
	if !strings.Contains(view, "back") {
		t.Error("Expected view to contain 'back' hint")
	}
}

func TestReviewStep_ViewWithEditor(t *testing.T) {
	// Set EDITOR env var to enable edit option
	oldEditor := os.Getenv("EDITOR")
	if err := os.Setenv("EDITOR", "vim"); err != nil {
		t.Fatalf("Failed to set EDITOR: %v", err)
	}
	defer func() {
		if oldEditor == "" {
			_ = os.Unsetenv("EDITOR")
		} else {
			_ = os.Setenv("EDITOR", oldEditor)
		}
	}()

	content := "# Test Spec"
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)
	step.SetSize(80, 20)

	view := step.View()

	// View should contain edit hint when EDITOR is set
	if !strings.Contains(view, "edit") {
		t.Error("Expected view to contain 'edit' hint when EDITOR is set")
	}
}

func TestReviewStep_Update_Scrolling(t *testing.T) {
	// Create long content to enable scrolling
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "Line "+string(rune('0'+i%10)))
	}
	content := strings.Join(lines, "\n")

	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)
	step.SetSize(80, 20)

	// Simulate down arrow key (should scroll)
	cmd := step.Update(tea.KeyPressMsg{Text: "down"})

	// Should return a command (viewport handles scrolling)
	_ = cmd // Command may be nil or a viewport command
}

func TestReviewStep_Update_SpecEditedMsg(t *testing.T) {
	content := "# Original Content"
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)
	step.SetSize(80, 20)

	// Simulate external edit
	newContent := "# Edited Content\n\nNew section added."
	cmd := step.Update(SpecEditedMsg{Content: newContent})

	if cmd != nil {
		t.Error("Expected no command from SpecEditedMsg")
	}
	if step.content != newContent {
		t.Errorf("Expected content to be updated to %q, got %q", newContent, step.content)
	}
	if !step.edited {
		t.Error("Expected edited flag to be true")
	}
}

func TestReviewStep_Content(t *testing.T) {
	content := "# Test Spec"
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)

	if step.Content() != content {
		t.Errorf("Expected Content() to return %q, got %q", content, step.Content())
	}

	// Update content via SpecEditedMsg
	newContent := "# Updated Spec"
	step.Update(SpecEditedMsg{Content: newContent})

	if step.Content() != newContent {
		t.Errorf("Expected Content() to return %q after edit, got %q", newContent, step.Content())
	}
}

func TestReviewStep_WasEdited(t *testing.T) {
	content := "# Test Spec"
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)

	if step.WasEdited() {
		t.Error("Expected WasEdited() to return false initially")
	}

	// Simulate external edit
	step.Update(SpecEditedMsg{Content: "# Edited"})

	if !step.WasEdited() {
		t.Error("Expected WasEdited() to return true after edit")
	}
}

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		content string
		width   int
	}{
		{
			name:    "simple markdown",
			content: "# Header\n\nParagraph text.",
			width:   80,
		},
		{
			name:    "wide width capped at 120",
			content: "Some text",
			width:   150, // Should be capped to 120
		},
		{
			name:    "code block",
			content: "```go\nfunc main() {}\n```",
			width:   80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderMarkdown(tt.content, tt.width)

			// Should return non-empty string
			if result == "" {
				t.Error("Expected non-empty rendered markdown")
			}

			// Result should not have trailing newline (glamour adds one, we strip it)
			if strings.HasSuffix(result, "\n\n") {
				t.Error("Expected single trailing newline to be stripped")
			}
		})
	}
}

func TestRenderHintBar(t *testing.T) {
	tests := []struct {
		name     string
		pairs    []string
		expected []string // Expected substrings in output
	}{
		{
			name:     "single pair",
			pairs:    []string{"↑↓", "scroll"},
			expected: []string{"↑↓", "scroll"},
		},
		{
			name:     "multiple pairs",
			pairs:    []string{"↑↓", "scroll", "e", "edit", "esc", "back"},
			expected: []string{"↑↓", "scroll", "e", "edit", "esc", "back"},
		},
		{
			name:     "empty pairs",
			pairs:    []string{},
			expected: []string{},
		},
		{
			name:     "odd number of pairs (invalid)",
			pairs:    []string{"↑↓", "scroll", "e"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderHintBar(tt.pairs...)

			if len(tt.expected) == 0 {
				if result != "" {
					t.Errorf("Expected empty result, got %q", result)
				}
				return
			}

			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("Expected result to contain %q, got %q", exp, result)
				}
			}
		})
	}
}

func TestReviewStep_ButtonBar(t *testing.T) {
	content := "# Test Spec\n\nSome content."
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)
	step.SetSize(80, 20)

	// Verify button bar exists
	if step.buttonBar == nil {
		t.Fatal("Expected button bar to be initialized")
	}

	// View should contain button bar rendering
	view := step.View()
	if !strings.Contains(view, "Restart") {
		t.Error("Expected view to contain 'Restart' button")
	}
	if !strings.Contains(view, "Save") {
		t.Error("Expected view to contain 'Save' button")
	}
}

func TestReviewStep_TabNavigationToButtons(t *testing.T) {
	content := "# Test Spec"
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)
	step.SetSize(80, 20)

	// Initially buttons not focused
	if step.buttonFocused {
		t.Error("Expected buttons not focused initially")
	}

	// Press Tab to move to buttons
	cmd := step.Update(tea.KeyPressMsg{Text: "tab"})
	if cmd != nil {
		t.Error("Expected nil command from tab")
	}

	// Buttons should now be focused
	if !step.buttonFocused {
		t.Error("Expected buttons to be focused after Tab")
	}
	if !step.buttonBar.IsFocused() {
		t.Error("Expected button bar to be focused")
	}
}

func TestReviewStep_ButtonActivation(t *testing.T) {
	tests := []struct {
		name          string
		focusedButton int // 0 = Restart, 1 = Save
		expectedMsg   string
		expectModal   bool
	}{
		{
			name:          "activate restart button",
			focusedButton: 0,
			expectModal:   true,
		},
		{
			name:          "activate save button",
			focusedButton: 1,
			expectedMsg:   "SaveSpecMsg",
			expectModal:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "# Test Spec"
			cfg := &config.Config{}
			step := NewReviewStep(content, cfg)
			step.SetSize(80, 20)

			// Focus buttons
			step.buttonFocused = true
			if tt.focusedButton == 0 {
				step.buttonBar.FocusFirst()
			} else {
				step.buttonBar.FocusLast()
			}

			// Activate button with Enter
			cmd := step.Update(tea.KeyPressMsg{Text: "enter"})

			if tt.expectModal {
				// Should show confirmation modal
				if !step.showConfirmRestart {
					t.Error("Expected confirmation modal to be shown")
				}
				if cmd != nil {
					t.Error("Expected nil command when showing modal")
				}
			} else {
				// Should return SaveSpecMsg
				if cmd == nil {
					t.Fatal("Expected command from save button")
				}
				msg := cmd()
				if _, ok := msg.(SaveSpecMsg); !ok {
					t.Errorf("Expected SaveSpecMsg, got %T", msg)
				}
			}
		})
	}
}

func TestReviewStep_RestartConfirmation(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		expectMsg bool
		msgType   string
	}{
		{
			name:      "confirm with Y",
			key:       "Y",
			expectMsg: true,
			msgType:   "RestartWizardMsg",
		},
		{
			name:      "confirm with y",
			key:       "y",
			expectMsg: true,
			msgType:   "RestartWizardMsg",
		},
		{
			name:      "cancel with N",
			key:       "N",
			expectMsg: false,
		},
		{
			name:      "cancel with n",
			key:       "n",
			expectMsg: false,
		},
		{
			name:      "cancel with ESC",
			key:       "esc",
			expectMsg: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "# Test Spec"
			cfg := &config.Config{}
			step := NewReviewStep(content, cfg)
			step.SetSize(80, 20)

			// Show confirmation modal
			step.showConfirmRestart = true

			// Send key
			cmd := step.Update(tea.KeyPressMsg{Text: tt.key})

			// Modal should be hidden
			if step.showConfirmRestart {
				t.Error("Expected confirmation modal to be hidden")
			}

			if tt.expectMsg {
				if cmd == nil {
					t.Fatal("Expected command from confirmation")
				}
				msg := cmd()
				if _, ok := msg.(RestartWizardMsg); !ok {
					t.Errorf("Expected RestartWizardMsg, got %T", msg)
				}
			} else {
				if cmd != nil {
					t.Error("Expected nil command from cancel")
				}
			}
		})
	}
}

func TestReviewStep_ConfirmationModalView(t *testing.T) {
	content := "# Test Spec"
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)
	step.SetSize(80, 20)

	// Show confirmation modal
	step.showConfirmRestart = true

	// View should show modal
	view := step.View()
	if !strings.Contains(view, "Restart Wizard") {
		t.Error("Expected view to contain 'Restart Wizard' in modal")
	}
	if !strings.Contains(view, "Press Y to restart") {
		t.Error("Expected view to contain restart instructions")
	}
	if !strings.Contains(view, "discard") {
		t.Error("Expected view to contain warning about discarding")
	}

	// Should not show viewport content when modal is visible
	if strings.Contains(view, "Test Spec") {
		t.Error("Expected viewport content to be hidden when modal is visible")
	}
}

func TestReviewStep_ConfirmationModalBlocksInput(t *testing.T) {
	content := "# Test Spec"
	cfg := &config.Config{}
	step := NewReviewStep(content, cfg)
	step.SetSize(80, 20)

	// Show confirmation modal
	step.showConfirmRestart = true

	// Try to scroll (should be blocked)
	cmd := step.Update(tea.KeyPressMsg{Text: "down"})
	if cmd != nil {
		t.Error("Expected input to be blocked by modal")
	}

	// Try to open editor (should be blocked)
	if err := os.Setenv("EDITOR", "vim"); err != nil {
		t.Fatalf("Failed to set EDITOR: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("EDITOR")
	}()

	cmd = step.Update(tea.KeyPressMsg{Text: "e"})
	if cmd != nil {
		t.Error("Expected editor key to be blocked by modal")
	}

	// Modal should still be shown
	if !step.showConfirmRestart {
		t.Error("Expected modal to still be shown")
	}
}
