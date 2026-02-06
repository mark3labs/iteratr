package specwizard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFirstLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line",
			input:    "This is a single line",
			expected: "This is a single line",
		},
		{
			name:     "multiple lines",
			input:    "First line\nSecond line\nThird line",
			expected: "First line",
		},
		{
			name:     "empty lines at start",
			input:    "\n\nFirst non-empty\nSecond",
			expected: "First non-empty",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \n\t\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstLine(tt.input)
			if result != tt.expected {
				t.Errorf("firstLine() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCreateNewREADME(t *testing.T) {
	newRow := "| [Test Spec](test-spec.md) | A test spec | 2026-01-15 |"
	result := createNewREADME(newRow)

	// Verify structure
	if !strings.Contains(result, "# Specs") {
		t.Error("Expected README to contain '# Specs' header")
	}
	if !strings.Contains(result, specsMarker) {
		t.Errorf("Expected README to contain marker %q", specsMarker)
	}
	if !strings.Contains(result, tableHeader) {
		t.Error("Expected README to contain table header")
	}
	if !strings.Contains(result, tableSep) {
		t.Error("Expected README to contain table separator")
	}
	if !strings.Contains(result, newRow) {
		t.Error("Expected README to contain new row")
	}

	// Verify order (marker before table)
	markerIdx := strings.Index(result, specsMarker)
	headerIdx := strings.Index(result, tableHeader)
	rowIdx := strings.Index(result, newRow)

	if markerIdx == -1 || headerIdx == -1 || rowIdx == -1 {
		t.Fatal("Missing required components in README")
	}
	if markerIdx >= headerIdx {
		t.Error("Marker should come before table header")
	}
	if headerIdx >= rowIdx {
		t.Error("Table header should come before new row")
	}
}

func TestInsertSpecEntry_NoMarker(t *testing.T) {
	// README without marker
	existingContent := `# My Specs

Some introduction text here.

## Getting Started

Instructions...
`
	newRow := "| [New Spec](new-spec.md) | Description | 2026-01-15 |"

	result := insertSpecEntry(existingContent, newRow)

	// Verify marker and table were appended
	if !strings.Contains(result, specsMarker) {
		t.Error("Expected marker to be added")
	}
	if !strings.Contains(result, tableHeader) {
		t.Error("Expected table header to be added")
	}
	if !strings.Contains(result, newRow) {
		t.Error("Expected new row to be added")
	}

	// Verify order
	lines := strings.Split(result, "\n")
	foundMarker := false
	foundHeader := false
	foundRow := false
	for _, line := range lines {
		if strings.TrimSpace(line) == specsMarker {
			foundMarker = true
		} else if foundMarker && strings.TrimSpace(line) == tableHeader {
			foundHeader = true
		} else if foundHeader && strings.TrimSpace(line) == newRow {
			foundRow = true
			break
		}
	}

	if !foundMarker || !foundHeader || !foundRow {
		t.Error("Marker, header, and row not in correct order")
	}
}

func TestInsertSpecEntry_WithMarkerNoTable(t *testing.T) {
	// README with marker but no table yet
	existingContent := `# Specs

<!-- SPECS -->

Some other content here.
`
	newRow := "| [New Spec](new-spec.md) | Description | 2026-01-15 |"

	result := insertSpecEntry(existingContent, newRow)

	// Verify table was inserted after marker
	if !strings.Contains(result, tableHeader) {
		t.Error("Expected table header to be added")
	}
	if !strings.Contains(result, newRow) {
		t.Error("Expected new row to be added")
	}

	// Verify order: marker -> header -> separator -> row
	lines := strings.Split(result, "\n")
	var markerIdx, headerIdx, sepIdx, rowIdx = -1, -1, -1, -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == specsMarker && markerIdx == -1 {
			markerIdx = i
		} else if trimmed == tableHeader && headerIdx == -1 {
			headerIdx = i
		} else if strings.HasPrefix(trimmed, "|--") && sepIdx == -1 {
			sepIdx = i
		} else if trimmed == newRow && rowIdx == -1 {
			rowIdx = i
		}
	}

	if markerIdx == -1 || headerIdx == -1 || sepIdx == -1 || rowIdx == -1 {
		t.Fatalf("Missing components: marker=%d header=%d sep=%d row=%d", markerIdx, headerIdx, sepIdx, rowIdx)
	}
	if markerIdx >= headerIdx {
		t.Error("Marker should come before header")
	}
	if headerIdx >= sepIdx {
		t.Error("Header should come before separator")
	}
	if sepIdx >= rowIdx {
		t.Error("Separator should come before row")
	}
}

func TestInsertSpecEntry_WithTableExisting(t *testing.T) {
	// README with marker and existing table
	existingContent := `# Specs

<!-- SPECS -->

| Name | Description | Date |
|------|-------------|------|
| [Old Spec](old-spec.md) | Old description | 2026-01-01 |

More content below.
`
	newRow := "| [New Spec](new-spec.md) | New description | 2026-01-15 |"

	result := insertSpecEntry(existingContent, newRow)

	// Verify new row was inserted after separator
	if !strings.Contains(result, newRow) {
		t.Error("Expected new row to be added")
	}

	// Verify order: marker -> header -> separator -> new row -> old row
	lines := strings.Split(result, "\n")
	var markerIdx, headerIdx, newRowIdx, oldRowIdx = -1, -1, -1, -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == specsMarker && markerIdx == -1 {
			markerIdx = i
		} else if trimmed == tableHeader && headerIdx == -1 {
			headerIdx = i
		} else if trimmed == newRow && newRowIdx == -1 {
			newRowIdx = i
		} else if strings.Contains(trimmed, "Old Spec") && oldRowIdx == -1 {
			oldRowIdx = i
		}
	}

	if newRowIdx == -1 || oldRowIdx == -1 {
		t.Fatalf("Missing rows: new=%d old=%d", newRowIdx, oldRowIdx)
	}
	if newRowIdx >= oldRowIdx {
		t.Error("New row should come before old row (inserted at top)")
	}

	// Verify old row still exists
	if !strings.Contains(result, "Old Spec") {
		t.Error("Old spec entry should be preserved")
	}
}

func TestUpdateREADME_NewFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	readmePath := filepath.Join(tmpDir, "README.md")

	// Update README (file doesn't exist)
	err := updateREADME(readmePath, "test-spec.md", "Test Spec", "This is a test description")
	if err != nil {
		t.Fatalf("updateREADME() failed: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read created README: %v", err)
	}

	contentStr := string(content)

	// Verify structure
	if !strings.Contains(contentStr, "# Specs") {
		t.Error("Expected README header")
	}
	if !strings.Contains(contentStr, specsMarker) {
		t.Error("Expected marker")
	}
	if !strings.Contains(contentStr, "Test Spec") {
		t.Error("Expected spec title in README")
	}
	if !strings.Contains(contentStr, "test-spec.md") {
		t.Error("Expected spec filename in README")
	}
	if !strings.Contains(contentStr, "This is a test description") {
		t.Error("Expected description in README")
	}

	// Verify date is present (current date in YYYY-MM-DD format)
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(contentStr, today) {
		t.Errorf("Expected today's date %s in README", today)
	}
}

func TestUpdateREADME_ExistingFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	readmePath := filepath.Join(tmpDir, "README.md")

	// Create initial README with marker
	initialContent := `# Specs

<!-- SPECS -->

| Name | Description | Date |
|------|-------------|------|
| [First Spec](first-spec.md) | First description | 2026-01-01 |
`
	err := os.WriteFile(readmePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial README: %v", err)
	}

	// Update README with second spec
	err = updateREADME(readmePath, "second-spec.md", "Second Spec", "Second description")
	if err != nil {
		t.Fatalf("updateREADME() failed: %v", err)
	}

	// Read updated content
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read updated README: %v", err)
	}

	contentStr := string(content)

	// Verify both specs are present
	if !strings.Contains(contentStr, "First Spec") {
		t.Error("Expected first spec to be preserved")
	}
	if !strings.Contains(contentStr, "Second Spec") {
		t.Error("Expected second spec to be added")
	}

	// Verify second spec comes first (inserted at top)
	firstIdx := strings.Index(contentStr, "First Spec")
	secondIdx := strings.Index(contentStr, "Second Spec")
	if firstIdx == -1 || secondIdx == -1 {
		t.Fatal("Missing spec entries")
	}
	if secondIdx >= firstIdx {
		t.Error("New spec should be inserted at top of table")
	}
}

func TestUpdateREADME_LongDescription(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	readmePath := filepath.Join(tmpDir, "README.md")

	// Long description (over 100 chars)
	longDesc := strings.Repeat("a", 150)

	err := updateREADME(readmePath, "test-spec.md", "Test Spec", longDesc)
	if err != nil {
		t.Fatalf("updateREADME() failed: %v", err)
	}

	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	contentStr := string(content)

	// Verify description was truncated
	if strings.Contains(contentStr, strings.Repeat("a", 101)) {
		t.Error("Description should be truncated to 100 chars")
	}
	if !strings.Contains(contentStr, "...") {
		t.Error("Truncated description should end with '...'")
	}
}

func TestUpdateREADME_MultilineDescription(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	readmePath := filepath.Join(tmpDir, "README.md")

	// Multi-line description
	multilineDesc := "First line of description\nSecond line\nThird line"

	err := updateREADME(readmePath, "test-spec.md", "Test Spec", multilineDesc)
	if err != nil {
		t.Fatalf("updateREADME() failed: %v", err)
	}

	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	contentStr := string(content)

	// Verify only first line is used
	if !strings.Contains(contentStr, "First line of description") {
		t.Error("Expected first line of description")
	}
	if strings.Contains(contentStr, "Second line") {
		t.Error("Second line should not be in README")
	}
}

func TestUpdateREADME_EmptyDescription(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	readmePath := filepath.Join(tmpDir, "README.md")

	err := updateREADME(readmePath, "test-spec.md", "Test Spec", "")
	if err != nil {
		t.Fatalf("updateREADME() failed: %v", err)
	}

	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	contentStr := string(content)

	// Verify placeholder description is used
	if !strings.Contains(contentStr, "No description provided") {
		t.Error("Expected placeholder description for empty description")
	}
}

func TestSaveSpec(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "specs")

	// Save spec
	title := "My Test Feature"
	description := "This is a test feature"
	content := `# My Test Feature

## Overview
Test overview

## Tasks
- [ ] Task 1
`
	specPath, err := saveSpec(specDir, title, description, content)
	if err != nil {
		t.Fatalf("saveSpec() failed: %v", err)
	}

	// Verify spec file was created
	expectedPath := filepath.Join(specDir, "my-test-feature.md")
	if specPath != expectedPath {
		t.Errorf("saveSpec() returned path %q, want %q", specPath, expectedPath)
	}

	specContent, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec file: %v", err)
	}

	if string(specContent) != content {
		t.Error("Spec file content doesn't match")
	}

	// Verify README was created and updated
	readmePath := filepath.Join(specDir, "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	readmeStr := string(readmeContent)
	if !strings.Contains(readmeStr, "My Test Feature") {
		t.Error("README should contain spec title")
	}
	if !strings.Contains(readmeStr, "my-test-feature.md") {
		t.Error("README should contain spec filename")
	}
	if !strings.Contains(readmeStr, "This is a test feature") {
		t.Error("README should contain spec description")
	}
}

func TestSaveSpec_SpecialCharactersInTitle(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "specs")

	// Title with special characters
	title := "User Auth & API Integration (v2.0)"
	description := "Test description"
	content := "# Test content"

	specPath, err := saveSpec(specDir, title, description, content)
	if err != nil {
		t.Fatalf("saveSpec() failed: %v", err)
	}

	// Verify slug is created correctly (should be lowercase, no special chars)
	if !strings.Contains(specPath, "user-auth-api-integration-v2-0.md") &&
		!strings.Contains(specPath, "user-auth-and-api-integration-v2-0.md") {
		t.Errorf("Unexpected slugified path: %s", specPath)
	}

	// Verify file exists
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("Spec file was not created")
	}
}

func TestUpdateREADME_PipesInTitleAndDescription(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	readmePath := filepath.Join(tmpDir, "README.md")

	// Title and description with pipes
	title := "Feature | Component A"
	description := "Implements A | B | C functionality"

	err := updateREADME(readmePath, "test-spec.md", title, description)
	if err != nil {
		t.Fatalf("updateREADME() failed: %v", err)
	}

	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	contentStr := string(content)

	// Verify pipes are escaped in the table
	if !strings.Contains(contentStr, "Feature \\| Component A") {
		t.Error("Pipes in title should be escaped")
	}
	if !strings.Contains(contentStr, "Implements A \\| B \\| C functionality") {
		t.Error("Pipes in description should be escaped")
	}

	// Verify table is still valid (count pipes per row)
	lines := strings.Split(contentStr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "| [Feature") {
			// Count unescaped pipes: should be exactly 4 (| title | desc | date |)
			unescapedPipes := 0
			escaped := false
			for i, ch := range trimmed {
				if ch == '\\' && i+1 < len(trimmed) && trimmed[i+1] == '|' {
					escaped = true
				} else if ch == '|' && !escaped {
					unescapedPipes++
				} else {
					escaped = false
				}
			}
			if unescapedPipes != 4 {
				t.Errorf("Table row should have exactly 4 unescaped pipes, got %d: %s", unescapedPipes, trimmed)
			}
		}
	}
}

func TestUpdateREADME_LongTitleWithUnicode(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	readmePath := filepath.Join(tmpDir, "README.md")

	// Long title with unicode (over 100 runes)
	longTitle := "一二三四五六七八九十" + strings.Repeat("a", 95) // 105 runes total
	description := "Test description"

	err := updateREADME(readmePath, "test-spec.md", longTitle, description)
	if err != nil {
		t.Fatalf("updateREADME() failed: %v", err)
	}

	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	contentStr := string(content)

	// Find the table row with our spec
	lines := strings.Split(contentStr, "\n")
	var titleInTable string
	for _, line := range lines {
		if strings.Contains(line, "test-spec.md") {
			// Extract title from [title](link) format
			start := strings.Index(line, "[")
			end := strings.Index(line, "]")
			if start != -1 && end != -1 && end > start {
				titleInTable = line[start+1 : end]
			}
			break
		}
	}

	// Verify title was truncated to 100 runes + "..."
	titleRunes := []rune(titleInTable)
	if len(titleRunes) > 103 { // 100 chars + "..."
		t.Errorf("Title should be truncated to 100 runes + '...', got %d runes", len(titleRunes))
	}
	if !strings.HasSuffix(titleInTable, "...") {
		t.Error("Truncated title should end with '...'")
	}
}
