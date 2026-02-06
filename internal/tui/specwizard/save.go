package specwizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/mark3labs/iteratr/internal/logger"
)

const (
	specsMarker = "<!-- SPECS -->"
	tableHeader = "| Name | Description | Date |"
	tableSep    = "|------|-------------|------|"
)

// saveSpec saves the spec content to a file and updates the README.
// Returns the path to the saved spec file.
func saveSpec(specDir, title, description, content string) (string, error) {
	// Ensure spec directory exists
	if err := os.MkdirAll(specDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create spec directory: %w", err)
	}

	// Slugify title for filename
	slugTitle := slug.Make(title)
	if slugTitle == "" {
		slugTitle = "unnamed-spec"
	}
	specPath := filepath.Join(specDir, slugTitle+".md")

	// Write spec file
	logger.Debug("Writing spec to %s", specPath)
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write spec file: %w", err)
	}

	// Update README
	readmePath := filepath.Join(specDir, "README.md")
	logger.Debug("Updating README at %s", readmePath)
	if err := updateREADME(readmePath, slugTitle+".md", title, description); err != nil {
		return "", fmt.Errorf("failed to update README: %w", err)
	}

	return specPath, nil
}

// updateREADME updates the specs README.md file with a new spec entry.
// If the README doesn't exist, it creates one with a header and table.
// If the <!-- SPECS --> marker exists, it inserts the new row after the marker.
// If the marker doesn't exist, it appends the marker and table to the end.
func updateREADME(readmePath, filename, title, description string) error {
	// Prepare description (first line, max 100 runes)
	desc := firstLine(description)
	if desc == "" {
		desc = "No description provided"
	}

	// Truncate description to 100 runes (rune-safe)
	descRunes := []rune(desc)
	if len(descRunes) > 100 {
		desc = string(descRunes[:97]) + "..."
	}

	// Truncate title to 100 runes (rune-safe)
	titleRunes := []rune(title)
	if len(titleRunes) > 100 {
		title = string(titleRunes[:97]) + "..."
	}

	// Escape pipes in title and description for markdown table cells
	escapedTitle := strings.ReplaceAll(title, "|", "\\|")
	escapedDesc := strings.ReplaceAll(desc, "|", "\\|")

	// Format date as YYYY-MM-DD
	date := time.Now().Format("2006-01-02")

	// Create new table row
	newRow := fmt.Sprintf("| [%s](%s) | %s | %s |", escapedTitle, filename, escapedDesc, date)

	// Read existing README or create new one
	var content string
	existingContent, err := os.ReadFile(readmePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read README: %w", err)
		}
		// Create new README with header and table
		logger.Debug("Creating new README at %s", readmePath)
		content = createNewREADME(newRow)
	} else {
		// Update existing README
		content = string(existingContent)
		logger.Debug("Updating existing README")
		content = insertSpecEntry(content, newRow)
	}

	// Write updated README
	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write README: %w", err)
	}

	return nil
}

// createNewREADME creates a new README with header and spec table.
func createNewREADME(newRow string) string {
	return fmt.Sprintf(`# Specs

Feature specifications created with iteratr.

%s

%s
%s
%s
`, specsMarker, tableHeader, tableSep, newRow)
}

// insertSpecEntry inserts a new spec entry into the README content.
// If the marker exists, it inserts after the marker.
// If the marker doesn't exist, it appends marker + table to the end.
func insertSpecEntry(content, newRow string) string {
	lines := strings.Split(content, "\n")

	// Find marker position
	markerIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == specsMarker {
			markerIdx = i
			break
		}
	}

	if markerIdx == -1 {
		// Marker not found - append marker and table to end
		logger.Debug("Marker not found, appending to end")
		// Ensure content ends with newline
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		// Add blank line before marker if content exists
		if len(strings.TrimSpace(content)) > 0 {
			content += "\n"
		}
		content += specsMarker + "\n\n"
		content += tableHeader + "\n"
		content += tableSep + "\n"
		content += newRow + "\n"
		return content
	}

	// Marker found - insert after marker
	logger.Debug("Marker found at line %d, inserting after", markerIdx)

	// Find where to insert (after marker, skip blank lines and table header if present)
	insertIdx := markerIdx + 1

	// Skip blank lines after marker
	for insertIdx < len(lines) && strings.TrimSpace(lines[insertIdx]) == "" {
		insertIdx++
	}

	// Check if table header exists
	hasTable := false
	if insertIdx < len(lines) && strings.TrimSpace(lines[insertIdx]) == tableHeader {
		hasTable = true
		insertIdx++ // Skip header
		// Skip separator line
		if insertIdx < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[insertIdx]), "|--") {
			insertIdx++
		}
	}

	// If no table exists, insert table header first
	if !hasTable {
		// Insert blank line + table header + separator before our row
		newLines := make([]string, 0, len(lines)+4)
		newLines = append(newLines, lines[:insertIdx]...)
		newLines = append(newLines, "")
		newLines = append(newLines, tableHeader)
		newLines = append(newLines, tableSep)
		newLines = append(newLines, newRow)
		newLines = append(newLines, lines[insertIdx:]...)
		return strings.Join(newLines, "\n")
	}

	// Table exists - insert new row at current position
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, newRow)
	newLines = append(newLines, lines[insertIdx:]...)

	return strings.Join(newLines, "\n")
}

// firstLine returns the first non-empty line from a multi-line string.
func firstLine(s string) string {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
