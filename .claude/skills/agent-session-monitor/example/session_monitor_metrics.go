package main

import (
	"fmt"
	"sync"
	"time"
)

// Metrics tracks processing statistics for the session monitor.
type Metrics struct {
	mu             sync.RWMutex
	TotalLines     int64
	ParsedLines    int64
	FailedLines    int64
	SessionsFound  int64
	StartTime      time.Time
	LastProcessed  time.Time
}

// NewMetrics creates a new Metrics instance with start time set to now.
func NewMetrics() *Metrics {
	return &Metrics{
		StartTime: time.Now(),
	}
}

// RecordLine increments the total line counter.
func (m *Metrics) RecordLine() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalLines++
	m.LastProcessed = time.Now()
}

// RecordParsed increments the successfully parsed line counter.
func (m *Metrics) RecordParsed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ParsedLines++
}

// RecordFailed increments the failed parse counter.
func (m *Metrics) RecordFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailedLines++
}

// RecordSession increments the session URL generated counter.
func (m *Metrics) RecordSession() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SessionsFound++
}

// Summary returns a human-readable summary of collected metrics.
func (m *Metrics) Summary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	elapsed := time.Since(m.StartTime).Round(time.Millisecond)
	successRate := 0.0
	if m.TotalLines > 0 {
		successRate = float64(m.ParsedLines) / float64(m.TotalLines) * 100
	}

	return fmt.Sprintf(
		"Metrics Summary:\n"+
			"  Elapsed:       %s\n"+
			"  Total Lines:   %d\n"+
			"  Parsed:        %d (%.1f%%)\n"+
			"  Failed:        %d\n"+
			"  Sessions:      %d\n",
		elapsed,
		m.TotalLines,
		m.ParsedLines, successRate,
		m.FailedLines,
		m.SessionsFound,
	)
}

// Reset clears all counters and resets the start time.
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalLines = 0
	m.ParsedLines = 0
	m.FailedLines = 0
	m.SessionsFound = 0
	m.StartTime = time.Now()
	m.LastProcessed = time.Time{}
}
