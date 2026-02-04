package testfixtures

import (
	"testing"
	"time"
)

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
