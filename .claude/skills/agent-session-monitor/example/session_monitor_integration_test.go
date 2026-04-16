package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIntegrationProcessLogFile tests the full pipeline with a real temp file
func TestIntegrationProcessLogFile(t *testing.T) {
	logLines := []string{
		`{"time":"2024-01-15T10:00:00Z","session_id":"sess-001","user_id":"user-42","request_id":"req-aaa","method":"POST","path":"/v1/chat","status":200,"latency_ms":320}`,
		`{"time":"2024-01-15T10:01:00Z","session_id":"sess-001","user_id":"user-42","request_id":"req-bbb","method":"POST","path":"/v1/chat","status":200,"latency_ms":410}`,
		`{"time":"2024-01-15T10:02:00Z","session_id":"sess-002","user_id":"user-99","request_id":"req-ccc","method":"POST","path":"/v1/chat","status":500,"latency_ms":150}`,
	}

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "access.log")

	err := os.WriteFile(logFile, []byte(strings.Join(logLines, "\n")+"\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write temp log file: %v", err)
	}

	sessions, err := ProcessLogFile(logFile)
	if err != nil {
		t.Fatalf("ProcessLogFile returned error: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}

	sess1, ok := sessions["sess-001"]
	if !ok {
		t.Fatal("session sess-001 not found")
	}
	if len(sess1.Requests) != 2 {
		t.Errorf("sess-001: expected 2 requests, got %d", len(sess1.Requests))
	}

	sess2, ok := sessions["sess-002"]
	if !ok {
		t.Fatal("session sess-002 not found")
	}
	if sess2.ErrorCount != 1 {
		t.Errorf("sess-002: expected 1 error, got %d", sess2.ErrorCount)
	}
}

// TestIntegrationGenerateSessionURL tests URL generation against a mock server
func TestIntegrationGenerateSessionURL(t *testing.T) {
	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sessionID := "sess-integration-001"
	url := GenerateSessionURL(ts.URL, sessionID)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("failed to GET generated URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(receivedPath, sessionID) {
		t.Errorf("expected path to contain session ID %q, got %q", sessionID, receivedPath)
	}
}

// TestIntegrationEmptyLogFile ensures empty files are handled gracefully
func TestIntegrationEmptyLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "empty.log")

	err := os.WriteFile(logFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("failed to create empty log file: %v", err)
	}

	sessions, err := ProcessLogFile(logFile)
	if err != nil {
		t.Fatalf("unexpected error on empty file: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for empty file, got %d", len(sessions))
	}
}

// TestIntegrationMalformedLines ensures malformed JSON lines are skipped without crashing
func TestIntegrationMalformedLines(t *testing.T) {
	lines := []string{
		`not-valid-json`,
		`{"session_id":"sess-ok","status":200,"latency_ms":100}`,
		`{broken json`,
	}

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "mixed.log")

	err := os.WriteFile(logFile, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write mixed log file: %v", err)
	}

	sessions, err := ProcessLogFile(logFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only the valid line with a session_id should be counted
	if len(sessions) != 1 {
		t.Errorf("expected 1 valid session, got %d", len(sessions))
	}
}
