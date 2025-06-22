package file

import (
	"bufio"
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

	// The target directory and file for talking the SCP protocol
	targetDir := filepath.Dir(remotePath)
	targetFile := filepath.Base(remotePath)

	// Convert to forward slashes for compatibility with Unix systems
	targetDir = filepath.ToSlash(targetDir)

	// Open the local file to determine size
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
	size := fileInfo.Size()

	// Define the SCP upload function
	scpFunc := func(w io.Writer, stdoutR *bufio.Reader) error {
		return scpUploadFile(targetFile, localFile, w, stdoutR, size)
	}

	// Execute the SCP command
	cmd := fmt.Sprintf("scp -vt %s", targetDir)
	return o.scpSession(sess, cmd, scpFunc)
}

// Download transfers a remote file to the local machine
func (o *Operations) Download(sessionID, remotePath, localPath string) error {
	// Get the session from the manager
	sess, err := o.sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Create the local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer localFile.Close()

	// Define the SCP download function
	scpFunc := func(w io.Writer, stdoutR *bufio.Reader) error {
		// Send null byte to initiate transfer
		if _, err := w.Write([]byte{0}); err != nil {
			return fmt.Errorf("failed to initiate transfer: %v", err)
		}

		// Read file header
		header, err := stdoutR.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read file header: %v", err)
		}

		if !strings.HasPrefix(header, "C") {
			return fmt.Errorf("invalid file header: %s", header)
		}

		// Parse file size
		parts := strings.Split(header, " ")
		if len(parts) < 3 {
			return fmt.Errorf("invalid file header format: %s", header)
		}

		// Parse file size
		fileSize, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid file size in header: %v", err)
		}

		// Send acknowledgment
		if _, err := w.Write([]byte{0}); err != nil {
			return fmt.Errorf("failed to send acknowledgment: %v", err)
		}

		// Copy file content with size limit
		if _, err := io.CopyN(localFile, stdoutR, fileSize); err != nil {
			return fmt.Errorf("failed to copy file content: %v", err)
		}

		// Read the final status byte
		statusBuf := make([]byte, 1)
		if _, err := stdoutR.Read(statusBuf); err != nil {
			return fmt.Errorf("failed to read status byte: %v", err)
		}

		if statusBuf[0] != 0 {
			message, _, err := stdoutR.ReadLine()
			if err != nil {
				return fmt.Errorf("error reading error message: %v", err)
			}
			return fmt.Errorf("SCP protocol error: %s", message)
		}

		// Send final acknowledgment
		if _, err := w.Write([]byte{0}); err != nil {
			return fmt.Errorf("failed to send final acknowledgment: %v", err)
		}

		return nil
	}

	// Execute the SCP command
	cmd := fmt.Sprintf("scp -vf %s", remotePath)
	return o.scpSession(sess, cmd, scpFunc)
}

// scpSession executes an SCP command and handles the SCP protocol
func (o *Operations) scpSession(sess *session.Session, scpCommand string, f func(io.Writer, *bufio.Reader) error) error {
	// Create a new SSH session
	sshSession, err := sess.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer sshSession.Close()

	// Get a pipe to stdin so that we can send data
	stdinW, err := sshSession.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %v", err)
	}
	defer stdinW.Close()

	// Get a pipe to stdout so that we can get responses back
	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	stdoutR := bufio.NewReader(stdoutPipe)

	// Set stderr to a bytes buffer
	stderr := new(bytes.Buffer)
	sshSession.Stderr = stderr

	// Start the SCP command
	if err := sshSession.Start(scpCommand); err != nil {
		return fmt.Errorf("failed to start SCP command: %v", err)
	}

	// Call our callback that executes in the context of SCP
	if err := f(stdinW, stdoutR); err != nil && err != io.EOF {
		return fmt.Errorf("SCP protocol error: %v", err)
	}

	// Wait for the SCP command to complete
	err = sshSession.Wait()
	if err != nil {
		// Log any stderr before returning an error
		scpErr := stderr.String()
		if len(scpErr) > 0 {
			return fmt.Errorf("SCP command failed: %v, stderr: %s", err, scpErr)
		}
		return fmt.Errorf("SCP command failed: %v", err)
	}

	return nil
}

// scpUploadFile uploads a file using the SCP protocol
func scpUploadFile(filename string, src io.Reader, w io.Writer, r *bufio.Reader, size int64) error {
	// Start the protocol
	fmt.Fprintf(w, "C0644 %d %s\n", size, filename)
	if err := checkSCPStatus(r); err != nil {
		return fmt.Errorf("failed to send file header: %v", err)
	}

	// Send file content
	if _, err := io.Copy(w, src); err != nil {
		return fmt.Errorf("failed to send file content: %v", err)
	}

	// Send file transfer completion
	if _, err := w.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to send file transfer completion: %v", err)
	}

	// Check for SCP acknowledgment
	if err := checkSCPStatus(r); err != nil {
		return fmt.Errorf("failed to get final acknowledgment: %v", err)
	}

	return nil
}

// checkSCPStatus checks that a prior command sent to SCP completed successfully
func checkSCPStatus(r *bufio.Reader) error {
	code, err := r.ReadByte()
	if err != nil {
		return err
	}

	if code != 0 {
		// Treat any non-zero as fatal errors
		message, _, err := r.ReadLine()
		if err != nil {
			return fmt.Errorf("error reading error message: %v", err)
		}
		return errors.New(string(message))
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
