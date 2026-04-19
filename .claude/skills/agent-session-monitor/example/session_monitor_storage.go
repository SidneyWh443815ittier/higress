package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StorageBackend defines the interface for session data persistence
type StorageBackend interface {
	Save(sessionID string, data SessionRecord) error
	Load(sessionID string) (SessionRecord, error)
	Delete(sessionID string) error
	List() ([]SessionRecord, error)
	Close() error
}

// SessionRecord holds persisted session data
type SessionRecord struct {
	SessionID  string            `json:"session_id"`
	URL        string            `json:"url"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	RequestCount int             `json:"request_count"`
}

// FileStorage implements StorageBackend using local JSON files
type FileStorage struct {
	mu      sync.RWMutex
	dir     string
	records map[string]SessionRecord
}

// NewFileStorage creates a new file-based storage backend
func NewFileStorage(dir string) (*FileStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}

	fs := &FileStorage{
		dir:     dir,
		records: make(map[string]SessionRecord),
	}

	// Load existing records from disk
	if err := fs.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load existing records: %w", err)
	}

	return fs, nil
}

// Save persists a session record to disk
func (fs *FileStorage) Save(sessionID string, data SessionRecord) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data.UpdatedAt = time.Now()
	if data.CreatedAt.IsZero() {
		data.CreatedAt = data.UpdatedAt
	}

	fs.records[sessionID] = data
	return fs.writeToDisk(sessionID, data)
}

// Load retrieves a session record by ID
func (fs *FileStorage) Load(sessionID string) (SessionRecord, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	record, ok := fs.records[sessionID]
	if !ok {
		return SessionRecord{}, fmt.Errorf("session %q not found", sessionID)
	}
	return record, nil
}

// Delete removes a session record
func (fs *FileStorage) Delete(sessionID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	delete(fs.records, sessionID)
	path := filepath.Join(fs.dir, sessionID+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete record file: %w", err)
	}
	return nil
}

// List returns all stored session records
func (fs *FileStorage) List() ([]SessionRecord, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	result := make([]SessionRecord, 0, len(fs.records))
	for _, r := range fs.records {
		result = append(result, r)
	}
	return result, nil
}

// Close flushes any pending writes and releases resources
func (fs *FileStorage) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return nil
}

func (fs *FileStorage) writeToDisk(sessionID string, data SessionRecord) error {
	path := filepath.Join(fs.dir, sessionID+".json")
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}
	return os.WriteFile(path, b, 0644)
}

func (fs *FileStorage) loadFromDisk() error {
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(fs.dir, entry.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var record SessionRecord
		if err := json.Unmarshal(b, &record); err != nil {
			continue
		}
		fs.records[record.SessionID] = record
	}
	return nil
}
