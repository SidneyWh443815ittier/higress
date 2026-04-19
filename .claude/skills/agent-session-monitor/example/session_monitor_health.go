package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// HealthStatus represents the health state of the service
type HealthStatus string

const (
	HealthStatusOK      HealthStatus = "ok"
	HealthStatusDegraded HealthStatus = "degraded"
	HealthStatusDown    HealthStatus = "down"
)

// HealthResponse is the JSON response for health check endpoints
type HealthResponse struct {
	Status    HealthStatus       `json:"status"`
	Timestamp string             `json:"timestamp"`
	Uptime    string             `json:"uptime"`
	Checks    map[string]Check   `json:"checks"`
	Metainfo  map[string]string  `json:"metainfo,omitempty"`
}

// Check represents a single health check result
type Check struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
}

// HealthHandler manages health and readiness endpoints
type HealthHandler struct {
	start      time.Time
	ready      atomic.Bool
	metrics    *Metrics
	config     *Config
}

// NewHealthHandler creates a new HealthHandler instance
func NewHealthHandler(metrics *Metrics, config *Config) *HealthHandler {
	h := &HealthHandler{
		start:   time.Now(),
		metrics: metrics,
		config:  config,
	}
	h.ready.Store(false)
	return h
}

// SetReady marks the service as ready to serve traffic
func (h *HealthHandler) SetReady(ready bool) {
	h.ready.Store(ready)
}

// LivenessHandler responds to Kubernetes liveness probes
// Returns 200 if the process is alive, 500 otherwise
func (h *HealthHandler) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    HealthStatusOK,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    fmt.Sprintf("%.0fs", time.Since(h.start).Seconds()),
		Checks: map[string]Check{
			"process": {Status: HealthStatusOK, Message: "running"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// Readinessif ready serve, 503 if not yet ready
func (h *(w http.ResponseWriter, r}
	overall := HealthStatusOK

	// Check if service marked ready
	if !h.ready.Load() {
		checks["ready"] = Check{Status: HealthStatusDown, Message: "service initializing"}
		overall = HealthStatusDown
	} else {
		checks["ready"] = Check{Status: HealthStatusOK, Message: "accepting traffic"}
	}

	// Check memory usage
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	memMB := ms.Alloc / 1024 / 1024
	if memMB > 500 {
		checks["memory"] = Check{Status: HealthStatusDegraded, Message: fmt.Sprintf("high memory usage: %dMB", memMB)}
		if overall == HealthStatusOK {
			overall = HealthStatusDegraded
		}
	} else {
		checks["memory"] = Check{Status: HealthStatusOK, Message: fmt.Sprintf("%dMB in use", memMB)}
	}

	resp := HealthResponse{
		Status:    overall,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    fmt.Sprintf("%.0fs", time.Since(h.start).Seconds()),
		Checks:    checks,
		Metainfo: map[string]string{
			"version":    "1.0.0",
			"go_version": runtime.Version(),
		},
	}

	statusCode := http.StatusOK
	if overall == HealthStatusDown {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// RegisterHealthRoutes registers health endpoints on the given mux
func RegisterHealthRoutes(mux *http.ServeMux, handler *HealthHandler) {
	mux.HandleFunc("/healthz", handler.LivenessHandler)
	mux.HandleFunc("/readyz", handler.ReadinessHandler)
}
