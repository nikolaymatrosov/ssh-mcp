package security

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// Config holds security configuration settings
type Config struct {
	AllowedHosts    []string      // List of allowed hosts (if empty, all hosts are allowed)
	DeniedHosts     []string      // List of denied hosts
	AllowedCommands []string      // List of allowed command prefixes (if empty, all commands are allowed)
	DeniedCommands  []string      // List of denied command prefixes
	RateLimit       time.Duration // Minimum time between operations (rate limiting)
	LoggingEnabled  bool          // Whether to log operations
}

// Manager handles security features for SSH operations
type Manager struct {
	config      Config
	rateLimiter map[string]time.Time // Maps session IDs to last operation time
	mu          sync.Mutex
}

// NewManager creates a new security manager with the given configuration
func NewManager(config Config) *Manager {
	return &Manager{
		config:      config,
		rateLimiter: make(map[string]time.Time),
	}
}

// CheckHost verifies if a host is allowed to connect
func (m *Manager) CheckHost(host string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove port from host if present
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	// Check denied hosts first
	for _, denied := range m.config.DeniedHosts {
		if matchHost(host, denied) {
			m.logOperation("host_denied", host, "")
			return fmt.Errorf("host %s is denied", host)
		}
	}

	// If allowed hosts is empty, all hosts are allowed
	if len(m.config.AllowedHosts) == 0 {
		return nil
	}

	// Check if host is in allowed list
	for _, allowed := range m.config.AllowedHosts {
		if matchHost(host, allowed) {
			return nil
		}
	}

	m.logOperation("host_not_allowed", host, "")
	return fmt.Errorf("host %s is not allowed", host)
}

// CheckCommand verifies if a command is allowed to execute
func (m *Manager) CheckCommand(sessionID, command string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check rate limiting
	if m.config.RateLimit > 0 {
		lastOp, exists := m.rateLimiter[sessionID]
		now := time.Now()
		if exists && now.Sub(lastOp) < m.config.RateLimit {
			m.logOperation("rate_limited", sessionID, command)
			return errors.New("rate limit exceeded, please try again later")
		}
		m.rateLimiter[sessionID] = now
	}

	// Check denied commands first
	for _, denied := range m.config.DeniedCommands {
		if strings.HasPrefix(command, denied) {
			m.logOperation("command_denied", sessionID, command)
			return fmt.Errorf("command '%s' is denied", command)
		}
	}

	// If allowed commands is empty, all commands are allowed
	if len(m.config.AllowedCommands) == 0 {
		m.logOperation("command_executed", sessionID, command)
		return nil
	}

	// Check if command is in allowed list
	for _, allowed := range m.config.AllowedCommands {
		if strings.HasPrefix(command, allowed) {
			m.logOperation("command_executed", sessionID, command)
			return nil
		}
	}

	m.logOperation("command_not_allowed", sessionID, command)
	return fmt.Errorf("command '%s' is not allowed", command)
}

// LogOperation logs an SSH operation if logging is enabled
func (m *Manager) logOperation(operation, sessionID, details string) {
	if !m.config.LoggingEnabled {
		return
	}

	log.Printf("[SSH-MCP] %s - Session: %s - Details: %s", operation, sessionID, details)
}

// CleanupRateLimiter removes old entries from the rate limiter
func (m *Manager) CleanupRateLimiter(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for sessionID, lastOp := range m.rateLimiter {
		if now.Sub(lastOp) > maxAge {
			delete(m.rateLimiter, sessionID)
		}
	}
}

// StartCleanupRoutine starts a background goroutine that periodically cleans up the rate limiter
func (m *Manager) StartCleanupRoutine(interval, maxAge time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute // Default cleanup interval
	}
	if maxAge <= 0 {
		maxAge = 30 * time.Minute // Default max age
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			m.CleanupRateLimiter(maxAge)
		}
	}()
}

// matchHost checks if a host matches a pattern (supports wildcards)
func matchHost(host, pattern string) bool {
	// Simple exact match
	if host == pattern {
		return true
	}

	// Wildcard match
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // Remove the *
		return strings.HasSuffix(host, suffix)
	}

	// CIDR match
	if strings.Contains(pattern, "/") {
		_, ipNet, err := net.ParseCIDR(pattern)
		if err != nil {
			return false
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return false
		}
		return ipNet.Contains(ip)
	}

	return false
}
