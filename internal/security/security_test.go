package security

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	// Test with empty config
	config := Config{}
	manager := NewManager(config)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.rateLimiter == nil {
		t.Error("Rate limiter map not initialized")
	}

	// Test with populated config
	config = Config{
		AllowedHosts:    []string{"example.com"},
		DeniedHosts:     []string{"evil.com"},
		AllowedCommands: []string{"ls", "cat"},
		DeniedCommands:  []string{"rm", "sudo"},
		RateLimit:       time.Second,
		LoggingEnabled:  true,
	}

	manager = NewManager(config)
	if len(manager.config.AllowedHosts) != 1 || manager.config.AllowedHosts[0] != "example.com" {
		t.Error("AllowedHosts not set correctly")
	}

	if len(manager.config.DeniedHosts) != 1 || manager.config.DeniedHosts[0] != "evil.com" {
		t.Error("DeniedHosts not set correctly")
	}

	if len(manager.config.AllowedCommands) != 2 {
		t.Error("AllowedCommands not set correctly")
	}

	if len(manager.config.DeniedCommands) != 2 {
		t.Error("DeniedCommands not set correctly")
	}

	if manager.config.RateLimit != time.Second {
		t.Error("RateLimit not set correctly")
	}

	if !manager.config.LoggingEnabled {
		t.Error("LoggingEnabled not set correctly")
	}
}

func TestCheckHost(t *testing.T) {
	// Test with empty allowed/denied lists (all hosts allowed)
	manager := NewManager(Config{})

	err := manager.CheckHost("example.com")
	if err != nil {
		t.Errorf("Expected host to be allowed, got error: %v", err)
	}

	// Test with host in denied list
	manager = NewManager(Config{
		DeniedHosts: []string{"evil.com"},
	})

	err = manager.CheckHost("evil.com")
	if err == nil {
		t.Error("Expected host to be denied, but it was allowed")
	}

	// Test with host in allowed list
	manager = NewManager(Config{
		AllowedHosts: []string{"example.com"},
	})

	err = manager.CheckHost("example.com")
	if err != nil {
		t.Errorf("Expected host to be allowed, got error: %v", err)
	}

	// Test with host not in allowed list
	err = manager.CheckHost("unknown.com")
	if err == nil {
		t.Error("Expected host to be denied, but it was allowed")
	}

	// Test with host:port format
	err = manager.CheckHost("example.com:22")
	if err != nil {
		t.Errorf("Expected host with port to be allowed, got error: %v", err)
	}

	// Test with wildcard pattern
	manager = NewManager(Config{
		AllowedHosts: []string{"*.example.com"},
	})

	err = manager.CheckHost("sub.example.com")
	if err != nil {
		t.Errorf("Expected wildcard match to be allowed, got error: %v", err)
	}

	err = manager.CheckHost("other.com")
	if err == nil {
		t.Error("Expected non-matching host to be denied, but it was allowed")
	}

	// Test with CIDR pattern
	manager = NewManager(Config{
		AllowedHosts: []string{"192.168.1.0/24"},
	})

	err = manager.CheckHost("192.168.1.10")
	if err != nil {
		t.Errorf("Expected IP in CIDR range to be allowed, got error: %v", err)
	}

	err = manager.CheckHost("192.168.2.10")
	if err == nil {
		t.Error("Expected IP outside CIDR range to be denied, but it was allowed")
	}
}

func TestCheckCommand(t *testing.T) {
	// Test with empty allowed/denied lists (all commands allowed)
	manager := NewManager(Config{})

	err := manager.CheckCommand("session1", "ls -la")
	if err != nil {
		t.Errorf("Expected command to be allowed, got error: %v", err)
	}

	// Test with command in denied list
	manager = NewManager(Config{
		DeniedCommands: []string{"rm"},
	})

	err = manager.CheckCommand("session1", "rm -rf /")
	if err == nil {
		t.Error("Expected command to be denied, but it was allowed")
	}

	// Test with command in allowed list
	manager = NewManager(Config{
		AllowedCommands: []string{"ls", "cat"},
	})

	err = manager.CheckCommand("session1", "ls -la")
	if err != nil {
		t.Errorf("Expected command to be allowed, got error: %v", err)
	}

	// Test with command not in allowed list
	err = manager.CheckCommand("session1", "rm -rf /")
	if err == nil {
		t.Error("Expected command to be denied, but it was allowed")
	}

	// Test rate limiting
	manager = NewManager(Config{
		RateLimit: 100 * time.Millisecond,
	})

	// First command should be allowed
	err = manager.CheckCommand("session1", "command1")
	if err != nil {
		t.Errorf("Expected first command to be allowed, got error: %v", err)
	}

	// Second command should be rate limited
	err = manager.CheckCommand("session1", "command2")
	if err == nil {
		t.Error("Expected rate limiting to deny second command, but it was allowed")
	}

	// Wait for rate limit to expire
	time.Sleep(150 * time.Millisecond)

	// Third command should be allowed again
	err = manager.CheckCommand("session1", "command3")
	if err != nil {
		t.Errorf("Expected command after rate limit expiry to be allowed, got error: %v", err)
	}

	// Different session should not be affected by rate limit of first session
	err = manager.CheckCommand("session2", "command1")
	if err != nil {
		t.Errorf("Expected command from different session to be allowed, got error: %v", err)
	}
}

func TestCleanupRateLimiter(t *testing.T) {
	manager := NewManager(Config{})

	// Add some entries to the rate limiter
	now := time.Now()
	manager.rateLimiter["session1"] = now.Add(-10 * time.Minute) // Old entry
	manager.rateLimiter["session2"] = now                        // Recent entry

	// Cleanup entries older than 5 minutes
	manager.CleanupRateLimiter(5 * time.Minute)

	// Check if old entry was removed
	if _, exists := manager.rateLimiter["session1"]; exists {
		t.Error("Old rate limiter entry was not removed")
	}

	// Check if recent entry was kept
	if _, exists := manager.rateLimiter["session2"]; !exists {
		t.Error("Recent rate limiter entry was removed")
	}
}

func TestStartCleanupRoutine(t *testing.T) {
	// This is a simple test to ensure the function doesn't panic
	manager := NewManager(Config{})

	// Should not panic
	manager.StartCleanupRoutine(100*time.Millisecond, 1*time.Second)

	// Test with zero values (should use defaults)
	manager.StartCleanupRoutine(0, 0)

	// Allow some time for the goroutine to run
	time.Sleep(10 * time.Millisecond)
}

func TestMatchHost(t *testing.T) {
	// Test exact match
	if !matchHost("example.com", "example.com") {
		t.Error("Exact match failed")
	}

	if matchHost("example.com", "other.com") {
		t.Error("Non-matching hosts should not match")
	}

	// Test wildcard match
	if !matchHost("sub.example.com", "*.example.com") {
		t.Error("Wildcard match failed")
	}

	if matchHost("example.com", "*.example.com") {
		t.Error("Wildcard should not match parent domain")
	}

	if matchHost("otherexample.com", "*.example.com") {
		t.Error("Wildcard should not match unrelated domain")
	}

	// Test CIDR match
	if !matchHost("192.168.1.10", "192.168.1.0/24") {
		t.Error("CIDR match failed")
	}

	if matchHost("192.168.2.10", "192.168.1.0/24") {
		t.Error("IP outside CIDR range should not match")
	}

	// Test invalid CIDR
	if matchHost("192.168.1.10", "invalid-cidr") {
		t.Error("Invalid CIDR should not match")
	}

	// Test invalid IP
	if matchHost("not-an-ip", "192.168.1.0/24") {
		t.Error("Invalid IP should not match CIDR")
	}
}
