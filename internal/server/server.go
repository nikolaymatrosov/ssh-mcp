package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"ssh-mcp/internal/security"
	"ssh-mcp/internal/session"
)

// Config holds configuration for the MCP server
type Config struct {
	Port            int
	SessionExpiry   time.Duration
	CleanupInterval time.Duration
	RateLimit       time.Duration
	LoggingEnabled  bool
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Port:            8081,
		SessionExpiry:   30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
		//RateLimit:       time.Second * 1,
		LoggingEnabled: true,
	}
}

// SetupServer creates and configures an MCP server with SSH tools
func SetupServer(config Config) (*server.MCPServer, *session.Manager, error) {
	// Initialize components
	sessionManager := session.NewManager(config.SessionExpiry)
	sessionManager.StartCleanupRoutine(config.CleanupInterval)

	securityManager := security.NewManager(security.Config{
		LoggingEnabled: config.LoggingEnabled,
		//RateLimit:      config.RateLimit,
	})
	securityManager.StartCleanupRoutine(config.CleanupInterval, config.SessionExpiry)

	// Create hooks for logging and security
	hooks := &server.Hooks{}

	if config.LoggingEnabled {
		hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
			fmt.Printf("Request: %s, %v, %v\n", method, id, message)
		})
		hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
			fmt.Printf("Success: %s, %v, %v, %v\n", method, id, message, result)
		})
		hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
			fmt.Printf("Error: %s, %v, %v, %v\n", method, id, message, err)
		})
	}

	// Create a new server
	mcpServer := server.NewMCPServer(
		"ssh-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithHooks(hooks),
	)

	// Get all tools
	tools := GetTools(sessionManager, securityManager)

	// Register all tools
	for _, tool := range tools {
		mcpTool := mcp.NewTool(
			tool.Name,
			tool.Opts...,
		)

		mcpServer.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Convert the handler to the new format
			handler, ok := tool.Handler.(func(args map[string]interface{}) (*mcp.CallToolResult, error))
			if !ok {
				return nil, fmt.Errorf("invalid handler for tool %s", tool.Name)
			}

			return handler(request.GetArguments())
		})
	}

	return mcpServer, sessionManager, nil
}

// StartHTTPServer starts the MCP server with HTTP transport
func StartHTTPServer(mcpServer *server.MCPServer, port int) error {
	httpServer := server.NewStreamableHTTPServer(mcpServer)
	addr := fmt.Sprintf(":%d", port)
	log.Printf("HTTP server listening on %s/mcp", addr)
	return httpServer.Start(addr)
}

// StartStdioServer starts the MCP server with stdio transport
func StartStdioServer(mcpServer *server.MCPServer) error {
	log.Printf("Starting stdio server")
	return server.ServeStdio(mcpServer)
}
