package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// MiddlewareConfig holds configuration for the HTTP middleware
type MiddlewareConfig struct {
	EnableMetrics  bool
	EnableLogging  bool
	SessionHeader  string
	RedirectBase   string
}

// DefaultMiddlewareConfig returns a MiddlewareConfig with sensible defaults
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		EnableMetrics: true,
		EnableLogging: true,
		SessionHeader: "X-Session-ID",
		RedirectBase:  "http://localhost:3000",
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	bytesWritten int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// SessionMonitorMiddleware injects session tracking into HTTP handlers
func SessionMonitorMiddleware(cfg MiddlewareConfig, metrics *Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		// Extract or generate session ID
		sessionID := r.Header.Get(cfg.SessionHeader)
		if sessionID == "" {
			sessionID = fmt.Sprintf("auto-%d", time.Now().UnixNano())
		}

		// Attach session URL to response header
		sessionURL := GenerateSessionURL(cfg.RedirectBase, sessionID)
		w.Header().Set("X-Session-URL", sessionURL)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		if cfg.EnableMetrics && metrics != nil {
			metrics.RecordRequest(r.Method, r.URL.Path, rw.statusCode, duration)
		}

		if cfg.EnableLogging {
			log.Printf("method=%s path=%s status=%d duration=%s session=%s",
				r.Method, r.URL.Path, rw.statusCode, duration, sessionID)
		}
	})
}

// SessionRedirectHandler returns an HTTP handler that redirects to the session monitor UI
func SessionRedirectHandler(baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "missing session_id parameter", http.StatusBadRequest)
			return
		}
		target := GenerateSessionURL(baseURL, sessionID)
		http.Redirect(w, r, target, http.StatusFound)
	}
}

// HealthHandler returns a simple health check endpoint
func HealthHandler(metrics *Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if metrics != nil {
			fmt.Fprintf(w, `{"status":"ok","requests_total":%d}`, metrics.RequestsTotal())
		} else {
			fmt.Fprint(w, `{"status":"ok"}`)
		}
	}
}
