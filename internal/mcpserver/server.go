package mcpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/mcp-go/server"
)

// Server manages an embedded MCP HTTP server that exposes task/note/session tools.
// The server is started when a session begins and provides native MCP protocol access
// to session management instead of spawning CLI processes.
type Server struct {
	store      *session.Store
	sessName   string
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	stdServer  *http.Server // Standard HTTP server that uses the listener
	port       int
	mu         sync.Mutex
}

// New creates a new MCP server instance for the given session.
// The server is not started until Start() is called.
func New(store *session.Store, sessionName string) *Server {
	return &Server{
		store:    store,
		sessName: sessionName,
	}
}

// Start starts the MCP HTTP server on a random available port.
// Blocks until the server is ready to accept connections.
// Returns the port number or an error if startup fails.
func (s *Server) Start(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stdServer != nil {
		return 0, fmt.Errorf("server already started")
	}

	// Create MCP server with registered tools
	s.mcpServer = server.NewMCPServer(
		"iteratr-tools",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tools
	if err := s.registerTools(); err != nil {
		return 0, fmt.Errorf("failed to register tools: %w", err)
	}

	// Find a random available port by creating a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find available port: %w", err)
	}

	// Get the port that was assigned
	s.port = listener.Addr().(*net.TCPAddr).Port

	// Create HTTP server with stateless mode and pass listener directly to avoid TOCTOU race
	mux := http.NewServeMux()
	mcpHandler := server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithStateLess(true),
	)
	mux.Handle("/mcp", mcpHandler)

	s.stdServer = &http.Server{
		Handler: mux,
	}
	s.httpServer = mcpHandler

	logger.Debug("Starting MCP server on port %d", s.port)

	// Start server in background using the pre-opened listener
	// Capture stdServer reference for goroutine to avoid race with Stop()
	stdServer := s.stdServer
	go func() {
		if err := stdServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("MCP server error: %v", err)
		}
	}()

	// Server is ready immediately after Start() returns
	logger.Debug("MCP server ready on port %d", s.port)
	return s.port, nil
}

// Stop stops the MCP HTTP server and cleans up resources.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stdServer == nil {
		return nil // Already stopped
	}

	logger.Debug("Stopping MCP server")
	if err := s.stdServer.Shutdown(context.Background()); err != nil {
		logger.Warn("Error stopping MCP server: %v", err)
		return fmt.Errorf("failed to stop server: %w", err)
	}

	s.httpServer = nil
	s.stdServer = nil
	s.mcpServer = nil
	logger.Debug("MCP server stopped")
	return nil
}

// URL returns the HTTP URL for the MCP server endpoint.
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("http://localhost:%d/mcp", s.port)
}
