package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2EFullPipeline tests the complete pipeline from log file to session URLs
func TestE2EFullPipeline(t *testing.T) {
	// Create a temporary directory for test artifacts
	tmpDir, err := os.MkdirTemp("", "session-monitor-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write a realistic access log with multiple sessions
	logContent := `2024-01-15T10:00:00Z session-abc123 user-1 GET /api/chat 200 150ms
2024-01-15T10:00:01Z session-abc123 user-1 POST /api/message 201 200ms
2024-01-15T10:00:02Z session-def456 user-2 GET /api/chat 200 120ms
2024-01-15T10:00:03Z session-abc123 user-1 GET /api/status 200 50ms
2024-01-15T10:00:04Z session-def456 user-2 POST /api/message 201 180ms
`
	logFile := filepath.Join(tmpDir, "access.log")
	if err := os.WriteFile(logFile, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// Process the log file
	sessions, err := ProcessLogFile(logFile)
	if err != nil {
		t.Fatalf("ProcessLogFile failed: %v", err)
	}

	// Validate session count
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}

	// Validate session IDs are present
	expectedSessions := map[string]int{
		"session-abc123": 3,
		"session-def456": 2,
	}
	for sessionID, expectedCount := range expectedSessions {
		count, ok := sessions[sessionID]
		if !ok {
			t.Errorf("expected session %s not found", sessionID)
			continue
		}
		if count != expectedCount {
			t.Errorf("session %s: expected %d requests, got %d", sessionID, expectedCount, count)
		}
	}
}

// TestE2ESessionURLGeneration validates URL generation for all sessions in a log
func TestE2ESessionURLGeneration(t *testing.T) {
	baseURL := "https://monitor.example.com"
	sessionIDs := []string{"session-abc123", "session-def456", "session-ghi789"}

	for _, sessionID := range sessionIDs {
		url, err := GenerateSessionURL(baseURL, sessionID)
		if err != nil {
			t.Errorf("GenerateSessionURL(%q) failed: %v", sessionID, err)
			continue
		}
		if !strings.HasPrefix(url, baseURL) {
			t.Errorf("URL %q does not start with base URL %q", url, baseURL)
		}
		if !strings.Contains(url, sessionID) {
			t.Errorf("URL %q does not contain session ID %q", url, sessionID)
		}
	}
}

// TestE2ELargeLogFile tests processing performance with a larger log file
func TestE2ELargeLogFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "session-monitor-large-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate a large log file with 1000 entries across 50 sessions
	var sb strings.Builder
	numSessions := 50
	for i := 0; i < 1000; i++ {
		sessionID := fmt.Sprintf("session-%03d", i%numSessions)
		line := fmt.Sprintf("2024-01-15T10:%02d:%02dZ %s user-%d GET /api/chat 200 100ms\n",
			(i/60)%60, i%60, sessionID, i%numSessions)
		sb.WriteString(line)
	}

	logFile := filepath.Join(tmpDir, "large_access.log")
	if err := os.WriteFile(logFile, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("failed to write large log file: %v", err)
	}

	sessions, err := ProcessLogFile(logFile)
	if err != nil {
		t.Fatalf("ProcessLogFile failed on large file: %v", err)
	}

	if len(sessions) != numSessions {
		t.Errorf("expected %d sessions, got %d", numSessions, len(sessions))
	}
}
