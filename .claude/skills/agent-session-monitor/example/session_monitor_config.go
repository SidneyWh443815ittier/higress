package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the session monitor.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Log      LogConfig      `yaml:"log"`
	Session  SessionConfig  `yaml:"session"`
	Metrics  MetricsConfig  `yaml:"metrics"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// LogConfig holds log file processing settings.
type LogConfig struct {
	FilePath   string `yaml:"file_path"`
	Format     string `yaml:"format"`
	MaxLines   int    `yaml:"max_lines"`
	TailFollow bool   `yaml:"tail_follow"`
}

// SessionConfig holds session URL generation settings.
type SessionConfig struct {
	BaseURL        string        `yaml:"base_url"`
	TokenTTL       time.Duration `yaml:"token_ttl"`
	SecretKey      string        `yaml:"secret_key"`
	AllowedOrigins []string      `yaml:"allowed_origins"`
}

// MetricsConfig holds Prometheus metrics settings.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
	Port    int    `yaml:"port"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Log: LogConfig{
			FilePath:   "/var/log/higress/access.log",
			Format:     "json",
			MaxLines:   10000,
			TailFollow: false,
		},
		Session: SessionConfig{
			BaseURL:  "http://localhost:8080",
			TokenTTL: 24 * time.Hour,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
			Port:    9090,
		},
	}
}

// LoadConfig reads and parses a YAML config file, overlaying defaults.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults when no config file is present.
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks that required fields are set and values are in range.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port %d is out of range [1, 65535]", c.Server.Port)
	}
	if c.Log.FilePath == "" {
		return fmt.Errorf("log.file_path must not be empty")
	}
	if c.Session.BaseURL == "" {
		return fmt.Errorf("session.base_url must not be empty")
	}
	if c.Session.TokenTTL <= 0 {
		return fmt.Errorf("session.token_ttl must be positive")
	}
	return nil
}
