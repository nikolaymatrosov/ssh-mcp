package file

import (
	"strings"
	"testing"

	"ssh-mcp/internal/session"
)

// TestNewOperations tests the NewOperations function
func TestNewOperations(t *testing.T) {
	sessionManager := session.NewManager(0) // Use default expiry
	ops := NewOperations(sessionManager)

	if ops == nil {
		t.Fatal("NewOperations returned nil")
	}

	// Check that the session manager was set correctly
	if ops.sessionManager != sessionManager {
		t.Error("Operations has incorrect session manager")
	}
}

// TestParseDirectoryListing tests the parsing of directory listing output
func TestParseDirectoryListing(t *testing.T) {
	// Sample output from ls -la command

	// Create a sample directory listing output
	output := `total 16
drwxr-xr-x  2 user group 4096 Jan  1 12:34 .
drwxr-xr-x 10 user group 4096 Jan  1 12:34 ..
-rw-r--r--  1 user group  123 Jan  1 12:34 file1.txt
drwxr-xr-x  3 user group 4096 Jan  1 12:34 dir1
`

	// Parse the output using similar logic to ListDirectory
	lines := strings.Split(output, "\n")
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

		isDir := "false"
		if permissions[0] == 'd' {
			isDir = "true"
		}

		result = append(result, map[string]string{
			"name":        name,
			"permissions": permissions,
			"size":        size,
			"date":        date,
			"isDirectory": isDir,
		})
	}

	// Verify the parsed results
	if len(result) != 4 {
		t.Errorf("Expected 4 entries, got %d", len(result))
	}

	// Check file1.txt details
	var file1 map[string]string
	for _, f := range result {
		if f["name"] == "file1.txt" {
			file1 = f
			break
		}
	}

	if file1 == nil {
		t.Fatal("file1.txt not found in results")
	}

	if file1["permissions"] != "-rw-r--r--" {
		t.Errorf("Expected permissions -rw-r--r--, got %s", file1["permissions"])
	}

	if file1["size"] != "123" {
		t.Errorf("Expected size 123, got %s", file1["size"])
	}

	if file1["isDirectory"] != "false" {
		t.Errorf("Expected isDirectory false, got %s", file1["isDirectory"])
	}

	// Check dir1 details
	var dir1 map[string]string
	for _, f := range result {
		if f["name"] == "dir1" {
			dir1 = f
			break
		}
	}

	if dir1 == nil {
		t.Fatal("dir1 not found in results")
	}

	if dir1["isDirectory"] != "true" {
		t.Errorf("Expected isDirectory true, got %s", dir1["isDirectory"])
	}
}
