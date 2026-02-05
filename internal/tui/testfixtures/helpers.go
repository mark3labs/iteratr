package testfixtures

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
)

// Initialize test environment
func init() {
	// Set Ascii profile to disable color output for consistent golden files across CI/platforms
	lipgloss.Writer.Profile = colorprofile.Ascii
}

// Canonical terminal size for all tests
const (
	TestTermWidth  = 120
	TestTermHeight = 40
)

// Conservative timeout for WaitFor (CI compatibility)
const (
	DefaultWaitDuration  = 5 * time.Second
	DefaultCheckInterval = 100 * time.Millisecond
)

// Flag for updating golden files (shared across all tests)
var UpdateGolden = flag.Bool("update", false, "update golden files")

// CompareGolden compares actual output with golden file.
// Replaces duplicate compareGolden functions across test files.
// Use -update flag to regenerate golden files.
func CompareGolden(t *testing.T, goldenPath, actual string) {
	t.Helper()

	// Update golden file if -update flag is set
	if *UpdateGolden {
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
	if actual != string(expected) {
		t.Errorf("output does not match golden file %s\n\nExpected:\n%s\n\nActual:\n%s",
			goldenPath, string(expected), actual)
	}
}

// RetryTest retries a test function up to maxAttempts times if it fails.
// Useful for handling flaky tests due to timing issues.
func RetryTest(t *testing.T, maxAttempts int, fn func() error) {
	t.Helper()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := fn(); err == nil {
			return // Test passed
		} else {
			lastErr = err
			if attempt < maxAttempts {
				t.Logf("Attempt %d/%d failed: %v (retrying...)", attempt, maxAttempts, err)
			}
		}
	}
	// All attempts failed
	t.Fatalf("Test failed after %d attempts: %v", maxAttempts, lastErr)
}

// GoldenPath builds a path to a golden file in the testdata directory.
// Example: GoldenPath("status_idle.golden") -> "testdata/status_idle.golden"
func GoldenPath(filename string) string {
	return filepath.Join("testdata", filename)
}

// Contains checks if a string contains a substring.
// This is a simple helper to make test assertions more readable.
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ErrSessionNotFound creates a "session not found" error with the given session ID.
// Useful for testing error handling in components that load sessions.
func ErrSessionNotFound(sessionID string) error {
	return fmt.Errorf("session not found: %s", sessionID)
}

// ErrStreamError creates a stream error.
// Useful for testing error handling in components that consume event streams.
func ErrStreamError() error {
	return errors.New("stream error: EOF")
}

// CompareRendered creates a screen buffer, renders content, and compares with golden file.
// This consolidates the common pattern of:
//
//	canvas := uv.NewScreenBuffer(TestTermWidth, TestTermHeight)
//	content.Render(canvas)
//	testfixtures.CompareGolden(t, goldenPath, canvas.Render())
func CompareRendered(t *testing.T, goldenPath string, renderFn func(canvas uv.ScreenBuffer)) {
	t.Helper()
	canvas := uv.NewScreenBuffer(TestTermWidth, TestTermHeight)
	renderFn(canvas)
	CompareGolden(t, goldenPath, canvas.Render())
}
