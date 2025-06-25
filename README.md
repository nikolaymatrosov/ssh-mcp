# SSH MCP Tool

SSH MCP is a tool that provides SSH functionality through an MCP (Machine Communication Protocol) interface. It allows you to connect to SSH servers, execute commands, transfer files, and manage SSH sessions.

## Features

- SSH connection management (connect/disconnect)
- Authentication methods (password, key-based)
- Command execution with timeout handling
- File transfer (upload/download)
- Directory listing
- Session management
- Security features (host allowlist/denylist, command filtering, rate limiting)

## Project Structure

- `main.go`: The main application entry point
- `internal/`: Internal packages
  - `ssh/`: SSH client functionality
  - `session/`: Session management
  - `file/`: File operations
  - `security/`: Security features
  - `server/`: MCP server implementation
- `e2e/`: End-to-end tests

## Development and Testing

This project uses testcontainers for end-to-end testing with a real SSH server.

### End-to-End Testing

The `e2e` directory contains end-to-end tests that verify the SSH MCP tool works correctly with a real SSH server. These tests use testcontainers to automatically create and manage an SSH server container for testing.

For more details, see [e2e/README.md](e2e/README.md).

## Getting Started

### Prerequisites

- Docker
- Go 1.24 or later

### Running the Application

1. Clone the repository
2. Start the SSH MCP server:

```bash
# Start with HTTP transport (default)
go run main.go

# Or specify the transport explicitly
go run main.go -t http

# Start with stdio transport
go run main.go -t stdio
```

When using HTTP transport, the server will start on port 8081.

### Running the Tests

```bash
# Unit tests
go test ./internal/...

# End-to-end tests
cd e2e
go test -v
```

The end-to-end tests will automatically start the necessary containers and servers for testing.

## Usage

The SSH MCP tool provides the following tools:

- `ssh_connect`: Establish an SSH connection
- `ssh_execute`: Execute a command over SSH
- `ssh_disconnect`: Close an SSH connection
- `ssh_list_sessions`: List active SSH sessions
- `ssh_upload_file`: Upload a file to the SSH server
- `ssh_download_file`: Download a file from the SSH server
- `ssh_list_directory`: List contents of a directory on the SSH server
- `ssh_upload_directory`: Upload a directory to the SSH server
- `ssh_download_directory`: Download a directory from the SSH server

These tools can be accessed through the MCP interface at `http://localhost:8081/mcp` (HTTP transport) or via standard input/output (stdio transport).
