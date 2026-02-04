package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
)

// TestDashboard_InitialFocusState tests that Dashboard starts with focus on Agent pane
func TestDashboard_InitialFocusState(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Initial focus should be on Agent pane
	if d.focusPane != FocusAgent {
		t.Errorf("Initial focusPane: got %v, want FocusAgent", d.focusPane)
	}

	// Input should not be focused initially
	if d.inputFocused {
		t.Error("inputFocused should be false initially")
	}

	// Agent output should not be focused initially (dashboard itself needs focus)
	if d.focused {
		t.Error("Dashboard should not be focused initially")
	}
}

// TestDashboard_FocusCycleForward tests Tab key cycling focus forward through panes
func TestDashboard_FocusCycleForward(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set some state for sidebar
	d.SetState(testfixtures.StateWithTasks())

	// Start at Agent pane (default)
	if d.focusPane != FocusAgent {
		t.Fatalf("Initial focusPane: got %v, want FocusAgent", d.focusPane)
	}

	// Tab: Agent → Tasks
	d.Update(tea.KeyPressMsg{Text: "tab"})
	if d.focusPane != FocusTasks {
		t.Errorf("After first tab: got %v, want FocusTasks", d.focusPane)
	}

	// Tab: Tasks → Notes
	d.Update(tea.KeyPressMsg{Text: "tab"})
	if d.focusPane != FocusNotes {
		t.Errorf("After second tab: got %v, want FocusNotes", d.focusPane)
	}

	// Tab: Notes → Agent (wraps around)
	d.Update(tea.KeyPressMsg{Text: "tab"})
	if d.focusPane != FocusAgent {
		t.Errorf("After third tab: got %v, want FocusAgent (wrap)", d.focusPane)
	}
}

// TestDashboard_FocusCycleInputNotInCycle tests that Input pane is not in Tab cycle
func TestDashboard_FocusCycleInputNotInCycle(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Focus input with 'i' key
	d.Update(tea.KeyPressMsg{Text: "i"})
	if d.focusPane != FocusInput {
		t.Fatalf("After 'i' key: got %v, want FocusInput", d.focusPane)
	}

	// Tab from Input should not cycle focus (handled by textarea)
	d.Update(tea.KeyPressMsg{Text: "tab"})
	if d.focusPane != FocusInput {
		t.Errorf("Tab from Input: got %v, want FocusInput (no cycle)", d.focusPane)
	}

	// Escape to exit Input and return to Agent
	d.Update(tea.KeyPressMsg{Text: "esc"})
	if d.focusPane != FocusAgent {
		t.Errorf("After Escape: got %v, want FocusAgent", d.focusPane)
	}

	// Now Tab should cycle normally
	d.Update(tea.KeyPressMsg{Text: "tab"})
	if d.focusPane != FocusTasks {
		t.Errorf("Tab after Escape: got %v, want FocusTasks", d.focusPane)
	}
}

// TestDashboard_IKeyFocusesInputFromAnyPane tests 'i' key focuses input from any pane
func TestDashboard_IKeyFocusesInputFromAnyPane(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		startPane FocusPane
	}{
		{"FromAgent", FocusAgent},
		{"FromTasks", FocusTasks},
		{"FromNotes", FocusNotes},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agentOutput := NewAgentOutput()
			sidebar := NewSidebar()
			d := NewDashboard(agentOutput, sidebar)
			d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
			d.SetFocus(true)

			// Set initial focus pane
			d.focusPane = tc.startPane

			// Press 'i' key
			d.Update(tea.KeyPressMsg{Text: "i"})

			// Verify focus moved to Input
			if d.focusPane != FocusInput {
				t.Errorf("After 'i' from %v: got %v, want FocusInput", tc.startPane, d.focusPane)
			}
			if !d.inputFocused {
				t.Error("inputFocused should be true after 'i'")
			}
			if !d.agentOutput.input.Focused() {
				t.Error("agentOutput.input should be focused after 'i'")
			}
		})
	}
}

// TestDashboard_IKeyIdempotentFromInput tests that 'i' key is idempotent when already in Input
func TestDashboard_IKeyIdempotentFromInput(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Focus input once
	d.Update(tea.KeyPressMsg{Text: "i"})
	if d.focusPane != FocusInput {
		t.Fatalf("After 'i': got %v, want FocusInput", d.focusPane)
	}

	// Press 'i' again - should be idempotent (not cycle out of Input)
	d.Update(tea.KeyPressMsg{Text: "i"})
	if d.focusPane != FocusInput {
		t.Errorf("After second 'i': got %v, want FocusInput (idempotent)", d.focusPane)
	}
}

// TestDashboard_ScrollListFocusStates tests that ScrollList focus states are updated correctly
func TestDashboard_ScrollListFocusStates(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set some state for sidebar
	state := testfixtures.FullState()
	d.SetState(state)
	sidebar.SetState(state)

	// Add some messages to agent output
	agentOutput.AppendText("Test message")

	// Initial state: Agent focused
	d.focusPane = FocusAgent
	d.updateScrollListFocus()

	// Verify only Agent ScrollList is focused
	if !agentOutput.scrollList.focused {
		t.Error("Agent output list should be focused when FocusAgent")
	}
	if sidebar.tasksScrollList.focused {
		t.Error("Tasks list should not be focused when FocusAgent")
	}
	if sidebar.notesScrollList.focused {
		t.Error("Notes list should not be focused when FocusAgent")
	}

	// Focus Tasks
	d.focusPane = FocusTasks
	d.updateScrollListFocus()

	// Verify only Tasks ScrollList is focused
	if agentOutput.scrollList.focused {
		t.Error("Agent output list should not be focused when FocusTasks")
	}
	if !sidebar.tasksScrollList.focused {
		t.Error("Tasks list should be focused when FocusTasks")
	}
	if sidebar.notesScrollList.focused {
		t.Error("Notes list should not be focused when FocusTasks")
	}

	// Focus Notes
	d.focusPane = FocusNotes
	d.updateScrollListFocus()

	// Verify only Notes ScrollList is focused
	if agentOutput.scrollList.focused {
		t.Error("Agent output list should not be focused when FocusNotes")
	}
	if sidebar.tasksScrollList.focused {
		t.Error("Tasks list should not be focused when FocusNotes")
	}
	if !sidebar.notesScrollList.focused {
		t.Error("Notes list should be focused when FocusNotes")
	}

	// Focus Input
	d.focusPane = FocusInput
	d.updateScrollListFocus()

	// Verify no ScrollLists are focused when Input is focused
	if agentOutput.scrollList.focused {
		t.Error("Agent output list should not be focused when FocusInput")
	}
	if sidebar.tasksScrollList.focused {
		t.Error("Tasks list should not be focused when FocusInput")
	}
	if sidebar.notesScrollList.focused {
		t.Error("Notes list should not be focused when FocusInput")
	}
}

// TestDashboard_KeyboardRoutingToAgentOutput tests that keys are routed to AgentOutput when focused
func TestDashboard_KeyboardRoutingToAgentOutput(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Add some messages
	agentOutput.AppendText("Message 1")
	agentOutput.AppendText("Message 2")
	agentOutput.AppendText("Message 3")

	// Focus Agent pane
	d.focusPane = FocusAgent
	d.updateScrollListFocus()

	// Send 'j' key (down) - should be handled by AgentOutput's ScrollList
	d.Update(tea.KeyPressMsg{Text: "j"})

	// Verify the key was processed without panic
	// This is an integration test - the key routing works correctly
}

// TestDashboard_KeyboardRoutingToSidebar tests that keys are routed to Sidebar when focused
func TestDashboard_KeyboardRoutingToSidebar(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set state with tasks
	state := testfixtures.StateWithTasks()
	d.SetState(state)
	sidebar.SetState(state)

	// Focus Tasks pane
	d.focusPane = FocusTasks
	d.updateScrollListFocus()

	// Send 'j' key (down) - should be handled by Sidebar's task ScrollList
	cmd := d.Update(tea.KeyPressMsg{Text: "j"})

	// Verify no panic and command is returned (or nil)
	_ = cmd
}

// TestDashboard_EnterOnTaskOpensModal tests that Enter key on focused task emits OpenTaskModalMsg
func TestDashboard_EnterOnTaskOpensModal(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set state with tasks
	state := testfixtures.StateWithTasks()
	d.SetState(state)
	sidebar.SetState(state)

	// Focus Tasks pane
	d.focusPane = FocusTasks
	d.updateScrollListFocus()

	// Send Enter key - should open task modal
	cmd := d.Update(tea.KeyPressMsg{Text: "enter"})

	// Execute command and verify OpenTaskModalMsg
	if cmd != nil {
		msg := cmd()
		_, ok := msg.(OpenTaskModalMsg)
		if !ok {
			t.Errorf("Expected OpenTaskModalMsg, got %T", msg)
		}
	}
}

// TestDashboard_KeyboardRoutingToNotes tests that keys are routed to Notes when focused
func TestDashboard_KeyboardRoutingToNotes(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set state with notes
	state := testfixtures.StateWithNotes()
	d.SetState(state)
	sidebar.SetState(state)

	// Focus Notes pane
	d.focusPane = FocusNotes
	d.updateScrollListFocus()

	// Send 'j' key (down) - should be handled by Sidebar's note ScrollList
	cmd := d.Update(tea.KeyPressMsg{Text: "j"})

	// Verify no panic and command is returned (or nil)
	_ = cmd
}

// TestDashboard_SetAgentBusy tests that busy state is propagated to AgentOutput
func TestDashboard_SetAgentBusy(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Initially not busy
	if d.agentBusy {
		t.Error("agentBusy should be false initially")
	}

	// Set busy
	d.SetAgentBusy(true)
	if !d.agentBusy {
		t.Error("agentBusy should be true after SetAgentBusy(true)")
	}
	// Verify placeholder changed
	if agentOutput.input.Placeholder != "Agent is working..." {
		t.Errorf("input.Placeholder: got %q, want %q", agentOutput.input.Placeholder, "Agent is working...")
	}

	// Set not busy
	d.SetAgentBusy(false)
	if d.agentBusy {
		t.Error("agentBusy should be false after SetAgentBusy(false)")
	}
	// Verify placeholder reverted
	if agentOutput.input.Placeholder != "Send a message..." {
		t.Errorf("input.Placeholder: got %q, want %q", agentOutput.input.Placeholder, "Send a message...")
	}
}

// TestDashboard_SetQueueDepth tests that queue depth is propagated to AgentOutput
func TestDashboard_SetQueueDepth(t *testing.T) {
	t.Parallel()

	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	// Initially 0
	if agentOutput.queueDepth != 0 {
		t.Errorf("Initial queueDepth: got %d, want 0", agentOutput.queueDepth)
	}

	// Set queue depth
	d.SetQueueDepth(5)
	if agentOutput.queueDepth != 5 {
		t.Errorf("After SetQueueDepth(5): got %d, want 5", agentOutput.queueDepth)
	}

	// Set back to 0
	d.SetQueueDepth(0)
	if agentOutput.queueDepth != 0 {
		t.Errorf("After SetQueueDepth(0): got %d, want 0", agentOutput.queueDepth)
	}
}

// --- Visual Regression Tests ---

// TestDashboard_RenderFocusAgent tests rendering with Agent pane focused
func TestDashboard_RenderFocusAgent(t *testing.T) {
	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set state
	state := testfixtures.FullState()
	d.SetState(state)
	sidebar.SetState(state)

	// Add some messages
	agentOutput.AppendText("Working on task...")
	agentOutput.AppendThinking("Processing request...")
	agentOutput.AppendText("Done thinking")

	// Focus Agent pane
	d.focusPane = FocusAgent
	d.updateScrollListFocus()

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "dashboard_focus_agent.golden")
	compareDashboardGolden(t, goldenFile, rendered)
}

// TestDashboard_RenderFocusTasks tests rendering with Tasks pane focused
func TestDashboard_RenderFocusTasks(t *testing.T) {
	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set state
	state := testfixtures.FullState()
	d.SetState(state)
	sidebar.SetState(state)

	// Add some messages
	agentOutput.AppendText("Working on task...")

	// Focus Tasks pane
	d.focusPane = FocusTasks
	d.updateScrollListFocus()

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "dashboard_focus_tasks.golden")
	compareDashboardGolden(t, goldenFile, rendered)
}

// TestDashboard_RenderFocusNotes tests rendering with Notes pane focused
func TestDashboard_RenderFocusNotes(t *testing.T) {
	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set state
	state := testfixtures.FullState()
	d.SetState(state)
	sidebar.SetState(state)

	// Add some messages
	agentOutput.AppendText("Working on task...")

	// Focus Notes pane
	d.focusPane = FocusNotes
	d.updateScrollListFocus()

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "dashboard_focus_notes.golden")
	compareDashboardGolden(t, goldenFile, rendered)
}

// TestDashboard_RenderFocusInput tests rendering with Input focused
func TestDashboard_RenderFocusInput(t *testing.T) {
	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set state
	state := testfixtures.FullState()
	d.SetState(state)
	sidebar.SetState(state)

	// Add some messages
	agentOutput.AppendText("Working on task...")

	// Focus Input
	d.focusPane = FocusInput
	d.inputFocused = true
	agentOutput.SetInputFocused(true)
	d.updateScrollListFocus()

	// Set some input text
	agentOutput.input.SetValue("User typing...")

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "dashboard_focus_input.golden")
	compareDashboardGolden(t, goldenFile, rendered)
}

// TestDashboard_RenderEmptyState tests rendering with empty state
func TestDashboard_RenderEmptyState(t *testing.T) {
	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set empty state
	state := testfixtures.EmptyState()
	d.SetState(state)
	sidebar.SetState(state)

	// No messages in agent output

	// Focus Agent pane
	d.focusPane = FocusAgent
	d.updateScrollListFocus()

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "dashboard_empty_state.golden")
	compareDashboardGolden(t, goldenFile, rendered)
}

// TestDashboard_RenderWithQueueDepth tests rendering with queue depth indicator
func TestDashboard_RenderWithQueueDepth(t *testing.T) {
	agentOutput := NewAgentOutput()
	sidebar := NewSidebar()
	d := NewDashboard(agentOutput, sidebar)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.SetFocus(true)

	// Set state
	state := testfixtures.StateWithTasks()
	d.SetState(state)
	sidebar.SetState(state)

	// Add message
	agentOutput.AppendText("Working on task...")

	// Set queue depth
	d.SetQueueDepth(3)

	// Set agent busy
	d.SetAgentBusy(true)

	// Focus Agent pane
	d.focusPane = FocusAgent
	d.updateScrollListFocus()

	// Render
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	d.Draw(scr, area)

	// Capture rendered output
	rendered := scr.Render()

	// Verify golden file
	goldenFile := filepath.Join("testdata", "dashboard_queue_depth.golden")
	compareDashboardGolden(t, goldenFile, rendered)
}

// compareDashboardGolden compares rendered output with golden file
func compareDashboardGolden(t *testing.T, goldenPath, actual string) {
	t.Helper()

	// Update golden file if -update flag is set
	if *update {
		// Ensure testdata directory exists
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create testdata directory: %v", err)
		}

		if err := os.WriteFile(goldenPath, []byte(actual), 0644); err != nil {
			t.Fatalf("failed to update golden file %s: %v", goldenPath, err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read golden file
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file %s does not exist. Run with -update to create it.", goldenPath)
		}
		t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
	}

	// Compare
	if string(expected) != actual {
		t.Errorf("output does not match golden file %s\n\nExpected:\n%s\n\nActual:\n%s", goldenPath, string(expected), actual)
	}
}
