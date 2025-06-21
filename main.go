package main

import (
	"log"

	"ssh-mcp/internal/server"
)

func main() {
	// Get default server configuration
	config := server.DefaultConfig()

	// Create and configure the server
	mcpServer, _, err := server.SetupServer(config)
	if err != nil {
		panic(err)
	}

	// Start the server
	log.Printf("Starting SSH-MCP server on :%d...", config.Port)
	mcpServer.Serve()
}
