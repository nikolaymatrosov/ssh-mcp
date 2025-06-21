package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ssh-mcp/internal/session"
)

// Operations handles file transfer and directory operations over SSH
type Operations struct {
	sessionManager *session.Manager
}

// NewOperations creates a new file operations handler
func NewOperations(sessionManager *session.Manager) *Operations {
	return &Operations{
		sessionManager: sessionManager,
	}
}

// Upload transfers a local file to the remote server
func (o *Operations) Upload(sessionID, localPath, remotePath string) error {
	// Get the session from the manager
	sess, err := o.sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Read the local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %v", err)
	}
	defer localFile.Close()

	// Get file info for size
	fileInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Create a new SSH session
	sshSession, err := sess.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer sshSession.Close()

	// Set up the SCP command
	cmd := fmt.Sprintf("scp -t %s", remotePath)
	stdin, err := sshSession.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	// Start the SCP command
	if err := sshSession.Start(cmd); err != nil {
		return fmt.Errorf("failed to start SCP command: %v", err)
	}

	// Check for SCP acknowledgment
	buffer := make([]byte, 1)
	if _, err := stdout.Read(buffer); err != nil {
		return fmt.Errorf("failed to read SCP acknowledgment: %v", err)
	}
	if buffer[0] != 0 {
		return errors.New("SCP protocol error")
	}

	// Send file header
	filename := filepath.Base(localPath)
	header := fmt.Sprintf("C0644 %d %s\n", fileInfo.Size(), filename)
	if _, err := stdin.Write([]byte(header)); err != nil {
		return fmt.Errorf("failed to send file header: %v", err)
	}

	// Check for SCP acknowledgment
	if _, err := stdout.Read(buffer); err != nil {
		return fmt.Errorf("failed to read SCP acknowledgment: %v", err)
	}
	if buffer[0] != 0 {
		return errors.New("SCP protocol error")
	}

	// Send file content
	if _, err := io.Copy(stdin, localFile); err != nil {
		return fmt.Errorf("failed to send file content: %v", err)
	}

	// Send file transfer completion
	if _, err := stdin.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to send file transfer completion: %v", err)
	}

	// Check for SCP acknowledgment
	if _, err := stdout.Read(buffer); err != nil {
		return fmt.Errorf("failed to read SCP acknowledgment: %v", err)
	}
	if buffer[0] != 0 {
		return errors.New("SCP protocol error")
	}

	// Wait for the command to complete
	if err := sshSession.Wait(); err != nil {
		return fmt.Errorf("SCP command failed: %v", err)
	}

	return nil
}

// Download transfers a remote file to the local machine
func (o *Operations) Download(sessionID, remotePath, localPath string) error {
	// Get the session from the manager
	sess, err := o.sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Create a new SSH session
	sshSession, err := sess.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer sshSession.Close()

	// Set up the SCP command
	cmd := fmt.Sprintf("scp -f %s", remotePath)
	stdin, err := sshSession.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	// Start the SCP command
	if err := sshSession.Start(cmd); err != nil {
		return fmt.Errorf("failed to start SCP command: %v", err)
	}

	// Send null byte to initiate transfer
	if _, err := stdin.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to initiate transfer: %v", err)
	}

	// Read file header
	headerBuf := new(bytes.Buffer)
	for {
		b := make([]byte, 1)
		if _, err := stdout.Read(b); err != nil {
			return fmt.Errorf("failed to read file header: %v", err)
		}
		if b[0] == '\n' {
			break
		}
		headerBuf.Write(b)
	}

	header := headerBuf.String()
	if !strings.HasPrefix(header, "C") {
		return fmt.Errorf("invalid file header: %s", header)
	}

	// Parse file size
	parts := strings.Split(header, " ")
	if len(parts) < 3 {
		return fmt.Errorf("invalid file header format: %s", header)
	}
	// File size is available in parts[1] if needed for progress reporting

	// Send acknowledgment
	if _, err := stdin.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to send acknowledgment: %v", err)
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer localFile.Close()

	// Copy file content
	if _, err := io.Copy(localFile, stdout); err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	// Send final acknowledgment
	if _, err := stdin.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to send final acknowledgment: %v", err)
	}

	// Wait for the command to complete
	if err := sshSession.Wait(); err != nil {
		return fmt.Errorf("SCP command failed: %v", err)
	}

	return nil
}

// ListDirectory lists the contents of a remote directory
func (o *Operations) ListDirectory(sessionID, remotePath string) ([]map[string]string, error) {
	// Get the session from the manager
	sess, err := o.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	// Create a new SSH session
	sshSession, err := sess.Client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer sshSession.Close()

	// Execute ls command
	cmd := fmt.Sprintf("ls -la %s", remotePath)
	output, err := sshSession.CombinedOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %v", err)
	}

	// Parse the output
	lines := strings.Split(string(output), "\n")
	result := make([]map[string]string, 0, len(lines))

	// Skip the first line (total count) and empty lines
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "total ") {
			continue
		}

		// Parse the ls output
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		// Extract file information
		permissions := fields[0]
		size := fields[4]
		date := strings.Join(fields[5:8], " ")
		name := strings.Join(fields[8:], " ")

		result = append(result, map[string]string{
			"name":        name,
			"permissions": permissions,
			"size":        size,
			"date":        date,
			"isDirectory": strconv.FormatBool(permissions[0] == 'd'),
		})
	}

	return result, nil
}
