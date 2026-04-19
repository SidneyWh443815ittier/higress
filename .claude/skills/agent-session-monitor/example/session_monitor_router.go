package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// Router handles HTTP routing for the session monitor service.
type Router struct {
	mux     *http.ServeMux
	metrics *Metrics
	logger  *Logger
	config  *Config
}

// NewRouter creates a new Router with all routes registered.
func NewRouter(cfg *Config, metrics *Metrics, logger *Logger) *Router {
	r := &Router{
		mux:     http.NewServeMux(),
		metrics: metrics,
		logger:  logger,
		config:  cfg,
	}
	r.registerRoutes()
	return r
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// registerRoutes wires up all HTTP endpoints.
func (r *Router) registerRoutes() {
	// Health and readiness probes
	RegisterHealthRoutes(r.mux)

	// Session URL generation endpoint
	r.mux.HandleFunc("/session", r.handleSession)

	// Metrics endpoint (Prometheus-style text)
	r.mux.HandleFunc("/metrics", r.handleMetrics)

	// Catch-all
	r.mux.HandleFunc("/", r.handleNotFound)
}

// sessionRequest is the expected JSON body for POST /session.
type sessionRequest struct {
	TraceID   string `json:"trace_id"`
	ServiceID string `json:"service_id"`
}

// sessionResponse is returned by POST /session.
type sessionResponse struct {
	SessionURL string `json:"session_url"`
	GeneratedAt string `json:"generated_at"`
}

// handleSession accepts a trace/service pair and returns a session URL.
func (r *Router) handleSession(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body sessionRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		r.logger.Error("failed to decode session request", "error", err)
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if body.TraceID == "" || body.ServiceID == "" {
		http.Error(w, "trace_id and service_id are required", http.StatusBadRequest)
		return
	}

	url := GenerateSessionURL(r.config.BaseURL, body.ServiceID, body.TraceID)

	r.metrics.RequestsTotal.Inc()
	r.logger.Info("session URL generated", "trace_id", body.TraceID, "service_id", body.ServiceID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(sessionResponse{
		SessionURL:  url,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleMetrics returns a simple text summary of current metrics.
func (r *Router) handleMetrics(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	r.metrics.WriteText(w)
}

// handleNotFound returns a JSON 404 for unmatched routes.
func (r *Router) handleNotFound(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "not found",
		"path":  req.URL.Path,
	})
}
