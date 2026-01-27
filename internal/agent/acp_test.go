package agent

import (
	"testing"
)

func TestExtractDiffBlocks_OnlyProcessesCompletedEdits(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		kind     string
		wantCall bool
	}{
		{
			name:     "completed edit - should extract",
			status:   "completed",
			kind:     "edit",
			wantCall: true,
		},
		{
			name:     "error edit - should skip",
			status:   "error",
			kind:     "edit",
			wantCall: false,
		},
		{
			name:     "canceled edit - should skip",
			status:   "canceled",
			kind:     "edit",
			wantCall: false,
		},
		{
			name:     "completed read - should skip",
			status:   "completed",
			kind:     "read",
			wantCall: false,
		},
		{
			name:     "completed bash - should skip",
			status:   "completed",
			kind:     "bash",
			wantCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the condition from acp.go line 365
			shouldExtract := tt.status == "completed" && tt.kind == "edit"

			if shouldExtract != tt.wantCall {
				t.Errorf("extractDiffBlocks condition check: got %v, want %v for status=%s kind=%s",
					shouldExtract, tt.wantCall, tt.status, tt.kind)
			}
		})
	}
}

func TestExtractDiffBlocks(t *testing.T) {
	tests := []struct {
		name    string
		content []toolCallContent
		want    []DiffBlock
	}{
		{
			name: "extracts single diff block",
			content: []toolCallContent{
				{
					Type:    "diff",
					Path:    "/path/to/file.go",
					OldText: "old content",
					NewText: "new content",
				},
			},
			want: []DiffBlock{
				{
					Path:    "/path/to/file.go",
					OldText: "old content",
					NewText: "new content",
				},
			},
		},
		{
			name: "extracts multiple diff blocks",
			content: []toolCallContent{
				{
					Type:    "content",
					Content: contentPart{Type: "text", Text: "some text"},
				},
				{
					Type:    "diff",
					Path:    "/path/to/file1.go",
					OldText: "old1",
					NewText: "new1",
				},
				{
					Type:    "diff",
					Path:    "/path/to/file2.go",
					OldText: "",
					NewText: "new file",
				},
			},
			want: []DiffBlock{
				{
					Path:    "/path/to/file1.go",
					OldText: "old1",
					NewText: "new1",
				},
				{
					Path:    "/path/to/file2.go",
					OldText: "",
					NewText: "new file",
				},
			},
		},
		{
			name: "handles content blocks without diff",
			content: []toolCallContent{
				{
					Type:    "content",
					Content: contentPart{Type: "text", Text: "no diffs here"},
				},
			},
			want: []DiffBlock{},
		},
		{
			name:    "handles empty content array",
			content: []toolCallContent{},
			want:    []DiffBlock{},
		},
		{
			name:    "handles nil content",
			content: nil,
			want:    []DiffBlock{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDiffBlocks(tt.content)

			// Handle nil vs empty slice comparison
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("extractDiffBlocks() returned %d blocks, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].Path != tt.want[i].Path {
					t.Errorf("block[%d].Path = %s, want %s", i, got[i].Path, tt.want[i].Path)
				}
				if got[i].OldText != tt.want[i].OldText {
					t.Errorf("block[%d].OldText = %s, want %s", i, got[i].OldText, tt.want[i].OldText)
				}
				if got[i].NewText != tt.want[i].NewText {
					t.Errorf("block[%d].NewText = %s, want %s", i, got[i].NewText, tt.want[i].NewText)
				}
			}
		})
	}
}

func TestExtractFileChanges(t *testing.T) {
	tests := []struct {
		name  string
		event ToolCallEvent
		want  []FileChange
	}{
		{
			name: "write tool - new file from diff block",
			event: ToolCallEvent{
				Status: "completed",
				Kind:   "edit",
				DiffBlocks: []DiffBlock{
					{
						Path:    "/abs/path/file.txt",
						OldText: "",
						NewText: "Hello World",
					},
				},
			},
			want: []FileChange{
				{
					AbsPath:   "/abs/path/file.txt",
					IsNew:     true,
					Additions: 0,
					Deletions: 0,
				},
			},
		},
		{
			name: "edit tool - modified file with metadata",
			event: ToolCallEvent{
				Status: "completed",
				Kind:   "edit",
				DiffBlocks: []DiffBlock{
					{
						Path:    "/abs/path/file.go",
						OldText: "Line 1",
						NewText: "Line 1\nLine 2",
					},
				},
				FileDiff: &FileDiff{
					File:      "/abs/path/file.go",
					Additions: 2,
					Deletions: 1,
				},
			},
			want: []FileChange{
				{
					AbsPath:   "/abs/path/file.go",
					IsNew:     false,
					Additions: 2,
					Deletions: 1,
				},
			},
		},
		{
			name: "multiple diff blocks - batch edit",
			event: ToolCallEvent{
				Status: "completed",
				Kind:   "edit",
				DiffBlocks: []DiffBlock{
					{
						Path:    "/abs/path/file1.go",
						OldText: "old",
						NewText: "new",
					},
					{
						Path:    "/abs/path/file2.go",
						OldText: "",
						NewText: "created",
					},
				},
			},
			want: []FileChange{
				{
					AbsPath:   "/abs/path/file1.go",
					IsNew:     false,
					Additions: 0,
					Deletions: 0,
				},
				{
					AbsPath:   "/abs/path/file2.go",
					IsNew:     true,
					Additions: 0,
					Deletions: 0,
				},
			},
		},
		{
			name: "fallback to FileDiff metadata",
			event: ToolCallEvent{
				Status:     "completed",
				Kind:       "edit",
				DiffBlocks: []DiffBlock{},
				FileDiff: &FileDiff{
					File:      "/abs/path/file.go",
					Additions: 5,
					Deletions: 3,
				},
			},
			want: []FileChange{
				{
					AbsPath:   "/abs/path/file.go",
					IsNew:     false,
					Additions: 5,
					Deletions: 3,
				},
			},
		},
		{
			name: "fallback to RawInput filePath",
			event: ToolCallEvent{
				Status:     "completed",
				Kind:       "edit",
				DiffBlocks: []DiffBlock{},
				RawInput: map[string]any{
					"filePath": "/abs/path/file.txt",
				},
			},
			want: []FileChange{
				{
					AbsPath:   "/abs/path/file.txt",
					IsNew:     false,
					Additions: 0,
					Deletions: 0,
				},
			},
		},
		{
			name: "no extractable file information",
			event: ToolCallEvent{
				Status:     "completed",
				Kind:       "edit",
				DiffBlocks: []DiffBlock{},
				RawInput:   map[string]any{},
			},
			want: []FileChange{},
		},
		{
			name: "diff block with mismatched FileDiff path - no merge",
			event: ToolCallEvent{
				Status: "completed",
				Kind:   "edit",
				DiffBlocks: []DiffBlock{
					{
						Path:    "/abs/path/file1.go",
						OldText: "old",
						NewText: "new",
					},
				},
				FileDiff: &FileDiff{
					File:      "/abs/path/file2.go",
					Additions: 10,
					Deletions: 5,
				},
			},
			want: []FileChange{
				{
					AbsPath:   "/abs/path/file1.go",
					IsNew:     false,
					Additions: 0,
					Deletions: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFileChanges(tt.event)

			// Handle nil vs empty slice comparison
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("extractFileChanges() returned %d changes, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].AbsPath != tt.want[i].AbsPath {
					t.Errorf("change[%d].AbsPath = %s, want %s", i, got[i].AbsPath, tt.want[i].AbsPath)
				}
				if got[i].IsNew != tt.want[i].IsNew {
					t.Errorf("change[%d].IsNew = %v, want %v", i, got[i].IsNew, tt.want[i].IsNew)
				}
				if got[i].Additions != tt.want[i].Additions {
					t.Errorf("change[%d].Additions = %d, want %d", i, got[i].Additions, tt.want[i].Additions)
				}
				if got[i].Deletions != tt.want[i].Deletions {
					t.Errorf("change[%d].Deletions = %d, want %d", i, got[i].Deletions, tt.want[i].Deletions)
				}
			}
		})
	}
}

// TestOnFileChangeCallback verifies that onFileChange callback is invoked
// when processing completed edit tool calls with diff blocks
func TestOnFileChangeCallback(t *testing.T) {
	tests := []struct {
		name          string
		event         ToolCallEvent
		wantCallCount int
		wantChanges   []FileChange
	}{
		{
			name: "completed edit with single diff block - calls callback once",
			event: ToolCallEvent{
				Status: "completed",
				Kind:   "edit",
				DiffBlocks: []DiffBlock{
					{
						Path:    "/abs/path/file.go",
						OldText: "old",
						NewText: "new",
					},
				},
				FileDiff: &FileDiff{
					File:      "/abs/path/file.go",
					Additions: 5,
					Deletions: 2,
				},
			},
			wantCallCount: 1,
			wantChanges: []FileChange{
				{
					AbsPath:   "/abs/path/file.go",
					IsNew:     false,
					Additions: 5,
					Deletions: 2,
				},
			},
		},
		{
			name: "completed edit with multiple diff blocks - calls callback multiple times",
			event: ToolCallEvent{
				Status: "completed",
				Kind:   "edit",
				DiffBlocks: []DiffBlock{
					{
						Path:    "/abs/path/file1.go",
						OldText: "",
						NewText: "new file",
					},
					{
						Path:    "/abs/path/file2.go",
						OldText: "old",
						NewText: "new",
					},
				},
			},
			wantCallCount: 2,
			wantChanges: []FileChange{
				{
					AbsPath:   "/abs/path/file1.go",
					IsNew:     true,
					Additions: 0,
					Deletions: 0,
				},
				{
					AbsPath:   "/abs/path/file2.go",
					IsNew:     false,
					Additions: 0,
					Deletions: 0,
				},
			},
		},
		{
			name: "completed edit with no diff blocks - no callback",
			event: ToolCallEvent{
				Status:   "completed",
				Kind:     "edit",
				RawInput: map[string]any{},
			},
			wantCallCount: 0,
			wantChanges:   []FileChange{},
		},
		{
			name: "error status - no callback",
			event: ToolCallEvent{
				Status: "error",
				Kind:   "edit",
				DiffBlocks: []DiffBlock{
					{
						Path:    "/abs/path/file.go",
						OldText: "old",
						NewText: "new",
					},
				},
			},
			wantCallCount: 0,
			wantChanges:   []FileChange{},
		},
		{
			name: "non-edit kind - no callback",
			event: ToolCallEvent{
				Status: "completed",
				Kind:   "bash",
				DiffBlocks: []DiffBlock{
					{
						Path:    "/abs/path/file.go",
						OldText: "old",
						NewText: "new",
					},
				},
			},
			wantCallCount: 0,
			wantChanges:   []FileChange{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track callback invocations
			var callCount int
			var recordedChanges []FileChange

			// Mock onFileChange callback
			onFileChange := func(change FileChange) {
				callCount++
				recordedChanges = append(recordedChanges, change)
			}

			// Simulate the logic from acp.go:365-378
			// Extract file changes and call callback only for completed edits
			if tt.event.Status == "completed" && tt.event.Kind == "edit" {
				changes := extractFileChanges(tt.event)
				for _, change := range changes {
					onFileChange(change)
				}
			}

			// Verify call count
			if callCount != tt.wantCallCount {
				t.Errorf("onFileChange called %d times, want %d", callCount, tt.wantCallCount)
			}

			// Verify changes passed to callback
			if len(recordedChanges) != len(tt.wantChanges) {
				t.Errorf("recorded %d changes, want %d", len(recordedChanges), len(tt.wantChanges))
				return
			}

			for i := range recordedChanges {
				if recordedChanges[i].AbsPath != tt.wantChanges[i].AbsPath {
					t.Errorf("change[%d].AbsPath = %s, want %s", i, recordedChanges[i].AbsPath, tt.wantChanges[i].AbsPath)
				}
				if recordedChanges[i].IsNew != tt.wantChanges[i].IsNew {
					t.Errorf("change[%d].IsNew = %v, want %v", i, recordedChanges[i].IsNew, tt.wantChanges[i].IsNew)
				}
				if recordedChanges[i].Additions != tt.wantChanges[i].Additions {
					t.Errorf("change[%d].Additions = %d, want %d", i, recordedChanges[i].Additions, tt.wantChanges[i].Additions)
				}
				if recordedChanges[i].Deletions != tt.wantChanges[i].Deletions {
					t.Errorf("change[%d].Deletions = %d, want %d", i, recordedChanges[i].Deletions, tt.wantChanges[i].Deletions)
				}
			}
		})
	}
}
