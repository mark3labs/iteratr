package template

import (
	"fmt"
	"os"
	"strings"
)

// Variables holds the data to be injected into template placeholders.
type Variables struct {
	Session   string // Session name
	Iteration string // Current iteration number
	Spec      string // Spec file content
	Inbox     string // Formatted inbox messages
	Notes     string // Formatted notes from previous iterations
	Tasks     string // Formatted task list
	Extra     string // Extra instructions
}

// Render replaces {{variable}} placeholders in template with actual values.
// Supports the following variables:
// - {{session}} - Session name
// - {{iteration}} - Current iteration number
// - {{spec}} - Spec file content
// - {{inbox}} - Formatted inbox messages (empty if none)
// - {{notes}} - Formatted notes (empty if none)
// - {{tasks}} - Formatted task list
// - {{extra}} - Extra instructions (empty if none)
func Render(template string, vars Variables) string {
	result := template

	replacements := map[string]string{
		"{{session}}":   vars.Session,
		"{{iteration}}": vars.Iteration,
		"{{spec}}":      vars.Spec,
		"{{inbox}}":     vars.Inbox,
		"{{notes}}":     vars.Notes,
		"{{tasks}}":     vars.Tasks,
		"{{extra}}":     vars.Extra,
	}

	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// LoadFromFile loads a template from a file.
// If the file doesn't exist or can't be read, returns an error.
func LoadFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", path, err)
	}
	return string(data), nil
}

// GetTemplate returns the template content.
// If customPath is non-empty, loads from that file.
// Otherwise returns the default embedded template.
func GetTemplate(customPath string) (string, error) {
	if customPath == "" {
		return DefaultTemplate, nil
	}
	return LoadFromFile(customPath)
}
