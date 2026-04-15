// Package main provides a session monitoring utility for Higress agent sessions.
// It parses access logs and generates session URLs for debugging and monitoring.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// SessionEntry represents a parsed log entry containing session information.
type SessionEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	SessionID  string    `json:"session_id"`
	RequestID  string    `json:"request_id"`
	ClientIP   string    `json:"client_ip"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int64     `json:"latency_ms"`
	Upstream   string    `json:"upstream"`
}

// MonitorConfig holds configuration for the session monitor.
type MonitorConfig struct {
	LogFile    string
	OutputJSON bool
	BaseURL    string
	FilterID   string
}

// logPattern matches the Higress access log format.
// Example: [2024-01-15T10:30:00Z] session=abc123 request=req456 ...
var logPattern = regexp.MustCompile(
	`\[(\S+)\]\s+session=(\S+)\s+request=(\S+)\s+client=(\S+)\s+(\S+)\s+(\S+)\s+status=(\d+)\s+latency=(\d+)ms\s+upstream=(\S+)`,
)

// ParseLogLine parses a single access log line into a SessionEntry.
func ParseLogLine(line string) (*SessionEntry, error) {
	matches := logPattern.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("line does not match expected format")
	}

	ts, err := time.Parse(time.RFC3339, matches[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp %q: %w", matches[1], err)
	}

	var statusCode, latencyMs int
	fmt.Sscanf(matches[7], "%d", &statusCode)
	fmt.Sscanf(matches[8], "%d", &latencyMs)

	return &SessionEntry{
		Timestamp:  ts,
		SessionID:  matches[2],
		RequestID:  matches[3],
		ClientIP:   matches[4],
		Method:     matches[5],
		Path:       matches[6],
		StatusCode: statusCode,
		LatencyMs:  int64(latencyMs),
		Upstream:   matches[9],
	}, nil
}

// GenerateSessionURL builds a session monitor URL for a given session ID.
func GenerateSessionURL(baseURL, sessionID string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	return fmt.Sprintf("%s/session/%s", baseURL, sessionID)
}

// ProcessLogFile reads and parses all session entries from the given log file.
func ProcessLogFile(path string) ([]*SessionEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	var entries []*SessionEntry
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entry, err := ParseLogLine(line)
		if err != nil {
			log.Printf("WARN: skipping line %d: %v", lineNum, err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

func main() {
	cfg := &MonitorConfig{}
	flag.StringVar(&cfg.LogFile, "log", "test_access.log", "Path to the access log file")
	flag.BoolVar(&cfg.OutputJSON, "json", false, "Output results as JSON")
	flag.StringVar(&cfg.BaseURL, "base-url", "http://localhost:8080", "Base URL for session monitor")
	flag.StringVar(&cfg.FilterID, "session", "", "Filter output to a specific session ID")
	flag.Parse()

	entries, err := ProcessLogFile(cfg.LogFile)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	if cfg.FilterID != "" {
		var filtered []*SessionEntry
		for _, e := range entries {
			if e.SessionID == cfg.FilterID {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if cfg.OutputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(entries); err != nil {
			log.Fatalf("ERROR: failed to encode JSON: %v", err)
		}
		return
	}

	for _, e := range entries {
		url := GenerateSessionURL(cfg.BaseURL, e.SessionID)
		fmt.Printf("[%s] session=%s status=%d latency=%dms url=%s\n",
			e.Timestamp.Format(time.RFC3339), e.SessionID, e.StatusCode, e.LatencyMs, url)
	}

	fmt.Printf("\nTotal entries: %d\n", len(entries))
}
