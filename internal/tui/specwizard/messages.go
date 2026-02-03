package specwizard

import "github.com/mark3labs/iteratr/internal/specmcp"

// TitleSubmittedMsg is sent when the user submits the title.
type TitleSubmittedMsg struct {
	Title string
}

// DescriptionSubmittedMsg is sent when the user submits the description.
type DescriptionSubmittedMsg struct {
	Description string
}

// SpecContentReceivedMsg is sent when the agent finishes generating the spec.
type SpecContentReceivedMsg struct {
	Content string
}

// SpecSavedMsg is sent when the spec has been saved to disk.
type SpecSavedMsg struct {
	Path string
}

// AgentErrorMsg is sent when there's an error starting or running the agent.
type AgentErrorMsg struct {
	Err error
}

// SpecContentRequestMsg wraps a spec content request from the MCP finish-spec handler.
type SpecContentRequestMsg struct {
	Request specmcp.SpecContentRequest
}
