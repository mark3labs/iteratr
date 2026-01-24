package template

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/session"
)

// Variables holds the data to be injected into template placeholders.
type Variables struct {
	Session   string // Session name
	Iteration string // Current iteration number
	Spec      string // Spec file content
	Notes     string // Formatted notes from previous iterations
	Tasks     string // Formatted task list
	History   string // Formatted iteration history
	Extra     string // Extra instructions
	Port      string // NATS server port
	Binary    string // Full path to iteratr binary
}

// Render replaces {{variable}} placeholders in template with actual values.
// Supports the following variables:
// - {{session}} - Session name
// - {{iteration}} - Current iteration number
// - {{spec}} - Spec file content
// - {{notes}} - Formatted notes (empty if none)
// - {{tasks}} - Formatted task list
// - {{history}} - Formatted iteration history
// - {{extra}} - Extra instructions (empty if none)
// - {{port}} - NATS server port
// - {{binary}} - Full path to iteratr binary
func Render(template string, vars Variables) string {
	result := template

	replacements := map[string]string{
		"{{session}}":   vars.Session,
		"{{iteration}}": vars.Iteration,
		"{{spec}}":      vars.Spec,
		"{{notes}}":     vars.Notes,
		"{{tasks}}":     vars.Tasks,
		"{{history}}":   vars.History,
		"{{extra}}":     vars.Extra,
		"{{port}}":      vars.Port,
		"{{binary}}":    vars.Binary,
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

// BuildConfig holds configuration for building a prompt.
type BuildConfig struct {
	SessionName       string         // Name of the session
	Store             *session.Store // Session store for loading state
	IterationNumber   int            // Current iteration number
	SpecPath          string         // Path to spec file
	TemplatePath      string         // Path to custom template (optional)
	ExtraInstructions string         // Extra instructions (optional)
	NATSPort          int            // NATS server port
}

// BuildPrompt loads session state, formats it, and injects it into the template.
// This is the main function for creating prompts with current state injection.
func BuildPrompt(ctx context.Context, cfg BuildConfig) (string, error) {
	logger.Debug("Building prompt for session: %s, iteration: %d", cfg.SessionName, cfg.IterationNumber)

	// Load session state
	state, err := cfg.Store.LoadState(ctx, cfg.SessionName)
	if err != nil {
		logger.Error("Failed to load session state: %v", err)
		return "", fmt.Errorf("failed to load session state: %w", err)
	}

	// Load spec file content
	specContent := ""
	if cfg.SpecPath != "" {
		logger.Debug("Loading spec file: %s", cfg.SpecPath)
		data, err := os.ReadFile(cfg.SpecPath)
		if err != nil {
			logger.Error("Failed to read spec file: %v", err)
			return "", fmt.Errorf("failed to read spec file: %w", err)
		}
		specContent = string(data)
		logger.Debug("Spec file loaded: %d bytes", len(specContent))
	}

	// Get the full path to the running binary
	binaryPath, err := os.Executable()
	if err != nil {
		logger.Warn("Failed to get executable path, using 'iteratr': %v", err)
		binaryPath = "iteratr"
	}
	logger.Debug("Binary path: %s", binaryPath)

	// Load template
	if cfg.TemplatePath != "" {
		logger.Debug("Using custom template: %s", cfg.TemplatePath)
	} else {
		logger.Debug("Using default embedded template")
	}
	templateContent, err := GetTemplate(cfg.TemplatePath)
	if err != nil {
		logger.Error("Failed to get template: %v", err)
		return "", fmt.Errorf("failed to get template: %w", err)
	}

	// Format state data
	vars := Variables{
		Session:   cfg.SessionName,
		Iteration: strconv.Itoa(cfg.IterationNumber),
		Spec:      specContent,
		Notes:     formatNotes(state),
		Tasks:     formatTasks(state),
		History:   formatIterationHistory(state),
		Extra:     cfg.ExtraInstructions,
		Port:      strconv.Itoa(cfg.NATSPort),
		Binary:    binaryPath,
	}

	logger.Debug("Formatted state: %d tasks, %d notes",
		len(state.Tasks), len(state.Notes))

	// Render template with variables
	result := Render(templateContent, vars)
	logger.Debug("Prompt rendered: %d characters", len(result))
	return result, nil
}

// formatNotes formats notes grouped by type for template injection.
// Returns empty string if no notes (section header will be omitted).
func formatNotes(state *session.State) string {
	if len(state.Notes) == 0 {
		return ""
	}

	// Group notes by type
	byType := make(map[string][]*session.Note)
	for _, note := range state.Notes {
		byType[note.Type] = append(byType[note.Type], note)
	}

	var sb strings.Builder
	sb.WriteString("## Notes\n")
	types := []string{"learning", "stuck", "tip", "decision"}
	for _, noteType := range types {
		notes := byType[noteType]
		if len(notes) == 0 {
			continue
		}

		// Uppercase first letter for display
		displayType := strings.ToUpper(noteType[:1]) + noteType[1:]
		sb.WriteString(fmt.Sprintf("%s:\n", displayType))
		for _, note := range notes {
			sb.WriteString(fmt.Sprintf("  - [#%d] %s\n", note.Iteration, note.Content))
		}
	}
	return sb.String()
}

// formatTasks formats tasks grouped by status for template injection.
// Always includes section header since workflow requires checking tasks.
func formatTasks(state *session.State) string {
	if len(state.Tasks) == 0 {
		return "## Current Tasks\nNo tasks yet - sync tasks from spec before starting work."
	}

	// Group tasks by status
	byStatus := make(map[string][]*session.Task)
	for _, task := range state.Tasks {
		byStatus[task.Status] = append(byStatus[task.Status], task)
	}

	var sb strings.Builder
	sb.WriteString("## Current Tasks\n")
	statuses := []string{"remaining", "in_progress", "completed", "blocked"}
	for _, status := range statuses {
		tasks := byStatus[status]
		if len(tasks) == 0 {
			continue
		}

		// Uppercase first letter for display
		displayStatus := strings.ToUpper(status[:1]) + strings.ReplaceAll(status[1:], "_", " ")
		sb.WriteString(fmt.Sprintf("%s:\n", displayStatus))
		for _, task := range tasks {
			// Format priority prefix [P0]-[P4]
			priorityPrefix := fmt.Sprintf("[P%d] ", task.Priority)

			// Format iteration info
			iterInfo := ""
			if task.Iteration > 0 {
				iterInfo = fmt.Sprintf(" [iteration #%d]", task.Iteration)
			}

			// Format dependency info
			depInfo := ""
			if len(task.DependsOn) > 0 {
				depIDs := make([]string, len(task.DependsOn))
				copy(depIDs, task.DependsOn)
				depInfo = fmt.Sprintf(" (depends on: %s)", strings.Join(depIDs, ", "))
			}

			sb.WriteString(fmt.Sprintf("  - %s[%s] %s%s%s\n", priorityPrefix, task.ID, task.Content, iterInfo, depInfo))
		}
	}

	return sb.String()
}

// formatIterationHistory formats recent iteration summaries for template injection.
// Shows the last 5 completed iterations with their summaries and tasks worked.
// Returns empty string if no history (section header will be omitted).
func formatIterationHistory(state *session.State) string {
	if len(state.Iterations) == 0 {
		return ""
	}

	// Filter to iterations with summaries
	withSummaries := []*session.Iteration{}
	for _, iter := range state.Iterations {
		if iter.Summary != "" {
			withSummaries = append(withSummaries, iter)
		}
	}

	if len(withSummaries) == 0 {
		return ""
	}

	// Take the last 5 iterations (most recent)
	start := 0
	if len(withSummaries) > 5 {
		start = len(withSummaries) - 5
	}
	recent := withSummaries[start:]

	var sb strings.Builder
	sb.WriteString("## Recent Progress\n")
	for _, iter := range recent {
		// Calculate time ago
		elapsed := time.Since(iter.EndedAt)
		timeAgo := formatTimeAgo(elapsed)

		// Format: "- #N (time ago): Summary"
		sb.WriteString(fmt.Sprintf("- #%d (%s): %s\n", iter.Number, timeAgo, iter.Summary))
	}

	return sb.String()
}

// formatTimeAgo formats a duration into a human-readable "time ago" string.
func formatTimeAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1min ago"
		}
		return fmt.Sprintf("%dmin ago", mins)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1hr ago"
		}
		return fmt.Sprintf("%dhr ago", hours)
	} else {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// countReadyTasks returns the number of tasks that are ready to work on.
// A task is ready if its status is "remaining" and all of its dependencies are completed.
func countReadyTasks(state *session.State) int {
	count := 0
	for _, task := range state.Tasks {
		if task.Status != "remaining" {
			continue
		}

		// Check if all dependencies are resolved (completed)
		allDepsCompleted := true
		for _, depID := range task.DependsOn {
			if depTask, exists := state.Tasks[depID]; exists {
				if depTask.Status != "completed" {
					allDepsCompleted = false
					break
				}
			} else {
				// Dependency doesn't exist - treat as unresolved
				allDepsCompleted = false
				break
			}
		}

		if allDepsCompleted {
			count++
		}
	}
	return count
}

// countBlockedTasks returns the number of tasks that are blocked.
// A task is blocked if it has status "blocked" OR if it has status "remaining" with unresolved dependencies.
func countBlockedTasks(state *session.State) int {
	count := 0
	for _, task := range state.Tasks {
		// Tasks explicitly marked as blocked
		if task.Status == "blocked" {
			count++
			continue
		}

		// Tasks that are remaining but have unresolved dependencies
		if task.Status == "remaining" && len(task.DependsOn) > 0 {
			hasUnresolvedDeps := false
			for _, depID := range task.DependsOn {
				if depTask, exists := state.Tasks[depID]; exists {
					if depTask.Status != "completed" {
						hasUnresolvedDeps = true
						break
					}
				} else {
					// Dependency doesn't exist - treat as unresolved
					hasUnresolvedDeps = true
					break
				}
			}
			if hasUnresolvedDeps {
				count++
			}
		}
	}
	return count
}
