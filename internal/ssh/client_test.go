package ssh

import (
	"strings"
	"testing"
	"time"

	"ssh-mcp/internal/session"
)

func TestGenerateSessionID(t *testing.T) {
	host := "example.com"
	username := "testuser"

	id1 := generateSessionID(host, username)

	// Ensure the ID contains the host and username
	if !contains(id1, host) {
		t.Errorf("Session ID %s does not contain host %s", id1, host)
	}

	if !contains(id1, username) {
		t.Errorf("Session ID %s does not contain username %s", id1, username)
	}

	// Ensure IDs are unique
	time.Sleep(1 * time.Nanosecond) // Ensure different timestamp
	id2 := generateSessionID(host, username)
	if id1 == id2 {
		t.Errorf("Session IDs should be unique, but got %s twice", id1)
	}
}

func TestNewClient(t *testing.T) {
	sessionManager := session.NewManager(30 * time.Minute)
	client := NewClient(sessionManager)

	if client == nil {
		t.Error("NewClient returned nil")
	}

	if client.sessionManager != sessionManager {
		t.Error("Client has incorrect session manager")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return s != "" && substr != "" && strings.Contains(s, substr)
}

// Note: More comprehensive tests would require mocking the SSH server
// or setting up an actual SSH server for integration testing.
// For this implementation, we're focusing on unit tests for the client's
// internal functionality.
