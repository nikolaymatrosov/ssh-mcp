# End-to-End Testing for SSH MCP

This directory contains end-to-end tests for the SSH MCP tool. These tests use testcontainers to create and manage a real SSH server in a Docker container to verify that the SSH MCP tool works correctly.

## Prerequisites

- Docker
- Go 1.24 or later

## Overview

The e2e tests use the testcontainers-go library to:

1. Start an SSH server container with a predefined user and test files
2. Start the SSH MCP server programmatically
3. Run tests that connect to the SSH server through the MCP server
4. Clean up all resources when the tests are complete

## Running the Tests

To run the end-to-end tests:

```bash
cd e2e
go test -v
```

The tests will automatically:
- Start an SSH server container
- Start the MCP server
- Run the tests
- Stop and remove the container when done

No manual setup is required, as testcontainers handles all the container lifecycle management.

## Test Details

The e2e tests verify the following functionality:

1. Connecting to the SSH server
2. Executing commands on the SSH server
3. Uploading files to the SSH server
4. Downloading files from the SSH server
5. Listing directories on the SSH server
6. Disconnecting from the SSH server

## SSH Server Details

The SSH server container is configured with the following:

- Host: `ssh-server`
- Port: `22`
- Username: `testuser`
- Password: `password`

The server includes some test files and directories:
- `/home/testuser/test-file.txt`: A test file with content "This is a test file"
- `/home/testuser/test-dir`: An empty test directory

## Troubleshooting

If the tests fail, check the following:

1. Make sure both containers are running:
   ```bash
   docker ps
   ```

2. Make sure the SSH MCP server is running:
   ```bash
   ps aux | grep main.go
   ```

3. Check the SSH server logs:
   ```bash
   docker logs ssh-server
   ```

4. Try connecting to the SSH server manually:
   ```bash
   ssh testuser@ssh-server -p 22
   # Password: password
   ```
