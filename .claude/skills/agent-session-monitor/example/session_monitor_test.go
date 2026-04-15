package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseLogLine tests the ParseLogLine function with various log formats
func TestParseLogLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantSession string
		wantUser    string
		wantErr     bool
	}{
		{
			name:        "valid log line with session and user",
			line:        `192.168.1.1 - - [01/Jan/2024:12:00:00 +0000] "GET /api/chat?session_id=abc123&user_id=user456 HTTP/1.1" 200 512`,
			wantSession: "abc123",
			wantUser:    "user456",
			wantErr:     false,
		},
		{
			name:        "log line with session only",
			line:        `10.0.0.1 - - [01/Jan/2024:12:01:00 +0000] "POST /api/agent?session_id=sess789 HTTP/1.1" 200 1024`,
			wantSession: "sess789",
			wantUser:    "",
			wantErr:     false,
		},
		{
			name:        "log line without session",
			line:        `127.0.0.1 - - [01/Jan/2024:12:02:00 +0000] "GET /health HTTP/1.1" 200 2`,
			wantSession: "",
			wantUser:    "",
			wantErr:     true,
		},
		{
			name:    "empty log line",
			line:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID, userID, err := ParseLogLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLogLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if sessionID != tt.wantSession {
				t.Errorf("ParseLogLine() sessionID = %v, want %v", sessionID, tt.wantSession)
			}
			if userID != tt.wantUser {
				t.Errorf("ParseLogLine() userID = %v, want %v", userID, tt.wantUser)
			}
		})
	}
}

// TestGenerateSessionURL tests URL generation for session monitoring
func TestGenerateSessionURL(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		userID    string
		wantURL   string
	}{
		{
			name:      "session and user",
			sessionID: "abc123",
			userID:    "user456",
			wantURL:   "https://monitor.example.com/sessions/abc123?user_id=user456",
		},
		{
			name:      "session only",
			sessionID: "sess789",
			userID:    "",
			wantURL:   "https://monitor.example.com/sessions/sess789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL := GenerateSessionURL(tt.sessionID, tt.userID)
			if gotURL != tt.wantURL {
				t.Errorf("GenerateSessionURL() = %v, want %v", gotURL, tt.wantURL)
			}
		})
	}
}

// TestProcessLogFile tests end-to-end log file processing
func TestProcessLogFile(t *testing.T) {
	// Create a temporary log file for testing
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test_access.log")

	logContent := `192.168.1.1 - - [01/Jan/2024:12:00:00 +0000] "GET /api/chat?session_id=abc123&user_id=user456 HTTP/1.1" 200 512
10.0.0.1 - - [01/Jan/2024:12:01:00 +0000] "POST /api/agent?session_id=sess789 HTTP/1.1" 200 1024
127.0.0.1 - - [01/Jan/2024:12:02:00 +0000] "GET /health HTTP/1.1" 200 2
`

	if err := os.WriteFile(logFile, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to create test log file: %v", err)
	}

	results, err := ProcessLogFile(logFile)
	if err != nil {
		t.Fatalf("ProcessLogFile() unexpected error: %v", err)
	}

	// Expect 2 sessions (health check line should be skipped)
	if len(results) != 2 {
		t.Errorf("ProcessLogFile() returned %d results, want 2", len(results))
	}

	// Verify first session
	if results[0].SessionID != "abc123" {
		t.Errorf("results[0].SessionID = %v, want abc123", results[0].SessionID)
	}
}
