package e2e

import (
	"context"
	"fmt"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"
)

// MCPClient is a client for the SSH MCP tool
type MCPClient struct {
	baseURL string
	client  *mcp_golang.Client
}

// NewMCPClient creates a new MCP client
func NewMCPClient(ctx context.Context, baseURL string) *MCPClient {
	// The baseURL is expected to include "/mcp", but we need to remove it
	// for the transport since it will add it back
	baseURLWithoutMCP := strings.TrimSuffix(baseURL, "/mcp")

	// Create an HTTP transport that connects to the server
	transport := http.NewHTTPClientTransport("/mcp")
	transport.WithBaseURL(baseURLWithoutMCP)

	// Create a new client with the transport
	client := mcp_golang.NewClient(transport)
	_, err := client.Initialize(ctx)
	if err != nil {
		panic(err)
	}

	return &MCPClient{
		baseURL: baseURL,
		client:  client,
	}
}

// SSHConnect connects to an SSH server
func (c *MCPClient) SSHConnect(host string, port int, username, password string) (string, error) {
	// Create the request payload
	payload := map[string]interface{}{
		"host":     host,
		"port":     port,
		"username": username,
		"password": password,
		"timeout":  10,
	}

	// Call the SSH connect tool
	response, err := c.client.CallTool(context.Background(), "ssh_connect", payload)
	if err != nil {
		return "", err
	}

	// Extract the session ID from the response
	// The response format is "Connected. Session ID: <session-id>"
	sessionID := ""
	if len(response.Content) > 0 && response.Content[0].TextContent != nil {
		content := response.Content[0].TextContent.Text
		// Parse the session ID from the content
		parts := strings.Split(content, "Session ID: ")
		if len(parts) > 1 {
			sessionID = strings.TrimSpace(parts[1])
		}
	}

	if sessionID == "" {
		return "", fmt.Errorf("failed to get session ID from response")
	}

	return sessionID, nil
}

// SSHExecuteCommand executes a command on the SSH server
func (c *MCPClient) SSHExecuteCommand(sessionID, command string) (string, error) {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId": sessionID,
		"command":   command,
		"timeout":   30,
	}

	// Call the SSH execute tool
	response, err := c.client.CallTool(context.Background(), "ssh_execute", payload)
	if err != nil {
		return "", err
	}

	// Extract the command output from the response
	if len(response.Content) > 0 && response.Content[0].TextContent != nil {
		return response.Content[0].TextContent.Text, nil
	}

	return "", fmt.Errorf("failed to get command output from response")
}

// SSHUploadFile uploads a file to the SSH server
func (c *MCPClient) SSHUploadFile(sessionID, localPath, remotePath string) error {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId":   sessionID,
		"source":      localPath,
		"destination": remotePath,
		"direction":   "upload",
	}

	// Call the SSH upload file tool
	_, err := c.client.CallTool(context.Background(), "ssh_upload_file", payload)
	return err
}

// SSHDownloadFile downloads a file from the SSH server
func (c *MCPClient) SSHDownloadFile(sessionID, remotePath, localPath string) error {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId":   sessionID,
		"source":      remotePath,
		"destination": localPath,
		"direction":   "download",
	}

	// Call the SSH download file tool
	_, err := c.client.CallTool(context.Background(), "ssh_download_file", payload)
	return err
}

// SSHListDirectory lists the contents of a directory on the SSH server
func (c *MCPClient) SSHListDirectory(sessionID, path string) (string, error) {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId": sessionID,
		"path":      path,
	}

	// Call the SSH list directory tool
	response, err := c.client.CallTool(context.Background(), "ssh_list_directory", payload)
	if err != nil {
		return "", err
	}

	// Extract the directory listing from the response
	if len(response.Content) > 0 && response.Content[0].TextContent != nil {
		return response.Content[0].TextContent.Text, nil
	}

	return "", fmt.Errorf("failed to get directory listing from response")
}

// SSHUploadDir uploads a directory to the SSH server
func (c *MCPClient) SSHUploadDir(sessionID, localDir, remoteDir string) error {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId":   sessionID,
		"source":      localDir,
		"destination": remoteDir,
	}

	// Call the SSH upload directory tool
	_, err := c.client.CallTool(context.Background(), "ssh_upload_directory", payload)
	return err
}

// SSHDownloadDir downloads a directory from the SSH server
func (c *MCPClient) SSHDownloadDir(sessionID, remoteDir, localDir string) error {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId":   sessionID,
		"source":      remoteDir,
		"destination": localDir,
		"direction":   "download",
		"isDirectory": true,
	}

	// Call the SSH download directory tool
	_, err := c.client.CallTool(context.Background(), "ssh_download_directory", payload)
	return err
}

// SSHDisconnect disconnects from the SSH server
func (c *MCPClient) SSHDisconnect(sessionID string) error {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId": sessionID,
	}

	// Call the SSH disconnect tool
	_, err := c.client.CallTool(context.Background(), "ssh_disconnect", payload)
	return err
}
