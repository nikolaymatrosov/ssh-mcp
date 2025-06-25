package ssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"ssh-mcp/internal/session"

	"golang.org/x/crypto/ssh"
)

// Client handles SSH connections and operations
type Client struct {
	sessionManager *session.Manager
}

// NewClient creates a new SSH client with the given session manager
func NewClient(sessionManager *session.Manager) *Client {
	return &Client{
		sessionManager: sessionManager,
	}
}

// Connect establishes a new SSH connection and returns a session ID
func (c *Client) Connect(args SSHConnectArgs) (string, error) {
	// Create SSH client configuration
	config := &ssh.ClientConfig{
		User:            args.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Replace with proper host key verification
		Timeout:         time.Duration(args.Timeout) * time.Second,
	}

	// Set up authentication
	if args.Password != "" {
		config.Auth = []ssh.AuthMethod{
			ssh.Password(args.Password),
		}
	} else if args.KeyPath != "" {
		key, err := os.ReadFile(args.KeyPath)
		if err != nil {
			return "", fmt.Errorf("unable to read private key: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return "", fmt.Errorf("unable to parse private key: %v", err)
		}

		config.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	} else {
		return "", errors.New("no authentication method provided")
	}

	// Connect to SSH server
	port := args.Port
	if port == 0 {
		port = 22 // Default SSH port
	}

	addr := net.JoinHostPort(args.Host, strconv.Itoa(port))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to SSH server: %v", err)
	}

	// Generate a unique session ID
	sessionID := generateSessionID(args.Host, args.Username)

	// Add the session to the manager
	c.sessionManager.AddSession(sessionID, client, args.Host, args.Username)

	return sessionID, nil
}

// ExecuteCommand executes a command on the SSH server
func (c *Client) ExecuteCommand(args SSHCommandArgs) (string, error) {
	// Get the session from the manager
	sess, err := c.sessionManager.GetSession(args.SessionID)
	if err != nil {
		return "", err
	}

	// Create a new SSH session
	sshSession, err := sess.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer sshSession.Close()

	// Set up output buffers
	var stdout, stderr bytes.Buffer
	sshSession.Stdout = &stdout
	sshSession.Stderr = &stderr

	// Execute the command with timeout
	if args.Timeout <= 0 {
		// Default timeout to 30 seconds if not specified
		args.Timeout = 30
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(args.Timeout)*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- sshSession.Run(args.Command)
	}()

	// Wait for command completion or timeout
	select {
	case err := <-errCh:
		if err != nil {
			return stderr.String(), fmt.Errorf("command execution failed: %v", err)
		}
		return stdout.String(), nil
	case <-ctx.Done():
		return "", errors.New("command execution timed out")
	}
}

// Disconnect closes an SSH connection
func (c *Client) Disconnect(args SSHDisconnectArgs) error {
	return c.sessionManager.RemoveSession(args.SessionID)
}

// ListSessions returns a list of active SSH sessions
func (c *Client) ListSessions() []map[string]string {
	sessions := c.sessionManager.ListSessions()
	result := make([]map[string]string, 0, len(sessions))

	for _, sess := range sessions {
		result = append(result, map[string]string{
			"id":           sess.ID,
			"host":         sess.Host,
			"username":     sess.Username,
			"createdAt":    sess.CreatedAt.Format(time.RFC3339),
			"lastActivity": sess.LastActivity.Format(time.RFC3339),
		})
	}

	return result
}

// generateSessionID creates a unique session ID
func generateSessionID(host, username string) string {
	return fmt.Sprintf("%s-%s-%d", host, username, time.Now().UnixNano())
}
