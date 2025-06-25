package main

import (
	"flag"
	"log"

	"ssh-mcp/internal/server"
)

func main() {
	// Parse command line flags
	var transport string
	flag.StringVar(&transport, "t", "http", "Transport type (stdio or http)")
	flag.StringVar(&transport, "transport", "http", "Transport type (stdio or http)")
	flag.Parse()

	// Get default server configuration
	config := server.DefaultConfig()

	// Create and configure the server
	mcpServer, _, err := server.SetupServer(config)
	if err != nil {
		panic(err)
	}

	// Start the server based on transport type
	log.Printf("Starting SSH-MCP server...")

	if transport == "http" {
		log.Printf("Using HTTP transport on port %d", config.Port)
		if err := server.StartHTTPServer(mcpServer, config.Port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		log.Printf("Using stdio transport")
		if err := server.StartStdioServer(mcpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
