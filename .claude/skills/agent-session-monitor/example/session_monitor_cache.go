package main

import (
	"sync"
	"time"
)

// CacheEntry holds a cached session URL with expiration
type CacheEntry struct {
	URL       string
	ExpiresAt time.Time
}

// SessionCache provides thread-safe caching for generated session URLs
type SessionCache struct {
	mu      sync.RWMutex
	entries map[string]CacheEntry
	ttl     time.Duration
}

// NewSessionCache creates a new SessionCache with the given TTL
func NewSessionCache(ttl time.Duration) *SessionCache {
	c := &SessionCache{
		entries: make(map[string]CacheEntry),
		ttl:     ttl,
	}
	go c.evictLoop()
	return c
}

// Get retrieves a cached URL by session ID, returning the URL and whether it was found
func (c *SessionCache) Get(sessionID string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[sessionID]
	if !ok {
		return "", false
	}
	if time.Now().After(entry.ExpiresAt) {
		return "", false
	}
	return entry.URL, true
}

// Set stores a session URL in the cache
func (c *SessionCache) Set(sessionID, url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[sessionID] = CacheEntry{
		URL:       url,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a session entry from the cache
func (c *SessionCache) Delete(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, sessionID)
}

// Size returns the current number of entries in the cache
func (c *SessionCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// evictLoop periodically removes expired entries from the cache
func (c *SessionCache) evictLoop() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.evictExpired()
	}
}

// evictExpired removes all expired entries
func (c *SessionCache) evictExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, id)
		}
	}
}
