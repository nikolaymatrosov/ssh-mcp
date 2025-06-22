package file

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
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

// UploadDir uploads a local directory to the remote server
func (o *Operations) UploadDir(sessionID, localDir, remoteDir string) error {
	// Get the session from the manager
	sess, err := o.sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Convert remote path to forward slashes for compatibility with Unix systems
	remoteDir = filepath.ToSlash(remoteDir)

	// Define the SCP upload directory function
	scpFunc := func(w io.Writer, r *bufio.Reader) error {
		// Read initial status byte from server
		code, err := r.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read status: %v", err)
		}
		if code != 0 {
			message, _, err := r.ReadLine()
			if err != nil {
				return fmt.Errorf("error reading error message: %v", err)
			}
			return errors.New(string(message))
		}

		// Open the source directory
		f, err := os.Open(localDir)
		if err != nil {
			return err
		}
		defer f.Close()

		// Read all entries in the directory
		entries, err := f.Readdir(-1)
		if err != nil {
			return err
		}

		// Upload the directory
		uploadEntries := func() error {
			return scpUploadDirEntries(localDir, entries, w, r)
		}

		if localDir[len(localDir)-1] != '/' {
			// No trailing slash, so include the directory name
			log.Printf("[DEBUG] SCP: starting directory upload: %s", filepath.Base(localDir))

			// Use Fprintln with proper spacing exactly as in the example
			fmt.Fprintln(w, "D0755 0", filepath.Base(localDir))
			if err := checkSCPStatus(r); err != nil {
				return err
			}

			if err := uploadEntries(); err != nil {
				return err
			}

			fmt.Fprintln(w, "E")
		} else {
			// Trailing slash, just upload the contents
			if err := uploadEntries(); err != nil {
				return err
			}
		}

		return nil
	}

	// Execute the SCP command
	cmd := fmt.Sprintf("scp -vrt %s", remoteDir)
	return o.scpSession(sess, cmd, scpFunc)
}

// DownloadDir downloads a remote directory to the local machine.
func (o *Operations) DownloadDir(sessionID, remotePath, localPath string) error {
	sess, err := o.sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Ensure local path exists and is a directory.
	fi, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(localPath, 0755); err != nil {
				return fmt.Errorf("failed to create local directory: %v", err)
			}
		} else {
			return err
		}
	} else if !fi.IsDir() {
		return fmt.Errorf("local path %s is not a directory", localPath)
	}

	scpFunc := func(w io.Writer, r *bufio.Reader) error {
		// Signal that we're ready for the protocol to start
		if _, err := w.Write([]byte{0}); err != nil {
			return err
		}

		return scpDownloadDir(localPath, w, r, true)
	}

	cmd := fmt.Sprintf("scp -rf %s", remotePath)
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

	// We only want to close once, so we nil stdinW after we close it,
	// and only close in the defer if it hasn't been closed already.
	defer func() {
		if stdinW != nil {
			stdinW.Close()
		}
	}()

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
	err = f(stdinW, stdoutR)

	// Close the stdin, which sends an EOF, and then set stdinW to nil so that
	// our defer func doesn't close it again since that is unsafe with
	// the Go SSH package.
	stdinW.Close()
	stdinW = nil

	// If we got an error (not EOF which is normal), return it
	if err != nil && err != io.EOF {
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
	// If size is 0, we need to create a temporary file to determine the actual size
	if size == 0 {
		// Create a temporary file where we can copy the contents of the src
		// so that we can determine the length, since SCP is length-prefixed.
		tf, err := os.CreateTemp("", "ssh-mcp-upload")
		if err != nil {
			return fmt.Errorf("error creating temporary file for upload: %v", err)
		}
		defer os.Remove(tf.Name())
		defer tf.Close()

		// Copy the data to the temporary file
		if _, err := io.Copy(tf, src); err != nil {
			return fmt.Errorf("error copying data to temporary file: %v", err)
		}

		// Sync the file so that the contents are definitely on disk
		if err := tf.Sync(); err != nil {
			return fmt.Errorf("error syncing temporary file: %v", err)
		}

		// Seek the file to the beginning so we can re-read all of it
		if _, err := tf.Seek(0, 0); err != nil {
			return fmt.Errorf("error seeking temporary file: %v", err)
		}

		// Get the file size
		fi, err := tf.Stat()
		if err != nil {
			return fmt.Errorf("error getting temporary file info: %v", err)
		}

		// Update the source and size
		src = tf
		size = fi.Size()
	}

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

	// Flush any buffered data
	if flusher, ok := w.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			return fmt.Errorf("failed to flush data: %v", err)
		}
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

// scpDownloadDir recursively downloads a directory.
func scpDownloadDir(destPath string, w io.Writer, r *bufio.Reader, stripName bool) error {
	for {
		header, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil // Clean exit
			}
			return err
		}

		// The protocol can end with an empty line from the server.
		if header == "" {
			return nil
		}

		switch header[0] {
		case 'E':
			// End of directory marker
			return ackSCP(w)
		case 'T':
			// Timestamp, which we ignore.
			// Acknowledge it and continue.
			if err := ackSCP(w); err != nil {
				return err
			}
			continue
		case 'C':
			// File transfer
			parts := strings.SplitN(header, " ", 3)
			if len(parts) != 3 {
				return fmt.Errorf("invalid file header: %q", header)
			}

			size, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid file size in header: %v", err)
			}

			name := strings.TrimRight(parts[2], "\n")

			// Acknowledge header
			if err := ackSCP(w); err != nil {
				return err
			}

			// Create file
			filePath := filepath.Join(destPath, name)
			file, err := os.Create(filePath)
			if err != nil {
				return err
			}

			// Copy contents
			_, err = io.CopyN(file, r, size)
			if err != nil {
				return err
			}
			err = file.Close()
			if err != nil {
				return fmt.Errorf("failed to close file %s: %v", filePath, err)
			}

			// Check status byte
			if err := checkSCPStatus(r); err != nil {
				return err
			}

			// Acknowledge file transfer
			if err := ackSCP(w); err != nil {
				return err
			}
		case 'D':
			// Directory transfer
			parts := strings.SplitN(header, " ", 3)
			if len(parts) != 3 {
				return fmt.Errorf("invalid directory header: %q", header)
			}

			name := strings.TrimRight(parts[2], "\n")

			// Acknowledge header
			if err := ackSCP(w); err != nil {
				return err
			}

			// Create directory
			dirPath := destPath
			if !stripName {
				dirPath = filepath.Join(destPath, name)
			}
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return err
			}

			// Recursively download directory contents
			if err := scpDownloadDir(dirPath, w, r, false); err != nil {
				return err
			}
		case 1, 2: // Warning or error message
			// The message is the line itself. We can just ignore it.
			continue
		default:
			return fmt.Errorf("unsupported scp command: %q", header)
		}
	}
}

func ackSCP(w io.Writer) error {
	// Acknowledge the SCP command by sending a null byte
	if _, err := w.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to acknowledge SCP command: %v", err)
	}
	return nil
}

// scpUploadDirProtocol initiates a directory upload in the SCP protocol
func scpUploadDirProtocol(dirName string, w io.Writer, r *bufio.Reader, f func() error) error {
	log.Printf("[DEBUG] SCP: starting directory upload: %s", dirName)
	fmt.Fprintln(w, "D0755 0", dirName)
	err := checkSCPStatus(r)
	if err != nil {
		return err
	}

	if err := f(); err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, "E")
	if err != nil {
		return err
	}

	return nil
}

// scpUploadDirEntries uploads the entries of a directory using SCP protocol
func scpUploadDirEntries(root string, entries []os.FileInfo, w io.Writer, r *bufio.Reader) error {
	for _, entry := range entries {
		localPath := filepath.Join(root, entry.Name())

		// Check if this is a symlink to a directory
		isSymlinkToDir := false
		if entry.Mode()&os.ModeSymlink == os.ModeSymlink {
			// Resolve the symlink
			symPath, err := filepath.EvalSymlinks(localPath)
			if err != nil {
				return err
			}

			// Check if it points to a directory
			symInfo, err := os.Lstat(symPath)
			if err != nil {
				return err
			}

			isSymlinkToDir = symInfo.IsDir()
		}

		if !entry.IsDir() && !isSymlinkToDir {
			// It's a regular file or symlink to a file
			file, err := os.Open(localPath)
			if err != nil {
				return err
			}

			err = func() error {
				defer file.Close()
				return scpUploadFile(entry.Name(), file, w, r, entry.Size())
			}()

			if err != nil {
				return err
			}

			continue
		}

		// It's a directory or symlink to directory, upload recursively
		err := scpUploadDirProtocol(entry.Name(), w, r, func() error {
			// Open the directory
			f, err := os.Open(localPath)
			if err != nil {
				return err
			}
			defer f.Close()

			// Read the directory entries
			subEntries, err := f.Readdir(-1)
			if err != nil {
				return err
			}

			// Upload the entries
			return scpUploadDirEntries(localPath, subEntries, w, r)
		})

		if err != nil {
			return err
		}
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
