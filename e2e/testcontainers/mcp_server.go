package testcontainers

import (
	"context"
	"fmt"
	"log"
	"time"

	"ssh-mcp/internal/server"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// MCPServer represents an SSH MCP server
type MCPServer struct {
	server     *mcpserver.MCPServer
	httpServer *mcpserver.StreamableHTTPServer
	stopChan   chan struct{}
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

	// Create HTTP server
	httpServer := mcpserver.NewStreamableHTTPServer(mcpServer)

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting SSH-MCP server on :%d...", port)

		// Signal that the server is ready
		time.Sleep(100 * time.Millisecond)

		// Start the server
		addr := fmt.Sprintf(":%d", port)
		err := httpServer.Start(addr)
		if err != nil {
			log.Printf("Error starting server: %v", err)
		}

		// The server will run until the stopChan is closed
		<-stopChan
	}()

	// Wait for the server to be ready
	time.Sleep(200 * time.Millisecond)

	return &MCPServer{
		server:     mcpServer,
		httpServer: httpServer,
		stopChan:   stopChan,
	}, nil
}

// Stop stops the SSH MCP server
func (s *MCPServer) Stop() {
	// Close the stop channel to signal the goroutine to exit
	close(s.stopChan)

	// Give the server a moment to shut down
	time.Sleep(100 * time.Millisecond)
}
