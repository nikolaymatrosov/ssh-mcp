package server

import (
	"fmt"
	"time"

	mcp_golang "github.com/metoro-io/mcp-golang"
	mcp_http "github.com/metoro-io/mcp-golang/transport/http"

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
func SetupServer(config Config) (*mcp_golang.Server, *session.Manager, error) {
	// Create an HTTP transport that listens on /mcp endpoint
	transport := mcp_http.NewHTTPTransport("/mcp").WithAddr(fmt.Sprintf(":%d", config.Port))

	// Create a new server with the transport
	server := mcp_golang.NewServer(
		transport,
		mcp_golang.WithName("ssh-mcp"),
		mcp_golang.WithInstructions("SSH client with MCP interface"),
		mcp_golang.WithVersion("0.1.0"),
	)

	// Initialize components
	sessionManager := session.NewManager(config.SessionExpiry)
	sessionManager.StartCleanupRoutine(config.CleanupInterval)

	securityManager := security.NewManager(security.Config{
		LoggingEnabled: config.LoggingEnabled,
		RateLimit:      config.RateLimit,
	})
	securityManager.StartCleanupRoutine(config.CleanupInterval, config.SessionExpiry)

	// Get all tools
	tools := GetTools(sessionManager, securityManager)

	// Register all tools
	for _, tool := range tools {
		err := server.RegisterTool(tool.Name, tool.Description, tool.Handler)
		if err != nil {
			return nil, nil, err
		}
	}

	return server, sessionManager, nil
}
