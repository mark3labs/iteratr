package tui

import (
	"testing"
)

func TestNewAgentOutput(t *testing.T) {
	ao := NewAgentOutput()

	if ao == nil {
		t.Fatal("expected non-nil agent output")
	}
	if ao.autoScroll != true {
		t.Error("expected autoScroll to be true by default")
	}
}

func TestAgentOutput_Append(t *testing.T) {
	ao := NewAgentOutput()

	// Append some content
	cmd := ao.Append("Test content")

	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestAgentOutput_UpdateSize(t *testing.T) {
	ao := NewAgentOutput()

	cmd := ao.UpdateSize(100, 50)

	// Command can be nil - just verify it doesn't panic
	_ = cmd

	if ao.width != 100 {
		t.Errorf("width: got %d, want 100", ao.width)
	}
	if ao.height != 50 {
		t.Errorf("height: got %d, want 50", ao.height)
	}
	if !ao.ready {
		t.Error("expected viewport to be ready after UpdateSize")
	}
}

func TestAgentOutput_Render(t *testing.T) {
	ao := NewAgentOutput()

	output := ao.Render()

	// Should render something even with no content
	if output == "" {
		t.Error("expected non-empty output")
	}
}
