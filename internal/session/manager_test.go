package session

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	// Test with valid expiry
	expiry := 10 * time.Minute
	manager := NewManager(expiry)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.sessionExpiry != expiry {
		t.Errorf("Expected session expiry %v, got %v", expiry, manager.sessionExpiry)
	}

	if manager.sessions == nil {
		t.Error("Sessions map not initialized")
	}

	// Test with zero expiry (should use default)
	manager = NewManager(0)
	if manager.sessionExpiry != 30*time.Minute {
		t.Errorf("Expected default session expiry %v, got %v", 30*time.Minute, manager.sessionExpiry)
	}
}

func TestAddAndGetSession(t *testing.T) {
	manager := NewManager(10 * time.Minute)

	id := "test-session"
	host := "example.com"
	username := "testuser"

	// For testing, we can use nil as the client since we're not testing SSH functionality
	session := manager.AddSession(id, nil, host, username)

	if session == nil {
		t.Fatal("AddSession returned nil")
	}

	if session.ID != id {
		t.Errorf("Expected session ID %s, got %s", id, session.ID)
	}

	if session.Host != host {
		t.Errorf("Expected host %s, got %s", host, session.Host)
	}

	if session.Username != username {
		t.Errorf("Expected username %s, got %s", username, session.Username)
	}

	// Check if session was added to the map
	if len(manager.sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(manager.sessions))
	}

	// Check if the session in the map is the same as the returned one
	if manager.sessions[id] != session {
		t.Error("Session in map is not the same as the returned one")
	}

	// Test GetSession
	retrievedSession, err := manager.GetSession(id)
	if err != nil {
		t.Errorf("GetSession returned error: %v", err)
	}

	if retrievedSession == nil {
		t.Fatal("GetSession returned nil for existing session")
	}

	if retrievedSession.ID != id {
		t.Errorf("Expected session ID %s, got %s", id, retrievedSession.ID)
	}

	// Test getting a non-existent session
	_, err = manager.GetSession("non-existent")
	if err == nil {
		t.Error("GetSession did not return error for non-existent session")
	}
}

func TestRemoveSession(t *testing.T) {
	manager := NewManager(10 * time.Minute)

	// Add a session with nil client
	id := "test-session"
	manager.AddSession(id, nil, "example.com", "testuser")

	// Test removing an existing session
	err := manager.RemoveSession(id)
	if err != nil {
		t.Errorf("RemoveSession returned error: %v", err)
	}

	// Check if the session was removed
	if len(manager.sessions) != 0 {
		t.Errorf("Expected 0 sessions after removal, got %d", len(manager.sessions))
	}

	// Test removing a non-existent session
	err = manager.RemoveSession("non-existent")
	if err == nil {
		t.Error("RemoveSession did not return error for non-existent session")
	}
}

func TestListSessions(t *testing.T) {
	manager := NewManager(10 * time.Minute)

	// Test with no sessions
	sessions := manager.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}

	// Add some sessions
	manager.AddSession("session1", nil, "host1", "user1")
	manager.AddSession("session2", nil, "host2", "user2")

	// Test with sessions
	sessions = manager.ListSessions()
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Verify session details
	sessionIDs := make(map[string]bool)
	for _, s := range sessions {
		sessionIDs[s.ID] = true
	}

	if !sessionIDs["session1"] || !sessionIDs["session2"] {
		t.Error("ListSessions did not return all sessions")
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	manager := NewManager(100 * time.Millisecond) // Short expiry for testing

	// Add two sessions
	manager.AddSession("session1", nil, "host1", "user1")
	manager.AddSession("session2", nil, "host2", "user2")

	// Make session1 expired by manipulating its last activity time
	manager.sessions["session1"].LastActivity = time.Now().Add(-200 * time.Millisecond)

	// Run cleanup
	count := manager.CleanupExpiredSessions()

	// Check if one session was cleaned up
	if count != 1 {
		t.Errorf("Expected 1 session to be cleaned up, got %d", count)
	}

	// Check if the right session was removed
	if len(manager.sessions) != 1 {
		t.Errorf("Expected 1 session after cleanup, got %d", len(manager.sessions))
	}

	if _, exists := manager.sessions["session1"]; exists {
		t.Error("Expired session was not removed")
	}

	if _, exists := manager.sessions["session2"]; !exists {
		t.Error("Non-expired session was removed")
	}
}

func TestStartCleanupRoutine(t *testing.T) {
	// This is a simple test to ensure the function doesn't panic
	// A more comprehensive test would require waiting for the goroutine to run
	manager := NewManager(10 * time.Minute)

	// Should not panic
	manager.StartCleanupRoutine(100 * time.Millisecond)

	// Test with zero interval (should use default)
	manager.StartCleanupRoutine(0)

	// Allow some time for the goroutine to run
	time.Sleep(10 * time.Millisecond)
}
