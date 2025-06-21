package testcontainers

import (
	"context"
	"log"
	"time"

	"ssh-mcp/internal/server"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

// MCPServer represents an SSH MCP server
type MCPServer struct {
	server   *mcp_golang.Server
	stopChan chan struct{}
}

// StartMCPServer starts an SSH MCP server on the specified port
func StartMCPServer(ctx context.Context, port int) (*MCPServer, error) {
	// Create server configuration
	config := server.DefaultConfig()
	config.Port = port

	// Create and configure the server
	mcpServer, _, err := server.SetupServer(config)
	if err != nil {
		return nil, err
	}

	// Create a channel to signal when to stop the server
	stopChan := make(chan struct{})

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting SSH-MCP server on :%d...", port)

		// Signal that the server is ready
		time.Sleep(100 * time.Millisecond)

		// Start the server
		mcpServer.Serve()

		// The server will run until the stopChan is closed
		<-stopChan
	}()

	// Wait for the server to be ready
	time.Sleep(200 * time.Millisecond)

	return &MCPServer{
		server:   mcpServer,
		stopChan: stopChan,
	}, nil
}

// Stop stops the SSH MCP server
func (s *MCPServer) Stop() {
	close(s.stopChan)
}
