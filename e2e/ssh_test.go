package e2e

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"ssh-mcp/e2e/testcontainers"
)

const (
	// SSH server credentials
	sshUser     = "testuser"
	sshPassword = "password"

	// Remote paths (on SSH server)
	remoteUserHome = "/home/testuser"

	// Relative file names (will be used with temp dir)
	testLocalFileName    = "test-local-file.txt"
	testRemoteFileName   = "test-file.txt"
	testUploadFileName   = "uploaded-file.txt"
	testDownloadFileName = "downloaded-file.txt"
	testDirName          = "test-dir"
)

// TestSSHConnection tests connecting to the SSH server
func TestSSHConnection(t *testing.T) {
	// Start the SSH server container
	ctx := context.Background()
	sshContainer, err := testcontainers.StartSSHContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start SSH server container: %v", err)
	}
	defer sshContainer.Stop(ctx)

	// Start the MCP server
	mcpPort := 8081
	mcpServer, err := testcontainers.StartMCPServer(ctx, mcpPort)
	if err != nil {
		t.Fatalf("Failed to start MCP server: %v", err)
	}
	defer mcpServer.Stop()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "ssh-mcp-e2e")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to the temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Create a test file for upload
	testContent := "This is a test file for upload"
	testLocalFilePath := path.Join(tempDir, testLocalFileName)
	err = os.WriteFile(testLocalFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Construct remote paths
	testRemotePath := path.Join(remoteUserHome, testRemoteFileName)
	testUploadPath := path.Join(remoteUserHome, testUploadFileName)
	testDownloadPath := path.Join(tempDir, testDownloadFileName)
	testRemoteDir := path.Join(remoteUserHome, testDirName)

	// Test SSH connection
	sessionID, err := connectToSSH(t, sshContainer.Host, sshContainer.Port)
	if err != nil {
		t.Fatalf("Failed to connect to SSH server: %v", err)
	}

	// Test command execution
	err = executeCommand(t, sessionID, "ls -la")
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	// Test file upload
	err = uploadFile(t, sessionID, testLocalFilePath, testUploadPath)
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	// Verify upload by executing a command to check file content
	err = verifyFileContent(t, sessionID, testUploadPath, testContent)
	if err != nil {
		t.Fatalf("Failed to verify uploaded file: %v", err)
	}

	// Test file download
	err = downloadFile(t, sessionID, testRemotePath, testDownloadPath)
	if err != nil {
		t.Fatalf("Failed to download file: %v", err)
	}

	// Verify downloaded file
	downloadedContent, err := os.ReadFile(testDownloadPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if !strings.Contains(string(downloadedContent), "This is a test file") {
		t.Fatalf("Downloaded file content doesn't match expected: %s", downloadedContent)
	}

	// Test directory listing
	err = listDirectory(t, sessionID, testRemoteDir)
	if err != nil {
		t.Fatalf("Failed to list directory: %v", err)
	}

	// Test disconnection
	err = disconnectSSH(t, sessionID)
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}
}

// Helper functions to interact with the SSH MCP tool

// getClient returns a new MCP client
func getClient(ctx context.Context) *MCPClient {
	// The baseURL should point to the SSH MCP server
	// When running in the dev container, the app service can access the SSH MCP server at http://localhost:8081
	return NewMCPClient(ctx, "http://localhost:8081/mcp")
}

func connectToSSH(t *testing.T, host string, port int) (string, error) {
	t.Logf("Connecting to SSH server at %s:%d...", host, port)
	client := getClient(t.Context())
	return client.SSHConnect(host, port, sshUser, sshPassword)
}

func executeCommand(t *testing.T, sessionID, command string) error {
	t.Logf("Executing command: %s", command)
	client := getClient(t.Context())
	output, err := client.SSHExecuteCommand(sessionID, command)
	if err != nil {
		return err
	}
	t.Logf("Command output: %s", output)
	return nil
}

func uploadFile(t *testing.T, sessionID, localPath, remotePath string) error {
	t.Logf("Uploading file from %s to %s", localPath, remotePath)
	client := getClient(t.Context())
	return client.SSHUploadFile(sessionID, localPath, remotePath)
}

func downloadFile(t *testing.T, sessionID, remotePath, localPath string) error {
	t.Logf("Downloading file from %s to %s", remotePath, localPath)
	client := getClient(t.Context())
	return client.SSHDownloadFile(sessionID, remotePath, localPath)
}

func listDirectory(t *testing.T, sessionID, path string) error {
	t.Logf("Listing directory: %s", path)
	client := getClient(t.Context())
	listing, err := client.SSHListDirectory(sessionID, path)
	if err != nil {
		return err
	}
	t.Logf("Directory listing: %s", listing)
	return nil
}

func verifyFileContent(t *testing.T, sessionID, remotePath, expectedContent string) error {
	t.Logf("Verifying content of file: %s", remotePath)
	client := getClient(t.Context())
	// Use cat command to get file content
	output, err := client.SSHExecuteCommand(sessionID, "cat "+remotePath)
	if err != nil {
		return err
	}

	if !strings.Contains(output, expectedContent) {
		return fmt.Errorf("file content doesn't match expected: got %s, want %s", output, expectedContent)
	}
	return nil
}

func disconnectSSH(t *testing.T, sessionID string) error {
	t.Logf("Disconnecting session: %s", sessionID)
	client := getClient(t.Context())
	return client.SSHDisconnect(sessionID)
}
