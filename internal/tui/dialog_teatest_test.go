package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// --- Dialog Unit Tests ---

func TestDialog_Initialization(t *testing.T) {
	t.Parallel()

	d := NewDialog()

	require.NotNil(t, d, "dialog should be initialized")
	require.False(t, d.IsVisible(), "dialog should not be visible initially")
	require.Equal(t, "OK", d.button, "dialog should have default button text 'OK'")
	require.Equal(t, "", d.title, "dialog should have empty title initially")
	require.Equal(t, "", d.message, "dialog should have empty message initially")
	require.Nil(t, d.onClose, "dialog should have no onClose callback initially")
}

func TestDialog_Show(t *testing.T) {
	t.Parallel()

	d := NewDialog()

	// Track callback invocation
	callbackCalled := false
	callback := func() tea.Cmd {
		callbackCalled = true
		return nil
	}

	d.Show("Test Title", "Test message content", callback)

	require.True(t, d.IsVisible(), "dialog should be visible after Show")
	require.Equal(t, "Test Title", d.title, "dialog should have correct title")
	require.Equal(t, "Test message content", d.message, "dialog should have correct message")
	require.NotNil(t, d.onClose, "dialog should have onClose callback")

	// Verify callback works
	cmd := d.onClose()
	require.Nil(t, cmd, "callback should return nil")
	require.True(t, callbackCalled, "callback should be invoked")
}

func TestDialog_ShowWithNilCallback(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	d.Show("Title", "Message", nil)

	require.True(t, d.IsVisible(), "dialog should be visible")
	require.Nil(t, d.onClose, "dialog should accept nil callback")
}

func TestDialog_Hide(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	d.Show("Title", "Message", nil)
	require.True(t, d.IsVisible(), "dialog should be visible after Show")

	d.Hide()
	require.False(t, d.IsVisible(), "dialog should not be visible after Hide")
}

func TestDialog_SetSize(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	d.SetSize(100, 50)

	require.Equal(t, 100, d.width, "dialog should store width")
	require.Equal(t, 50, d.height, "dialog should store height")
}

func TestDialog_UpdateWhenInvisible(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	require.False(t, d.IsVisible(), "dialog should not be visible")

	// Update should return nil when not visible
	cmd := d.Update(tea.KeyPressMsg{Text: "enter"})
	require.Nil(t, cmd, "Update should return nil when dialog is not visible")
}

func TestDialog_UpdateEnterKey(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	callbackCalled := false
	callbackCmd := func() tea.Msg { return SessionCompleteMsg{} }
	onClose := func() tea.Cmd {
		callbackCalled = true
		return callbackCmd
	}

	d.Show("Title", "Message", onClose)
	require.True(t, d.IsVisible(), "dialog should be visible")

	// Press Enter key
	cmd := d.Update(tea.KeyPressMsg{Text: "enter"})

	require.False(t, d.IsVisible(), "dialog should be hidden after Enter key")
	require.True(t, callbackCalled, "onClose callback should be invoked")
	require.NotNil(t, cmd, "Update should return callback command")

	// Verify returned command produces expected message
	msg := cmd()
	_, ok := msg.(SessionCompleteMsg)
	require.True(t, ok, "callback command should return SessionCompleteMsg")
}

func TestDialog_UpdateSpaceKey(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	callbackCalled := false
	onClose := func() tea.Cmd {
		callbackCalled = true
		return nil
	}

	d.Show("Title", "Message", onClose)
	require.True(t, d.IsVisible(), "dialog should be visible")

	// Press Space key
	cmd := d.Update(tea.KeyPressMsg{Text: "space"})

	require.False(t, d.IsVisible(), "dialog should be hidden after Space key")
	require.True(t, callbackCalled, "onClose callback should be invoked")
	require.Nil(t, cmd, "Update should return nil when callback returns nil")
}

func TestDialog_UpdateEscKey(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	callbackCalled := false
	onClose := func() tea.Cmd {
		callbackCalled = true
		return nil
	}

	d.Show("Title", "Message", onClose)
	require.True(t, d.IsVisible(), "dialog should be visible")

	// Press Esc key
	cmd := d.Update(tea.KeyPressMsg{Text: "esc"})

	require.False(t, d.IsVisible(), "dialog should be hidden after Esc key")
	require.True(t, callbackCalled, "onClose callback should be invoked")
	require.Nil(t, cmd, "Update should return nil when callback returns nil")
}

func TestDialog_UpdateNilCallback(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	d.Show("Title", "Message", nil)
	require.True(t, d.IsVisible(), "dialog should be visible")

	// Press Enter with nil callback
	cmd := d.Update(tea.KeyPressMsg{Text: "enter"})

	require.False(t, d.IsVisible(), "dialog should be hidden after Enter key")
	require.Nil(t, cmd, "Update should return nil when callback is nil")
}

func TestDialog_UpdateOtherKeys(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	callbackCalled := false
	onClose := func() tea.Cmd {
		callbackCalled = true
		return nil
	}

	d.Show("Title", "Message", onClose)
	require.True(t, d.IsVisible(), "dialog should be visible")

	// Press other keys (should not close dialog)
	testKeys := []string{"a", "b", "1", "tab", "up", "down"}
	for _, key := range testKeys {
		t.Run(key, func(t *testing.T) {
			cmd := d.Update(tea.KeyPressMsg{Text: key})
			require.True(t, d.IsVisible(), "dialog should remain visible for key '%s'", key)
			require.False(t, callbackCalled, "callback should not be invoked for key '%s'", key)
			require.Nil(t, cmd, "Update should return nil for key '%s'", key)
		})
	}
}

func TestDialog_HandleClickWhenInvisible(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	require.False(t, d.IsVisible(), "dialog should not be visible")

	// HandleClick should return nil when not visible
	cmd := d.HandleClick(10, 10)
	require.Nil(t, cmd, "HandleClick should return nil when dialog is not visible")
	require.False(t, d.IsVisible(), "dialog should remain invisible")
}

func TestDialog_HandleClickAnywhere(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	callbackCalled := false
	callbackCmd := func() tea.Msg { return SessionCompleteMsg{} }
	onClose := func() tea.Cmd {
		callbackCalled = true
		return callbackCmd
	}

	d.Show("Title", "Message", onClose)
	d.SetSize(100, 50)
	require.True(t, d.IsVisible(), "dialog should be visible")

	// Click anywhere (dialog dismisses on any click)
	cmd := d.HandleClick(50, 25)

	require.False(t, d.IsVisible(), "dialog should be hidden after click")
	require.True(t, callbackCalled, "onClose callback should be invoked")
	require.NotNil(t, cmd, "HandleClick should return callback command")

	// Verify returned command produces expected message
	msg := cmd()
	_, ok := msg.(SessionCompleteMsg)
	require.True(t, ok, "callback command should return SessionCompleteMsg")
}

func TestDialog_HandleClickNilCallback(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	d.Show("Title", "Message", nil)
	require.True(t, d.IsVisible(), "dialog should be visible")

	// Click with nil callback
	cmd := d.HandleClick(10, 10)

	require.False(t, d.IsVisible(), "dialog should be hidden after click")
	require.Nil(t, cmd, "HandleClick should return nil when callback is nil")
}

func TestDialog_MultipleShowCalls(t *testing.T) {
	t.Parallel()

	d := NewDialog()

	// First Show
	callback1Called := false
	callback1 := func() tea.Cmd {
		callback1Called = true
		return nil
	}
	d.Show("Title 1", "Message 1", callback1)
	require.Equal(t, "Title 1", d.title, "dialog should have first title")
	require.Equal(t, "Message 1", d.message, "dialog should have first message")

	// Second Show (should replace callback)
	callback2Called := false
	callback2 := func() tea.Cmd {
		callback2Called = true
		return nil
	}
	d.Show("Title 2", "Message 2", callback2)
	require.Equal(t, "Title 2", d.title, "dialog should have second title")
	require.Equal(t, "Message 2", d.message, "dialog should have second message")

	// Close should invoke callback2, not callback1
	cmd := d.Update(tea.KeyPressMsg{Text: "enter"})
	require.False(t, callback1Called, "first callback should not be invoked")
	require.True(t, callback2Called, "second callback should be invoked")
	require.Nil(t, cmd, "Update should return nil")
}

func TestDialog_CallbackReturnsCommand(t *testing.T) {
	t.Parallel()

	d := NewDialog()

	// Create nested commands to verify command chain
	innerCalled := false
	innerCmd := func() tea.Msg {
		innerCalled = true
		return SessionCompleteMsg{}
	}

	onClose := func() tea.Cmd {
		return innerCmd
	}

	d.Show("Title", "Message", onClose)

	// Dismiss dialog via Enter key
	cmd := d.Update(tea.KeyPressMsg{Text: "enter"})
	require.NotNil(t, cmd, "Update should return command from callback")

	// Execute command and verify it runs
	msg := cmd()
	require.True(t, innerCalled, "inner command should be executed")
	_, ok := msg.(SessionCompleteMsg)
	require.True(t, ok, "command should return SessionCompleteMsg")
}

// --- Dialog Visual Rendering Tests ---

func TestDialog_DrawWhenInvisible(t *testing.T) {
	t.Parallel()

	d := NewDialog()
	require.False(t, d.IsVisible(), "dialog should not be visible")

	// Create screen and area
	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}

	// Draw should do nothing when invisible
	d.Draw(scr, area)

	// Verify screen is empty
	output := scr.String()
	require.Empty(t, strings.TrimSpace(output), "screen should be empty when dialog is not visible")
}

// --- Dialog Golden File Tests ---

func TestDialogGolden_SimpleMessage(t *testing.T) {
	d := NewDialog()
	d.Show("Success", "Operation completed successfully", nil)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}

	d.Draw(scr, area)
	output := scr.String()

	goldenFile := filepath.Join("testdata", "dialog_simple.golden")
	testfixtures.CompareGolden(t, goldenFile, output)
}

func TestDialogGolden_LongTitle(t *testing.T) {
	d := NewDialog()
	d.Show(
		"This is a very long dialog title that might affect layout",
		"Short message",
		nil,
	)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}

	d.Draw(scr, area)
	output := scr.String()

	goldenFile := filepath.Join("testdata", "dialog_long_title.golden")
	testfixtures.CompareGolden(t, goldenFile, output)
}

func TestDialogGolden_LongMessage(t *testing.T) {
	d := NewDialog()
	longMessage := "This is a longer message that spans multiple lines and should wrap properly within the dialog container. " +
		"It demonstrates how the dialog handles extended content gracefully."
	d.Show("Information", longMessage, nil)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}

	d.Draw(scr, area)
	output := scr.String()

	goldenFile := filepath.Join("testdata", "dialog_long_message.golden")
	testfixtures.CompareGolden(t, goldenFile, output)
}

func TestDialogGolden_ErrorDialog(t *testing.T) {
	d := NewDialog()
	d.Show("Error", "Failed to connect to server", nil)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}

	d.Draw(scr, area)
	output := scr.String()

	goldenFile := filepath.Join("testdata", "dialog_error.golden")
	testfixtures.CompareGolden(t, goldenFile, output)
}

func TestDialogGolden_MultiLineMessage(t *testing.T) {
	d := NewDialog()
	multiLineMsg := "Line 1: First line of message\nLine 2: Second line\nLine 3: Third line with more details"
	d.Show("Multi-line Dialog", multiLineMsg, nil)
	d.SetSize(testfixtures.TestTermWidth, testfixtures.TestTermHeight)

	scr := uv.NewScreenBuffer(testfixtures.TestTermWidth, testfixtures.TestTermHeight)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: testfixtures.TestTermWidth, Y: testfixtures.TestTermHeight},
	}

	d.Draw(scr, area)
	output := scr.String()

	goldenFile := filepath.Join("testdata", "dialog_multiline.golden")
	testfixtures.CompareGolden(t, goldenFile, output)
}

func TestDialogGolden_SmallScreen(t *testing.T) {
	d := NewDialog()
	d.Show("Confirm", "Are you sure you want to proceed?", nil)

	// Small screen size (60x20)
	smallWidth := 60
	smallHeight := 20
	d.SetSize(smallWidth, smallHeight)

	scr := uv.NewScreenBuffer(smallWidth, smallHeight)
	area := uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: smallWidth, Y: smallHeight},
	}

	d.Draw(scr, area)
	output := scr.String()

	goldenFile := filepath.Join("testdata", "dialog_small_screen.golden")
	testfixtures.CompareGolden(t, goldenFile, output)
}
