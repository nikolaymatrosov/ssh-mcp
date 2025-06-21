package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// MCPClient is a client for the SSH MCP tool
type MCPClient struct {
	baseURL string
}

// NewMCPClient creates a new MCP client
func NewMCPClient(baseURL string) *MCPClient {
	return &MCPClient{
		baseURL: baseURL,
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
	resp, err := c.callTool("ssh_connect", payload)
	if err != nil {
		return "", err
	}

	// Extract the session ID from the response
	// The response format is "Connected. Session ID: <session-id>"
	sessionID := ""
	if content, ok := resp["content"].(string); ok {
		// Parse the session ID from the content
		fmt.Sscanf(content, "Connected. Session ID: %s", &sessionID)
	}

	if sessionID == "" {
		return "", fmt.Errorf("failed to get session ID from response: %v", resp)
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
	resp, err := c.callTool("ssh_execute", payload)
	if err != nil {
		return "", err
	}

	// Extract the command output from the response
	if content, ok := resp["content"].(string); ok {
		return content, nil
	}

	return "", fmt.Errorf("failed to get command output from response: %v", resp)
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
	_, err := c.callTool("ssh_upload_file", payload)
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
	_, err := c.callTool("ssh_download_file", payload)
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
	resp, err := c.callTool("ssh_list_directory", payload)
	if err != nil {
		return "", err
	}

	// Extract the directory listing from the response
	if content, ok := resp["content"].(string); ok {
		return content, nil
	}

	return "", fmt.Errorf("failed to get directory listing from response: %v", resp)
}

// SSHDisconnect disconnects from the SSH server
func (c *MCPClient) SSHDisconnect(sessionID string) error {
	// Create the request payload
	payload := map[string]interface{}{
		"sessionId": sessionID,
	}

	// Call the SSH disconnect tool
	_, err := c.callTool("ssh_disconnect", payload)
	return err
}

// callTool calls an MCP tool with the given name and payload
func (c *MCPClient) callTool(toolName string, payload map[string]interface{}) (map[string]interface{}, error) {
	// Create the request URL
	url := fmt.Sprintf("%s/mcp/tools/%s", c.baseURL, toolName)

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, body)
	}

	// Parse the response JSON
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return result, nil
}
