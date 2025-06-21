package session

import (
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Session represents an active SSH session
type Session struct {
	ID           string
	Client       *ssh.Client
	CreatedAt    time.Time
	LastActivity time.Time
	Host         string
	Username     string
}

// Manager handles SSH session tracking and lifecycle
type Manager struct {
	sessions      map[string]*Session
	mu            sync.RWMutex
	sessionExpiry time.Duration
}

// NewManager creates a new session manager with the given session expiry duration
func NewManager(sessionExpiry time.Duration) *Manager {
	if sessionExpiry <= 0 {
		sessionExpiry = 30 * time.Minute // Default expiry time
	}
	
	return &Manager{
		sessions:      make(map[string]*Session),
		sessionExpiry: sessionExpiry,
	}
}

// AddSession adds a new SSH session to the manager
func (m *Manager) AddSession(id string, client *ssh.Client, host, username string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	session := &Session{
		ID:           id,
		Client:       client,
		CreatedAt:    now,
		LastActivity: now,
		Host:         host,
		Username:     username,
	}
	
	m.sessions[id] = session
	return session
}

// GetSession retrieves a session by ID and updates its last activity time
func (m *Manager) GetSession(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	session, exists := m.sessions[id]
	if !exists {
		return nil, errors.New("session not found")
	}
	
	session.LastActivity = time.Now()
	return session, nil
}

// RemoveSession removes a session from the manager
func (m *Manager) RemoveSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	session, exists := m.sessions[id]
	if !exists {
		return errors.New("session not found")
	}
	
	// Close the SSH client connection
	if session.Client != nil {
		session.Client.Close()
	}
	
	delete(m.sessions, id)
	return nil
}

// ListSessions returns a list of all active sessions
func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	
	return sessions
}

// CleanupExpiredSessions removes sessions that have been inactive for longer than the expiry duration
func (m *Manager) CleanupExpiredSessions() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	expiredCount := 0
	
	for id, session := range m.sessions {
		if now.Sub(session.LastActivity) > m.sessionExpiry {
			// Close the SSH client connection
			if session.Client != nil {
				session.Client.Close()
			}
			
			delete(m.sessions, id)
			expiredCount++
		}
	}
	
	return expiredCount
}

// StartCleanupRoutine starts a background goroutine that periodically cleans up expired sessions
func (m *Manager) StartCleanupRoutine(interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute // Default cleanup interval
	}
	
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for range ticker.C {
			m.CleanupExpiredSessions()
		}
	}()
}