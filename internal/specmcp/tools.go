package specmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerTools registers the ask-questions and finish-spec tools with the MCP server.
func (s *Server) registerTools() error {
	// ask-questions: array of question objects
	s.mcpServer.AddTool(
		mcp.NewTool("ask-questions",
			mcp.WithDescription("Ask the user one or more questions and receive their answers"),
			mcp.WithArray("questions", mcp.Required(),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{
							"type":        "string",
							"description": "Full question text",
						},
						"header": map[string]any{
							"type":        "string",
							"description": "Short label (max 30 chars)",
						},
						"options": map[string]any{
							"type":        "array",
							"description": "Available answer options",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"label": map[string]any{
										"type":        "string",
										"description": "Display text (1-5 words)",
									},
									"description": map[string]any{
										"type":        "string",
										"description": "Longer description of the option",
									},
								},
								"required": []string{"label"},
							},
						},
						"multiple": map[string]any{
							"type":        "boolean",
							"description": "Allow multi-select (default: false)",
						},
					},
					"required": []string{"question", "header", "options"},
				})),
		),
		s.handleAskQuestions,
	)

	// finish-spec: finalize the spec with markdown content
	s.mcpServer.AddTool(
		mcp.NewTool("finish-spec",
			mcp.WithDescription("Finalize the feature specification with complete markdown content"),
			mcp.WithString("content", mcp.Required(),
				mcp.Description("Complete specification in markdown format"),
			),
		),
		s.handleFinishSpec,
	)

	return nil
}

// handleAskQuestions handles the ask-questions tool call from the agent.
// This handler parses questions, sends them to the UI via channel, and blocks until answers are received.
func (s *Server) handleAskQuestions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if a question request is already pending
	// This prevents duplicate questions from appearing when the agent calls ask-questions multiple times
	s.mu.Lock()
	if s.questionPending {
		s.mu.Unlock()
		return mcp.NewToolResultError("a question request is already pending - please wait for the user to answer the current questions before asking more"), nil
	}
	s.questionPending = true
	s.mu.Unlock()

	// Ensure we clear the pending flag when done (success or failure)
	defer func() {
		s.mu.Lock()
		s.questionPending = false
		s.mu.Unlock()
	}()

	// Extract arguments
	args := request.GetArguments()
	if args == nil {
		return mcp.NewToolResultError("no arguments provided"), nil
	}

	// Extract questions array
	questionsRaw, ok := args["questions"]
	if !ok {
		return mcp.NewToolResultError("missing 'questions' parameter"), nil
	}

	// Type assert to []any (mcp-go returns arrays as []any)
	questionsArray, ok := questionsRaw.([]any)
	if !ok {
		return mcp.NewToolResultError("'questions' is not an array"), nil
	}

	if len(questionsArray) == 0 {
		return mcp.NewToolResultError("at least one question is required"), nil
	}

	// Parse each question
	questions := make([]Question, 0, len(questionsArray))
	for i, qRaw := range questionsArray {
		// Convert to map[string]any
		qMap, ok := qRaw.(map[string]any)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("question %d is not an object", i)), nil
		}

		// Extract question (required)
		questionText, ok := qMap["question"].(string)
		if !ok || questionText == "" {
			return mcp.NewToolResultError(fmt.Sprintf("question %d missing or empty 'question' field", i)), nil
		}

		// Extract header (required)
		header, ok := qMap["header"].(string)
		if !ok || header == "" {
			return mcp.NewToolResultError(fmt.Sprintf("question %d missing or empty 'header' field", i)), nil
		}

		// Extract options array (required)
		optionsRaw, ok := qMap["options"]
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("question %d missing 'options' field", i)), nil
		}

		optionsArray, ok := optionsRaw.([]any)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("question %d 'options' is not an array", i)), nil
		}

		if len(optionsArray) == 0 {
			return mcp.NewToolResultError(fmt.Sprintf("question %d must have at least one option", i)), nil
		}

		// Parse options
		options := make([]Option, 0, len(optionsArray))
		for j, optRaw := range optionsArray {
			optMap, ok := optRaw.(map[string]any)
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("question %d option %d is not an object", i, j)), nil
			}

			// Extract label (required)
			label, ok := optMap["label"].(string)
			if !ok || label == "" {
				return mcp.NewToolResultError(fmt.Sprintf("question %d option %d missing or empty 'label' field", i, j)), nil
			}

			// Extract description (optional)
			description := ""
			if desc, ok := optMap["description"].(string); ok {
				description = desc
			}

			options = append(options, Option{
				Label:       label,
				Description: description,
			})
		}

		// Extract multiple flag (optional, defaults to false)
		multiple := false
		if multipleVal, ok := qMap["multiple"].(bool); ok {
			multiple = multipleVal
		}

		questions = append(questions, Question{
			Question: questionText,
			Header:   header,
			Options:  options,
			Multiple: multiple,
		})
	}

	// Send questions to UI and block for answers
	resultCh := make(chan []interface{}, 1)

	// Send request to UI via channel with context awareness
	select {
	case s.questionCh <- QuestionRequest{
		Questions: questions,
		ResultCh:  resultCh,
	}:
		// Request sent, now block waiting for UI response
	case <-ctx.Done():
		return mcp.NewToolResultError("cancelled"), nil
	}

	// Block until answers received from UI or context cancelled
	select {
	case answers := <-resultCh:
		// Return answers as JSON array (each element is string or []string)
		answersJSON, err := json.Marshal(answers)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal answers: %v", err)), nil
		}
		return mcp.NewToolResultText(string(answersJSON)), nil
	case <-ctx.Done():
		return mcp.NewToolResultError("cancelled"), nil
	}
}

// handleFinishSpec handles the finish-spec tool call.
// It validates the content parameter and sends it to the UI via the specContentCh channel,
// blocking until the UI confirms the save operation.
func (s *Server) handleFinishSpec(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments
	args := request.GetArguments()
	if args == nil {
		return mcp.NewToolResultError("no arguments provided"), nil
	}

	// Extract and validate content parameter
	content, ok := args["content"].(string)
	if !ok {
		return mcp.NewToolResultError("content parameter must be a string"), nil
	}

	if content == "" {
		return mcp.NewToolResultError("content cannot be empty"), nil
	}

	// Create response channel for this request
	resultCh := make(chan error, 1)

	// Send request to UI via channel
	req := SpecContentRequest{
		Content:  content,
		ResultCh: resultCh,
	}

	select {
	case s.specContentCh <- req:
		// Request sent, now block waiting for UI response
	case <-ctx.Done():
		return mcp.NewToolResultError("request cancelled"), nil
	}

	// Block until UI confirms save
	select {
	case err := <-resultCh:
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText("Spec saved successfully"), nil
	case <-ctx.Done():
		return mcp.NewToolResultError("request cancelled"), nil
	}
}
