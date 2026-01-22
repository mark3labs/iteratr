package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/spf13/cobra"
)

var toolFlags struct {
	name    string
	dataDir string
}

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Execute session tools (used by opencode plugins)",
	Long: `Execute session management tools for interacting with a running iteratr session.
These commands are typically called by opencode plugins, not directly by users.`,
}

func init() {
	// Add tool subcommand to root
	rootCmd.AddCommand(toolCmd)

	// Add subcommands for each tool operation
	toolCmd.AddCommand(taskAddCmd)
	toolCmd.AddCommand(taskStatusCmd)
	toolCmd.AddCommand(taskPriorityCmd)
	toolCmd.AddCommand(taskDependsCmd)
	toolCmd.AddCommand(taskListCmd)
	toolCmd.AddCommand(taskNextCmd)
	toolCmd.AddCommand(noteAddCmd)
	toolCmd.AddCommand(noteListCmd)
	toolCmd.AddCommand(inboxListCmd)
	toolCmd.AddCommand(inboxMarkReadCmd)
	toolCmd.AddCommand(iterationSummaryCmd)
	toolCmd.AddCommand(sessionCompleteCmd)

	// Common flags for all tool subcommands
	toolCmd.PersistentFlags().StringVarP(&toolFlags.name, "name", "n", "", "Session name (required)")
	toolCmd.PersistentFlags().StringVar(&toolFlags.dataDir, "data-dir", "", "Data directory (default: from ITERATR_DATA_DIR or .iteratr)")
}

// connectToSession connects to a running iteratr session's NATS server
func connectToSession() (*session.Store, func(), error) {
	// Determine data directory
	dataDir := toolFlags.dataDir
	if dataDir == "" {
		dataDir = os.Getenv("ITERATR_DATA_DIR")
	}
	if dataDir == "" {
		dataDir = ".iteratr"
	}

	// Read port from port file
	natsDataDir := dataDir + "/nats"
	port, err := nats.ReadPort(natsDataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to session (is iteratr build running?): %w", err)
	}

	// Connect to NATS
	nc, err := nats.ConnectToPort(port)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Get stream
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := js.Stream(ctx, nats.StreamName)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("failed to get stream: %w", err)
	}

	// Create store
	store := session.NewStore(js, stream)

	// Return cleanup function
	cleanup := func() {
		nc.Close()
	}

	return store, cleanup, nil
}

// task-add command
var taskAddCmd = &cobra.Command{
	Use:   "task-add",
	Short: "Add a new task",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		content, _ := cmd.Flags().GetString("content")
		status, _ := cmd.Flags().GetString("status")

		if content == "" {
			return fmt.Errorf("content is required")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		task, err := store.TaskAdd(ctx, toolFlags.name, session.TaskAddParams{
			Content: content,
			Status:  status,
		})
		if err != nil {
			return err
		}

		// Output JSON for parsing
		output, _ := json.Marshal(map[string]string{
			"id":      task.ID,
			"status":  task.Status,
			"content": task.Content,
		})
		fmt.Println(string(output))
		return nil
	},
}

func init() {
	taskAddCmd.Flags().String("content", "", "Task content (required)")
	taskAddCmd.Flags().String("status", "remaining", "Initial status")
}

// task-status command
var taskStatusCmd = &cobra.Command{
	Use:   "task-status",
	Short: "Update task status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		id, _ := cmd.Flags().GetString("id")
		status, _ := cmd.Flags().GetString("status")

		if id == "" {
			return fmt.Errorf("task ID is required")
		}
		if status == "" {
			return fmt.Errorf("status is required")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		err = store.TaskStatus(ctx, toolFlags.name, session.TaskStatusParams{
			ID:     id,
			Status: status,
		})
		if err != nil {
			return err
		}

		fmt.Println("OK")
		return nil
	},
}

func init() {
	taskStatusCmd.Flags().String("id", "", "Task ID (required)")
	taskStatusCmd.Flags().String("status", "", "New status (required)")
}

// task-priority command
var taskPriorityCmd = &cobra.Command{
	Use:   "task-priority",
	Short: "Update task priority",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		id, _ := cmd.Flags().GetString("id")
		priority, _ := cmd.Flags().GetInt("priority")

		if id == "" {
			return fmt.Errorf("task ID is required")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		err = store.TaskPriority(ctx, toolFlags.name, session.TaskPriorityParams{
			ID:       id,
			Priority: priority,
		})
		if err != nil {
			return err
		}

		fmt.Println("OK")
		return nil
	},
}

func init() {
	taskPriorityCmd.Flags().String("id", "", "Task ID (required)")
	taskPriorityCmd.Flags().Int("priority", 2, "Priority level (0=critical, 1=high, 2=medium, 3=low, 4=backlog)")
}

// task-depends command
var taskDependsCmd = &cobra.Command{
	Use:   "task-depends",
	Short: "Add task dependency",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		id, _ := cmd.Flags().GetString("id")
		dependsOn, _ := cmd.Flags().GetString("depends-on")

		if id == "" {
			return fmt.Errorf("task ID is required")
		}
		if dependsOn == "" {
			return fmt.Errorf("depends-on is required")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		err = store.TaskDepends(ctx, toolFlags.name, session.TaskDependsParams{
			ID:        id,
			DependsOn: dependsOn,
		})
		if err != nil {
			return err
		}

		fmt.Println("OK")
		return nil
	},
}

func init() {
	taskDependsCmd.Flags().String("id", "", "Task ID (required)")
	taskDependsCmd.Flags().String("depends-on", "", "Task ID this task depends on (required)")
}

// task-list command
var taskListCmd = &cobra.Command{
	Use:   "task-list",
	Short: "List all tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		result, err := store.TaskList(ctx, toolFlags.name)
		if err != nil {
			return err
		}

		// Format output for agent consumption
		var lines []string
		formatTasks := func(status string, tasks []*session.Task) {
			if len(tasks) == 0 {
				return
			}
			lines = append(lines, fmt.Sprintf("%s:", strings.Title(strings.ReplaceAll(status, "_", " "))))
			for _, t := range tasks {
				lines = append(lines, fmt.Sprintf("  [%s] %s", t.ID[:8], t.Content))
			}
		}

		formatTasks("remaining", result.Remaining)
		formatTasks("in_progress", result.InProgress)
		formatTasks("completed", result.Completed)
		formatTasks("blocked", result.Blocked)

		if len(lines) == 0 {
			fmt.Println("No tasks")
		} else {
			fmt.Println(strings.Join(lines, "\n"))
		}
		return nil
	},
}

// note-add command
var noteAddCmd = &cobra.Command{
	Use:   "note-add",
	Short: "Add a note",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		content, _ := cmd.Flags().GetString("content")
		noteType, _ := cmd.Flags().GetString("type")

		if content == "" {
			return fmt.Errorf("content is required")
		}
		if noteType == "" {
			return fmt.Errorf("type is required")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		note, err := store.NoteAdd(ctx, toolFlags.name, session.NoteAddParams{
			Content: content,
			Type:    noteType,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Note added: [%s] %s\n", note.Type, note.ID[:8])
		return nil
	},
}

func init() {
	noteAddCmd.Flags().String("content", "", "Note content (required)")
	noteAddCmd.Flags().String("type", "", "Note type: learning, stuck, tip, decision (required)")
}

// note-list command
var noteListCmd = &cobra.Command{
	Use:   "note-list",
	Short: "List notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		noteType, _ := cmd.Flags().GetString("type")

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		notes, err := store.NoteList(ctx, toolFlags.name, session.NoteListParams{
			Type: noteType,
		})
		if err != nil {
			return err
		}

		if len(notes) == 0 {
			fmt.Println("No notes")
			return nil
		}

		for _, note := range notes {
			fmt.Printf("[%s] (#%d) %s\n", note.Type, note.Iteration, note.Content)
		}
		return nil
	},
}

func init() {
	noteListCmd.Flags().String("type", "", "Filter by type")
}

// inbox-list command
var inboxListCmd = &cobra.Command{
	Use:   "inbox-list",
	Short: "List inbox messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		messages, err := store.InboxList(ctx, toolFlags.name)
		if err != nil {
			return err
		}

		// Filter to unread only
		var unread []*session.Message
		for _, msg := range messages {
			if !msg.Read {
				unread = append(unread, msg)
			}
		}

		if len(unread) == 0 {
			fmt.Println("No unread messages")
			return nil
		}

		for _, msg := range unread {
			fmt.Printf("[%s] %s\n", msg.ID[:8], msg.Content)
		}
		return nil
	},
}

// inbox-mark-read command
var inboxMarkReadCmd = &cobra.Command{
	Use:   "inbox-mark-read",
	Short: "Mark inbox message as read",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("message ID is required")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		err = store.InboxMarkRead(ctx, toolFlags.name, session.InboxMarkReadParams{
			ID: id,
		})
		if err != nil {
			return err
		}

		fmt.Println("OK")
		return nil
	},
}

func init() {
	inboxMarkReadCmd.Flags().String("id", "", "Message ID (required)")
}

// task-next command
var taskNextCmd = &cobra.Command{
	Use:   "task-next",
	Short: "Get the next highest priority unblocked task",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		task, err := store.TaskNext(ctx, toolFlags.name)
		if err != nil {
			return err
		}

		if task == nil {
			fmt.Println("No ready tasks")
			return nil
		}

		// Output JSON for parsing
		output, _ := json.Marshal(map[string]any{
			"id":       task.ID,
			"content":  task.Content,
			"priority": task.Priority,
			"status":   task.Status,
		})
		fmt.Println(string(output))
		return nil
	},
}

// iteration-summary command
var iterationSummaryCmd = &cobra.Command{
	Use:   "iteration-summary",
	Short: "Record a summary for the current iteration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		summary, _ := cmd.Flags().GetString("summary")
		if summary == "" {
			return fmt.Errorf("summary is required")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()

		// Load state to get current iteration number
		state, err := store.LoadState(ctx, toolFlags.name)
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		// Find the current (last) iteration number
		iterNum := 1
		if len(state.Iterations) > 0 {
			iterNum = state.Iterations[len(state.Iterations)-1].Number
		}

		// Collect task IDs that are in_progress or were recently worked on
		var tasksWorked []string
		for id, task := range state.Tasks {
			if task.Status == "in_progress" || task.Iteration == iterNum {
				tasksWorked = append(tasksWorked, id)
			}
		}

		err = store.IterationSummary(ctx, toolFlags.name, iterNum, summary, tasksWorked)
		if err != nil {
			return err
		}

		fmt.Printf("Summary recorded for iteration #%d\n", iterNum)
		return nil
	},
}

func init() {
	iterationSummaryCmd.Flags().String("summary", "", "Summary of what was accomplished (required)")
}

// session-complete command
var sessionCompleteCmd = &cobra.Command{
	Use:   "session-complete",
	Short: "Mark session as complete",
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolFlags.name == "" {
			return fmt.Errorf("session name is required (--name)")
		}

		store, cleanup, err := connectToSession()
		if err != nil {
			return err
		}
		defer cleanup()

		ctx := context.Background()
		err = store.SessionComplete(ctx, toolFlags.name)
		if err != nil {
			return err
		}

		fmt.Println("Session marked complete")
		return nil
	},
}
