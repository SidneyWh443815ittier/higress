package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NotifierConfig holds configuration for session event notifications
type NotifierConfig struct {
	WebhookURL    string        `yaml:"webhook_url"`
	Timeout       time.Duration `yaml:"timeout"`
	RetryCount    int           `yaml:"retry_count"`
	RetryInterval time.Duration `yaml:"retry_interval"`
	Enabled       bool          `yaml:"enabled"`
}

// SessionEvent represents a session lifecycle event
type SessionEvent struct {
	SessionID  string            `json:"session_id"`
	EventType  string            `json:"event_type"` // created, updated, expired
	Timestamp  time.Time         `json:"timestamp"`
	SessionURL string            `json:"session_url,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Notifier sends session events to configured webhooks
type Notifier struct {
	config NotifierConfig
	client *http.Client
	logger *Logger
}

// NewNotifier creates a new Notifier instance
func NewNotifier(cfg NotifierConfig, logger *Logger) *Notifier {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.RetryCount == 0 {
		cfg.RetryCount = 3
	}
	if cfg.RetryInterval == 0 {
		cfg.RetryInterval = 500 * time.Millisecond
	}
	return &Notifier{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		logger: logger,
	}
}

// Notify sends a session event notification
func (n *Notifier) Notify(event SessionEvent) error {
	if !n.config.Enabled || n.config.WebhookURL == "" {
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= n.config.RetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(n.config.RetryInterval)
			n.logger.Info(fmt.Sprintf("retrying notification attempt %d/%d", attempt, n.config.RetryCount))
		}

		if err := n.sendWebhook(payload); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("notification failed after %d attempts: %w", n.config.RetryCount, lastErr)
}

// sendWebhook performs the HTTP POST to the webhook endpoint
func (n *Notifier) sendWebhook(payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, n.config.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "higress-session-monitor/1.0")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}
	return nil
}

// NotifyAsync sends a session event notification asynchronously
func (n *Notifier) NotifyAsync(event SessionEvent) {
	go func() {
		if err := n.Notify(event); err != nil {
			n.logger.Error(fmt.Sprintf("async notification error: %v", err))
		}
	}()
}
