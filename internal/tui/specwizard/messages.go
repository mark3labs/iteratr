package specwizard

import (
	"github.com/mark3labs/iteratr/internal/agent"
	"github.com/mark3labs/iteratr/internal/specmcp"
)

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

// ExecBuildMsg is sent when the user clicks Start Build button.
// It triggers execution of iteratr build --spec <path> and exits the wizard.
type ExecBuildMsg struct {
	SpecPath string
}

// ShowCancelConfirmMsg is sent when ESC is pressed on the first question.
// It triggers the agent phase to show the cancel confirmation modal.
type ShowCancelConfirmMsg struct{}

// AgentPhaseReadyMsg is sent when the agent phase resources are initialized.
// Update assigns the carried fields to WizardModel and launches RunIteration.
type AgentPhaseReadyMsg struct {
	MCPServer   *specmcp.Server
	AgentStep   *AgentPhase
	AgentRunner *agent.Runner
	Prompt      string
}

// SaveErrorMsg is sent when spec save fails.
// It displays an error modal with retry/cancel options.
type SaveErrorMsg struct {
	Err error
}

// RetrySaveMsg is sent when user chooses to retry after a save error.
type RetrySaveMsg struct{}
