package main

import (
	"fmt"
	"os"
	"testing"
)

// BenchmarkParseLogLine measures performance of log line parsing
func BenchmarkParseLogLine(b *testing.B) {
	line := `192.168.1.100 - - [01/Jan/2024:12:00:00 +0000] "POST /v1/chat/completions HTTP/1.1" 200 1234 "-" "curl/7.68.0" session_id=abc123 latency=150ms`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseLogLine(line)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkParseLogLineMalformed measures performance when parsing malformed lines
func BenchmarkParseLogLineMalformed(b *testing.B) {
	line := `this is not a valid log line`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//nolint:errcheck
		ParseLogLine(line)
	}
}

// BenchmarkGenerateSessionURL measures URL generation performance
func BenchmarkGenerateSessionURL(b *testing.B) {
	sessionID := "bench-session-abc123"
	baseURL := "https://monitor.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateSessionURL(sessionID, baseURL)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkProcessLogFile measures full file processing performance
func BenchmarkProcessLogFile(b *testing.B) {
	// Create a temp log file with multiple entries
	tmpFile, err := os.CreateTemp("", "bench_access_*.log")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write 100 log lines
	for i := 0; i < 100; i++ {
		line := fmt.Sprintf(
			`192.168.1.%d - - [01/Jan/2024:12:00:%02d +0000] "POST /v1/chat/completions HTTP/1.1" 200 1234 "-" "curl/7.68.0" session_id=session%d latency=150ms\n`,
			i%255, i%60, i,
		)
		if _, err := tmpFile.WriteString(line); err != nil {
			b.Fatalf("failed to write log line: %v", err)
		}
	}
	tmpFile.Close()

	baseURL := "https://monitor.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ProcessLogFile(tmpFile.Name(), baseURL)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkProcessLogFileParallel measures concurrent file processing
func BenchmarkProcessLogFileParallel(b *testing.B) {
	tmpFile, err := os.CreateTemp("", "bench_parallel_*.log")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	for i := 0; i < 50; i++ {
		line := fmt.Sprintf(
			`10.0.0.%d - - [01/Jan/2024:12:00:%02d +0000] "POST /v1/chat/completions HTTP/1.1" 200 512 "-" "agent/1.0" session_id=par%d latency=200ms\n`,
			i%255, i%60, i,
		)
		if _, err := tmpFile.WriteString(line); err != nil {
			b.Fatalf("failed to write: %v", err)
		}
	}
	tmpFile.Close()

	baseURL := "https://monitor.example.com"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := ProcessLogFile(tmpFile.Name(), baseURL)
			if err != nil {
				b.Errorf("unexpected error: %v", err)
			}
		}
	})
}
