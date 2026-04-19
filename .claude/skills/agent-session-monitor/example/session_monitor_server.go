package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server wraps the HTTP server with session monitor dependencies.
type Server struct {
	httpServer *http.Server
	config     *Config
	metrics    *Metrics
	logger     *log.Logger
}

// NewServer creates a new Server with the provided config.
func NewServer(cfg *Config) *Server {
	logger := log.New(os.Stdout, "[session-monitor] ", log.LstdFlags)
	metrics := NewMetrics()

	mux := http.NewServeMux()

	// Register health routes
	RegisterHealthRoutes(mux)

	// Session URL generation endpoint
	mux.HandleFunc("/session/url", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "missing session_id", http.StatusBadRequest)
			return
		}
		url := GenerateSessionURL(cfg.BaseURL, sessionID)
		metrics.IncrementRequests()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"url": url})
	})

	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics.Snapshot())
	})

	// Apply middleware
	handler := SessionMonitorMiddleware(DefaultMiddlewareConfig())(mux)

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.ListenAddr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		},
		config:  cfg,
		metrics: metrics,
		logger:  logger,
	}
}

// Start begins listening and blocks until the server shuts down.
func (s *Server) Start() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		s.logger.Printf("listening on %s", s.config.ListenAddr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		s.logger.Printf("received signal %v, shutting down", sig)
	}

	return s.Shutdown()
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.logger.Println("server stopped")
	return s.httpServer.Shutdown(ctx)
}
