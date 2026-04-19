package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// GracefulShutdown manages graceful shutdown of the HTTP server.
// It listens for OS signals and initiates a clean shutdown with a timeout.
type GracefulShutdown struct {
	server  *http.Server
	logger  *Logger
	timeout time.Duration
}

// NewGracefulShutdown creates a new GracefulShutdown handler.
func NewGracefulShutdown(server *http.Server, logger *Logger, timeout time.Duration) *GracefulShutdown {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &GracefulShutdown{
		server:  server,
		logger:  logger,
		timeout: timeout,
	}
}

// Wait blocks until a termination signal is received, then shuts down the server.
// Supported signals: SIGINT, SIGTERM.
func (g *GracefulShutdown) Wait() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	g.logger.Info("received shutdown signal", map[string]interface{}{
		"signal": sig.String(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	g.logger.Info("shutting down server", map[string]interface{}{
		"timeout_seconds": g.timeout.Seconds(),
	})

	if err := g.server.Shutdown(ctx); err != nil {
		g.logger.Error("server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	g.logger.Info("server exited cleanly", nil)
	return nil
}

// RunWithGracefulShutdown starts the server and blocks until shutdown completes.
// It returns any error from either ListenAndServe or Shutdown.
func RunWithGracefulShutdown(srv *Server, logger *Logger, shutdownTimeout time.Duration) error {
	httpServer := srv.HTTPServer()
	gs := NewGracefulShutdown(httpServer, logger, shutdownTimeout)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", map[string]interface{}{
			"addr": httpServer.Addr,
		})
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for either a startup error or a shutdown signal.
	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("server failed to start", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
		return nil
	case <-waitForSignal():
		return gs.Wait()
	}
}

// waitForSignal returns a channel that receives when SIGINT or SIGTERM is sent.
func waitForSignal() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}
