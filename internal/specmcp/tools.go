package specmcp

import (
	"context"

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
								"required": []string{"label", "description"},
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

// handleAskQuestions handles the ask-questions tool call.
// Implementation will be added in a future task.
func (s *Server) handleAskQuestions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Implement in TAS-16
	return mcp.NewToolResultError("ask-questions handler not yet implemented"), nil
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
