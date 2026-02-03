package specmcp

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/mcp-go/server"
)

// Server manages the spec-specific MCP server that exposes ask-questions and finish-spec tools.
// This server is separate from the iteratr-tools MCP server used in build sessions.
type Server struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	port       int
	mu         sync.Mutex

	// Wizard state (passed during initialization)
	specTitle string // Title from wizard step 1
	specDir   string // From config

	// Channels for UI communication (per-request response channel pattern)
	questionCh    chan QuestionRequest
	specContentCh chan SpecContentRequest
}

// QuestionRequest represents a batch of questions from the agent.
// The MCP handler blocks on ResultCh until the UI sends answers.
type QuestionRequest struct {
	Questions []Question
	ResultCh  chan []interface{} // Handler blocks on this; UI sends answers here
}

// SpecContentRequest represents the final spec content from finish-spec tool.
// The MCP handler blocks on ResultCh until the UI confirms save.
type SpecContentRequest struct {
	Content  string
	ResultCh chan error // Handler blocks on this; UI sends nil on save success
}

// Question represents a single question with options.
type Question struct {
	Question string   // Full question text
	Header   string   // Short label (max 30 chars)
	Options  []Option // Available options
	Multiple bool     // Allow multi-select
}

// Option represents a single answer option.
type Option struct {
	Label       string // Display text (1-5 words)
	Description string // Longer description
}

// New creates a new spec MCP server instance.
// The server is not started until Start() is called.
func New(specTitle, specDir string) *Server {
	return &Server{
		specTitle:     specTitle,
		specDir:       specDir,
		questionCh:    make(chan QuestionRequest, 1),
		specContentCh: make(chan SpecContentRequest, 1),
	}
}

// Start starts the MCP HTTP server on a random available port.
// Returns the port number or an error if startup fails.
func (s *Server) Start(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer != nil {
		return 0, fmt.Errorf("server already started")
	}

	// Create MCP server
	s.mcpServer = server.NewMCPServer(
		"iteratr-spec",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tools
	if err := s.registerTools(); err != nil {
		return 0, fmt.Errorf("failed to register tools: %w", err)
	}

	// Find a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find available port: %w", err)
	}

	// Get the port that was assigned
	s.port = listener.Addr().(*net.TCPAddr).Port
	// Close the listener - we just needed it to find a free port
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("failed to close listener: %w", err)
	}

	// Create HTTP server with stateless mode
	s.httpServer = server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithStateLess(true),
	)

	logger.Debug("Starting spec MCP server on port %d", s.port)

	// Start server in background
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	httpServer := s.httpServer
	go func() {
		if err := httpServer.Start(addr); err != nil {
			logger.Error("Spec MCP server error: %v", err)
		}
	}()

	logger.Debug("Spec MCP server ready on port %d", s.port)
	return s.port, nil
}

// Stop stops the MCP HTTP server and cleans up resources.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer == nil {
		return nil // Already stopped
	}

	logger.Debug("Stopping spec MCP server")
	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		logger.Warn("Error stopping spec MCP server: %v", err)
		return fmt.Errorf("failed to stop server: %w", err)
	}

	s.httpServer = nil
	s.mcpServer = nil
	logger.Debug("Spec MCP server stopped")
	return nil
}

// URL returns the HTTP URL for the MCP server endpoint.
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("http://localhost:%d/mcp", s.port)
}

// QuestionChan returns channel for receiving question requests from MCP handlers.
func (s *Server) QuestionChan() <-chan QuestionRequest {
	return s.questionCh
}

// SpecContentChan returns channel for receiving spec content from finish-spec handler.
func (s *Server) SpecContentChan() <-chan SpecContentRequest {
	return s.specContentCh
}
