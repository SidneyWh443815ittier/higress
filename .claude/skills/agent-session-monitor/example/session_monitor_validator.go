package main

import (
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a validation failure with context.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Validator provides input validation for session monitor data.
type Validator struct {
	MaxSessionIDLen int
	MaxURLLen       int
	AllowedSchemes  []string
}

// NewValidator returns a Validator with sensible defaults.
func NewValidator() *Validator {
	return &Validator{
		MaxSessionIDLen: 256,
		MaxURLLen:       2048,
		AllowedSchemes:  []string{"http", "https"},
	}
}

// ValidateSessionID checks that a session ID is non-empty, within length
// limits, and contains only safe characters.
func (v *Validator) ValidateSessionID(id string) error {
	if strings.TrimSpace(id) == "" {
		return &ValidationError{Field: "session_id", Message: "must not be empty"}
	}
	if !utf8.ValidString(id) {
		return &ValidationError{Field: "session_id", Message: "must be valid UTF-8"}
	}
	if len(id) > v.MaxSessionIDLen {
		return &ValidationError{
			Field:   "session_id",
			Message: "exceeds maximum length of " + itoa(v.MaxSessionIDLen),
		}
	}
	// Reject characters that could break log formats or URLs.
	for _, ch := range id {
		if ch == '\n' || ch == '\r' || ch == '\x00' {
			return &ValidationError{Field: "session_id", Message: "contains invalid control character"}
		}
	}
	return nil
}

// ValidateURL checks that a generated session URL is well-formed and uses an
// allowed scheme.
func (v *Validator) ValidateURL(raw string) error {
	if raw == "" {
		return &ValidationError{Field: "url", Message: "must not be empty"}
	}
	if len(raw) > v.MaxURLLen {
		return &ValidationError{
			Field:   "url",
			Message: "exceeds maximum length of " + itoa(v.MaxURLLen),
		}
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return &ValidationError{Field: "url", Message: "invalid URL: " + err.Error()}
	}
	if !v.schemeAllowed(u.Scheme) {
		return &ValidationError{
			Field:   "url",
			Message: "scheme '" + u.Scheme + "' is not allowed",
		}
	}
	return nil
}

// ValidateLogLine performs lightweight validation on a raw log line before
// parsing, rejecting obviously malformed input early.
func (v *Validator) ValidateLogLine(line string) error {
	if strings.TrimSpace(line) == "" {
		return errors.New("log line is empty")
	}
	if !utf8.ValidString(line) {
		return errors.New("log line contains invalid UTF-8")
	}
	return nil
}

// schemeAllowed returns true when scheme is in the allowed list.
func (v *Validator) schemeAllowed(scheme string) bool {
	for _, s := range v.AllowedSchemes {
		if strings.EqualFold(s, scheme) {
			return true
		}
	}
	return false
}

// itoa is a minimal int-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
