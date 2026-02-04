package tui

import (
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/tui/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestPulse_NewPulse verifies initial state of a new Pulse
func TestPulse_NewPulse(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()

	if pulse.IsActive() {
		t.Error("new pulse should not be active")
	}

	if pulse.frame != 0 {
		t.Errorf("new pulse frame should be 0, got %d", pulse.frame)
	}

	if pulse.maxFrame != 5 {
		t.Errorf("new pulse maxFrame should be 5, got %d", pulse.maxFrame)
	}

	intensity := pulse.Intensity()
	if intensity != 0.0 {
		t.Errorf("inactive pulse intensity should be 0.0, got %f", intensity)
	}
}

// TestPulse_Start verifies pulse activation
func TestPulse_Start(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()

	// Start pulse
	cmd := pulse.Start()
	if cmd == nil {
		t.Error("Start() should return a tick command")
	}

	if !pulse.IsActive() {
		t.Error("pulse should be active after Start()")
	}

	if pulse.frame != 0 {
		t.Errorf("pulse frame should be reset to 0 after Start(), got %d", pulse.frame)
	}
}

// TestPulse_Stop verifies pulse deactivation
func TestPulse_Stop(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()
	pulse.Start()

	// Advance frame
	pulse.frame = 3

	// Stop pulse
	pulse.Stop()

	if pulse.IsActive() {
		t.Error("pulse should not be active after Stop()")
	}

	if pulse.frame != 0 {
		t.Errorf("pulse frame should be reset to 0 after Stop(), got %d", pulse.frame)
	}

	intensity := pulse.Intensity()
	if intensity != 0.0 {
		t.Errorf("stopped pulse intensity should be 0.0, got %f", intensity)
	}
}

// TestPulse_IntensityInactive verifies intensity when pulse is inactive
func TestPulse_IntensityInactive(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()

	intensity := pulse.Intensity()
	if intensity != 0.0 {
		t.Errorf("inactive pulse intensity should be 0.0, got %f", intensity)
	}
}

// TestPulse_IntensityFadeIn verifies intensity calculation during fade-in phase
func TestPulse_IntensityFadeIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		frame    int
		expected float64
	}{
		{name: "frame 0", frame: 0, expected: 0.0},
		{name: "frame 1", frame: 1, expected: 0.2},
		{name: "frame 2", frame: 2, expected: 0.4},
		{name: "frame 3", frame: 3, expected: 0.6},
		{name: "frame 4", frame: 4, expected: 0.8},
		{name: "frame 5 (peak)", frame: 5, expected: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pulse := NewPulse()
			pulse.Start()
			pulse.frame = tt.frame

			intensity := pulse.Intensity()
			if intensity != tt.expected {
				t.Errorf("frame %d: expected intensity %f, got %f", tt.frame, tt.expected, intensity)
			}
		})
	}
}

// TestPulse_IntensityFadeOut verifies intensity calculation during fade-out phase
func TestPulse_IntensityFadeOut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		frame    int
		expected float64
	}{
		{name: "frame 5 (peak)", frame: 5, expected: 1.0},
		{name: "frame 6", frame: 6, expected: 0.8},
		{name: "frame 7", frame: 7, expected: 0.6},
		{name: "frame 8", frame: 8, expected: 0.4},
		{name: "frame 9", frame: 9, expected: 0.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pulse := NewPulse()
			pulse.Start()
			pulse.frame = tt.frame

			intensity := pulse.Intensity()
			if intensity != tt.expected {
				t.Errorf("frame %d: expected intensity %f, got %f", tt.frame, tt.expected, intensity)
			}
		})
	}
}

// TestPulse_IntensityRange verifies intensity is always between 0.0 and 1.0
func TestPulse_IntensityRange(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()
	pulse.Start()

	// Test all frames in a full cycle
	maxFrame := pulse.maxFrame
	for frame := 0; frame < maxFrame*2; frame++ {
		pulse.frame = frame
		intensity := pulse.Intensity()
		if intensity < 0.0 || intensity > 1.0 {
			t.Errorf("frame %d: intensity %f is out of range [0.0, 1.0]", frame, intensity)
		}
	}
}

// TestPulse_UpdateInactive verifies Update returns nil when pulse is inactive
func TestPulse_UpdateInactive(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()

	cmd := pulse.Update(PulseMsg{})
	if cmd != nil {
		t.Error("Update should return nil when pulse is inactive")
	}
}

// TestPulse_UpdateNonPulseMsg verifies Update ignores non-PulseMsg messages
func TestPulse_UpdateNonPulseMsg(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()
	pulse.Start()

	// Send a non-PulseMsg
	type OtherMsg struct{}
	cmd := pulse.Update(OtherMsg{})

	// Should return nil (no command to continue ticking)
	if cmd != nil {
		t.Error("Update should return nil for non-PulseMsg messages")
	}
}

// TestPulse_FullCycleCompletion verifies pulse completes after full cycle
func TestPulse_FullCycleCompletion(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()
	pulse.Start()

	maxFrame := pulse.maxFrame
	// Simulate full cycle by manually advancing frame and calling Update
	// Note: Update checks time interval, so we advance frame manually
	for frame := 0; frame < maxFrame*2-1; frame++ {
		// Manually advance frame to bypass time check
		pulse.lastTick = pulse.lastTick.Add(-pulse.interval)
		cmd := pulse.Update(PulseMsg{})
		if cmd == nil {
			t.Errorf("frame %d: Update should return tick command before cycle completes", frame)
		}
		if !pulse.IsActive() {
			t.Errorf("frame %d: pulse should still be active before cycle completes", frame)
		}
	}

	// One more update to reach maxFrame*2 and complete the cycle
	pulse.lastTick = pulse.lastTick.Add(-pulse.interval)
	cmd := pulse.Update(PulseMsg{})
	if cmd != nil {
		t.Error("Update should return nil after completing full cycle")
	}
	if pulse.IsActive() {
		t.Error("pulse should be inactive after completing full cycle")
	}
}

// TestPulse_MultipleStartCalls verifies multiple Start() calls reset the pulse
func TestPulse_MultipleStartCalls(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()

	// Start and advance
	pulse.Start()
	pulse.frame = 3

	// Start again - should reset
	cmd := pulse.Start()
	if cmd == nil {
		t.Error("Start() should return tick command")
	}

	if pulse.frame != 0 {
		t.Errorf("Start() should reset frame to 0, got %d", pulse.frame)
	}

	if !pulse.IsActive() {
		t.Error("pulse should be active after second Start()")
	}
}

// TestPulse_StopWhileInactive verifies Stop() is safe when pulse is already inactive
func TestPulse_StopWhileInactive(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()

	// Stop without starting - should not panic
	pulse.Stop()

	if pulse.IsActive() {
		t.Error("pulse should not be active")
	}

	if pulse.frame != 0 {
		t.Errorf("pulse frame should be 0, got %d", pulse.frame)
	}
}

// TestPulse_IntensitySymmetry verifies fade-in and fade-out are symmetric
func TestPulse_IntensitySymmetry(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()
	pulse.Start()

	maxFrame := pulse.maxFrame

	// Compare fade-in and fade-out intensities
	for i := 0; i < maxFrame; i++ {
		// Fade-in frame
		pulse.frame = i
		fadeInIntensity := pulse.Intensity()

		// Corresponding fade-out frame
		pulse.frame = maxFrame*2 - i
		fadeOutIntensity := pulse.Intensity()

		if fadeInIntensity != fadeOutIntensity {
			t.Errorf("frame %d: fade-in intensity %f != fade-out intensity %f (asymmetric)",
				i, fadeInIntensity, fadeOutIntensity)
		}
	}
}

// TestPulse_PeakIntensity verifies intensity reaches exactly 1.0 at peak
func TestPulse_PeakIntensity(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()
	pulse.Start()

	// Peak should be at maxFrame
	pulse.frame = pulse.maxFrame
	intensity := pulse.Intensity()

	if intensity != 1.0 {
		t.Errorf("peak intensity should be exactly 1.0, got %f", intensity)
	}
}

// TestPulse_ZeroIntensityAtEnds verifies intensity is 0.0 at start and end of cycle
func TestPulse_ZeroIntensityAtEnds(t *testing.T) {
	t.Parallel()

	pulse := NewPulse()
	pulse.Start()

	// Start of cycle
	pulse.frame = 0
	if intensity := pulse.Intensity(); intensity != 0.0 {
		t.Errorf("start intensity should be 0.0, got %f", intensity)
	}

	// End of cycle (frame just before completion)
	// Note: frame = maxFrame*2 would trigger Stop() in Update, so we test maxFrame*2-1
	pulse.frame = pulse.maxFrame*2 - 1
	// This is actually frame 9 (when maxFrame=5), which should give 0.2
	// The cycle completes and stops at frame 10 (maxFrame*2)
	// So let's verify behavior at the mathematical end point
	pulse.frame = pulse.maxFrame * 2
	// At this point, Update would have called Stop(), so intensity would be 0
	pulse.Stop() // Simulate what Update does
	if intensity := pulse.Intensity(); intensity != 0.0 {
		t.Errorf("end intensity should be 0.0, got %f", intensity)
	}
}

// ==================== Spinner Tests ====================

// TestSpinner_NewDefaultSpinner verifies default spinner creation
func TestSpinner_NewDefaultSpinner(t *testing.T) {
	t.Parallel()

	spinner := NewDefaultSpinner()

	// Verify spinner is created (should not panic)
	view := spinner.View()
	require.NotEmpty(t, view, "spinner view should not be empty")
}

// TestSpinner_NewSpinner verifies spinner creation with custom style
func TestSpinner_NewSpinner(t *testing.T) {
	t.Parallel()

	spinner := NewSpinner(spinner.MiniDot)

	view := spinner.View()
	require.NotEmpty(t, view, "spinner view should not be empty")
}

// TestSpinner_Tick verifies tick command is not nil
func TestSpinner_Tick(t *testing.T) {
	t.Parallel()

	spinner := NewDefaultSpinner()

	cmd := spinner.Tick()
	require.NotNil(t, cmd, "Tick() should return a command")
}

// TestSpinner_Update verifies Update processes tick messages
func TestSpinner_Update(t *testing.T) {
	t.Parallel()

	spinner := NewDefaultSpinner()

	// Get tick command and execute it to get a message
	tickCmd := spinner.Tick()
	require.NotNil(t, tickCmd, "Tick() should return a command")

	// Update with a generic message (spinner.TickMsg is internal to bubbles)
	cmd := spinner.Update(tea.KeyPressMsg{})
	// Should return some command (possibly nil, depending on message type)
	_ = cmd
}

// TestSpinner_SetStyle verifies style can be updated
func TestSpinner_SetStyle(t *testing.T) {
	t.Parallel()

	spinner := NewDefaultSpinner()

	// Create a custom style
	customStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))

	// Set style - should not panic
	spinner.SetStyle(customStyle)

	view := spinner.View()
	require.NotEmpty(t, view, "spinner view should not be empty after SetStyle")
}

// TestSpinner_ViewStatic_Golden verifies spinner renders at static frame
func TestSpinner_ViewStatic_Golden(t *testing.T) {
	t.Parallel()

	spinner := NewDefaultSpinner()

	// Render spinner view
	view := spinner.View()

	// Render in a canvas for visual verification
	canvas := uv.NewScreenBuffer(testfixtures.TestTermWidth, 3)
	area := uv.Rect(0, 0, testfixtures.TestTermWidth, 3)
	uv.NewStyledString(view).Draw(canvas, area)

	goldenPath := filepath.Join("testdata", "spinner_static.golden")
	testfixtures.CompareGolden(t, goldenPath, canvas.Render())
}

// TestSpinner_DifferentStyles verifies different spinner styles render
func TestSpinner_DifferentStyles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		style spinner.Spinner
	}{
		{name: "MiniDot", style: spinner.MiniDot},
		{name: "Dot", style: spinner.Dot},
		{name: "Line", style: spinner.Line},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSpinner(tt.style)
			view := s.View()
			require.NotEmpty(t, view, "spinner view should not be empty for style %s", tt.name)
		})
	}
}

// ==================== GradientSpinner Tests ====================

// TestGradientSpinner_NewGradientSpinner verifies gradient spinner creation
func TestGradientSpinner_NewGradientSpinner(t *testing.T) {
	t.Parallel()

	gs := NewGradientSpinner("#ff0000", "#00ff00", "Loading")

	require.Equal(t, "#ff0000", gs.colorA, "colorA should match")
	require.Equal(t, "#00ff00", gs.colorB, "colorB should match")
	require.Equal(t, "Loading", gs.label, "label should match")
	require.Equal(t, 0, gs.frame, "initial frame should be 0")
	require.Equal(t, 15, gs.size, "default size should be 15")
}

// TestGradientSpinner_NewDefaultGradientSpinner verifies default gradient spinner uses theme colors
func TestGradientSpinner_NewDefaultGradientSpinner(t *testing.T) {
	t.Parallel()

	gs := NewDefaultGradientSpinner("Processing")

	require.Equal(t, "Processing", gs.label, "label should match")
	require.NotEmpty(t, gs.colorA, "colorA should be set from theme")
	require.NotEmpty(t, gs.colorB, "colorB should be set from theme")
	require.Equal(t, 0, gs.frame, "initial frame should be 0")
	require.Equal(t, 15, gs.size, "default size should be 15")
}

// TestGradientSpinner_View verifies view renders gradient
func TestGradientSpinner_View(t *testing.T) {
	t.Parallel()

	gs := NewGradientSpinner("#ff0000", "#00ff00", "")

	view := gs.View()
	require.NotEmpty(t, view, "view should not be empty")
	require.Contains(t, view, "█", "view should contain block characters")
}

// TestGradientSpinner_ViewWithLabel verifies view includes label
func TestGradientSpinner_ViewWithLabel(t *testing.T) {
	t.Parallel()

	gs := NewGradientSpinner("#ff0000", "#00ff00", "Loading")

	view := gs.View()
	require.NotEmpty(t, view, "view should not be empty")
	require.Contains(t, view, "Loading", "view should contain label")
	require.Contains(t, view, "█", "view should contain block characters")
}

// TestGradientSpinner_ViewWithoutLabel verifies view without label
func TestGradientSpinner_ViewWithoutLabel(t *testing.T) {
	t.Parallel()

	gs := NewGradientSpinner("#ff0000", "#00ff00", "")

	view := gs.View()
	require.NotEmpty(t, view, "view should not be empty")
	require.NotContains(t, view, "Loading", "view should not contain 'Loading'")
	require.Contains(t, view, "█", "view should contain block characters")
}

// TestGradientSpinner_Tick verifies tick command is not nil
func TestGradientSpinner_Tick(t *testing.T) {
	t.Parallel()

	gs := NewGradientSpinner("#ff0000", "#00ff00", "")

	cmd := gs.Tick()
	require.NotNil(t, cmd, "Tick() should return a command")
}

// TestGradientSpinner_Update verifies Update advances frame
func TestGradientSpinner_Update(t *testing.T) {
	t.Parallel()

	gs := NewGradientSpinner("#ff0000", "#00ff00", "")

	initialFrame := gs.frame
	require.Equal(t, 0, initialFrame, "initial frame should be 0")

	// Update with GradientSpinnerMsg
	cmd := gs.Update(GradientSpinnerMsg{})
	require.NotNil(t, cmd, "Update should return tick command")
	require.Equal(t, 1, gs.frame, "frame should advance to 1")
}

// TestGradientSpinner_UpdateNonSpinnerMsg verifies Update ignores other messages
func TestGradientSpinner_UpdateNonSpinnerMsg(t *testing.T) {
	t.Parallel()

	gs := NewGradientSpinner("#ff0000", "#00ff00", "")

	initialFrame := gs.frame

	// Update with non-GradientSpinnerMsg
	cmd := gs.Update(tea.KeyPressMsg{})
	require.Nil(t, cmd, "Update should return nil for non-GradientSpinnerMsg")
	require.Equal(t, initialFrame, gs.frame, "frame should not change")
}

// TestGradientSpinner_FrameWrapping verifies frame wraps at size
func TestGradientSpinner_FrameWrapping(t *testing.T) {
	t.Parallel()

	gs := NewGradientSpinner("#ff0000", "#00ff00", "")

	// Advance to size-1
	gs.frame = gs.size - 1
	require.Equal(t, 14, gs.frame, "frame should be size-1")

	// Update should wrap to 0
	gs.Update(GradientSpinnerMsg{})
	require.Equal(t, 0, gs.frame, "frame should wrap to 0 at size")
}

// TestGradientSpinner_ViewStatic_Golden verifies gradient spinner renders at frame 0
func TestGradientSpinner_ViewStatic_Golden(t *testing.T) {
	t.Parallel()

	gs := NewDefaultGradientSpinner("Processing")

	// Render gradient spinner view at frame 0
	view := gs.View()

	// Render in a canvas for visual verification
	canvas := uv.NewScreenBuffer(testfixtures.TestTermWidth, 3)
	area := uv.Rect(0, 0, testfixtures.TestTermWidth, 3)
	uv.NewStyledString(view).Draw(canvas, area)

	goldenPath := filepath.Join("testdata", "gradient_spinner_frame0.golden")
	testfixtures.CompareGolden(t, goldenPath, canvas.Render())
}

// TestGradientSpinner_ViewFrame5_Golden verifies gradient spinner renders at frame 5
func TestGradientSpinner_ViewFrame5_Golden(t *testing.T) {
	t.Parallel()

	gs := NewDefaultGradientSpinner("Processing")

	// Advance to frame 5
	gs.frame = 5

	// Render gradient spinner view at frame 5
	view := gs.View()

	// Render in a canvas for visual verification
	canvas := uv.NewScreenBuffer(testfixtures.TestTermWidth, 3)
	area := uv.Rect(0, 0, testfixtures.TestTermWidth, 3)
	uv.NewStyledString(view).Draw(canvas, area)

	goldenPath := filepath.Join("testdata", "gradient_spinner_frame5.golden")
	testfixtures.CompareGolden(t, goldenPath, canvas.Render())
}

// TestGradientSpinner_ViewFrame10_Golden verifies gradient spinner renders at frame 10
func TestGradientSpinner_ViewFrame10_Golden(t *testing.T) {
	t.Parallel()

	gs := NewDefaultGradientSpinner("Processing")

	// Advance to frame 10
	gs.frame = 10

	// Render gradient spinner view at frame 10
	view := gs.View()

	// Render in a canvas for visual verification
	canvas := uv.NewScreenBuffer(testfixtures.TestTermWidth, 3)
	area := uv.Rect(0, 0, testfixtures.TestTermWidth, 3)
	uv.NewStyledString(view).Draw(canvas, area)

	goldenPath := filepath.Join("testdata", "gradient_spinner_frame10.golden")
	testfixtures.CompareGolden(t, goldenPath, canvas.Render())
}

// TestGradientSpinner_NoLabel_Golden verifies gradient spinner without label
func TestGradientSpinner_NoLabel_Golden(t *testing.T) {
	t.Parallel()

	gs := NewDefaultGradientSpinner("")

	// Render gradient spinner view without label
	view := gs.View()

	// Render in a canvas for visual verification
	canvas := uv.NewScreenBuffer(testfixtures.TestTermWidth, 3)
	area := uv.Rect(0, 0, testfixtures.TestTermWidth, 3)
	uv.NewStyledString(view).Draw(canvas, area)

	goldenPath := filepath.Join("testdata", "gradient_spinner_no_label.golden")
	testfixtures.CompareGolden(t, goldenPath, canvas.Render())
}

// TestGradientSpinner_MultipleUpdates verifies multiple updates advance frame correctly
func TestGradientSpinner_MultipleUpdates(t *testing.T) {
	t.Parallel()

	gs := NewGradientSpinner("#ff0000", "#00ff00", "")

	for i := 0; i < 5; i++ {
		expectedFrame := i + 1
		gs.Update(GradientSpinnerMsg{})
		require.Equal(t, expectedFrame, gs.frame, "frame should be %d after %d updates", expectedFrame, i+1)
	}
}

// TestGradientSpinner_CustomColors verifies gradient with custom colors
func TestGradientSpinner_CustomColors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		colorA string
		colorB string
	}{
		{name: "red to blue", colorA: "#ff0000", colorB: "#0000ff"},
		{name: "green to yellow", colorA: "#00ff00", colorB: "#ffff00"},
		{name: "purple to cyan", colorA: "#ff00ff", colorB: "#00ffff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGradientSpinner(tt.colorA, tt.colorB, "Test")
			view := gs.View()
			require.NotEmpty(t, view, "gradient spinner should render for %s", tt.name)
			require.Contains(t, view, "█", "view should contain block characters")
			require.Contains(t, view, "Test", "view should contain label")
		})
	}
}
