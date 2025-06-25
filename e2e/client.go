package e2e

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPClient is a client for the SSH MCP tool
type MCPClient struct {
	baseURL string
	client  *client.Client
}

// NewMCPClient creates a new MCP client
func NewMCPClient(ctx context.Context, baseURL string) *MCPClient {
	// Create an HTTP transport that connects to the server
	httpTransport, err := transport.NewStreamableHTTP(baseURL)
	if err != nil {
		panic(fmt.Errorf("failed to create HTTP transport: %v", err))
	}

	// Create a new client with the transport
	c := client.NewClient(httpTransport)

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "SSH-MCP Client",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		panic(fmt.Errorf("failed to initialize client: %v", err))
	}

	return &MCPClient{
		baseURL: baseURL,
		client:  c,
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

	// Create a CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = "ssh_connect"
	request.Params.Arguments = payload

	// Call the SSH connect tool
	ctx := context.Background()
	response, err := c.client.CallTool(ctx, request)
	if err != nil {
		return "", err
	}

	// Extract the session ID from the response
	// The response format is "Connected. Session ID: <session-id>"
	sessionID := ""
	if len(response.Content) > 0 {
		content, ok := response.Content[0].(mcp.TextContent)
		if ok {
			// Parse the session ID from the content
			parts := strings.Split(content.Text, "Session ID: ")
			if len(parts) > 1 {
				sessionID = strings.TrimSpace(parts[1])
			}
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

	// Create a CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = "ssh_execute"
	request.Params.Arguments = payload

	// Call the SSH execute tool
	ctx := context.Background()
	response, err := c.client.CallTool(ctx, request)
	if err != nil {
		return "", err
	}

	// Extract the command output from the response
	if len(response.Content) > 0 {
		content, ok := response.Content[0].(mcp.TextContent)
		if ok {
			return content.Text, nil
		}
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

	// Create a CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = "ssh_upload_file"
	request.Params.Arguments = payload

	// Call the SSH upload file tool
	ctx := context.Background()
	_, err := c.client.CallTool(ctx, request)
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

	// Create a CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = "ssh_download_file"
	request.Params.Arguments = payload

	// Call the SSH download file tool
	ctx := context.Background()
	_, err := c.client.CallTool(ctx, request)
	return err
}

// SSHListDirectory lists the contents of a directory on the SSH server
func (c *MCPClient) SSHListDirectory(sessionID, path string) (string, error) {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId": sessionID,
		"path":      path,
	}

	// Create a CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = "ssh_list_directory"
	request.Params.Arguments = payload

	// Call the SSH list directory tool
	ctx := context.Background()
	response, err := c.client.CallTool(ctx, request)
	if err != nil {
		return "", err
	}

	// Extract the directory listing from the response
	if len(response.Content) > 0 {
		content, ok := response.Content[0].(mcp.TextContent)
		if ok {
			return content.Text, nil
		}
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

	// Create a CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = "ssh_upload_directory"
	request.Params.Arguments = payload

	// Call the SSH upload directory tool
	ctx := context.Background()
	_, err := c.client.CallTool(ctx, request)
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

	// Create a CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = "ssh_download_directory"
	request.Params.Arguments = payload

	// Call the SSH download directory tool
	ctx := context.Background()
	_, err := c.client.CallTool(ctx, request)
	return err
}

// SSHDisconnect disconnects from the SSH server
func (c *MCPClient) SSHDisconnect(sessionID string) error {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId": sessionID,
	}

	// Create a CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = "ssh_disconnect"
	request.Params.Arguments = payload

	// Call the SSH disconnect tool
	ctx := context.Background()
	_, err := c.client.CallTool(ctx, request)
	return err
}
