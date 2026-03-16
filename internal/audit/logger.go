package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is the JSONL payload written to the audit log.
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Payload   any       `json:"payload"`
}

// Logger appends audit events to a single local file.
type Logger struct {
	mu  sync.Mutex
	enc *json.Encoder
	f   *os.File
}

// New opens or creates an audit log file.
func New(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Logger{enc: json.NewEncoder(f), f: f}, nil
}

// Close closes the underlying file handle.
func (l *Logger) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Close()
}

// Write encodes one event as a single JSON line.
func (l *Logger) Write(eventType string, payload any) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.enc.Encode(Event{
		Timestamp: time.Now().UTC(),
		Type:      eventType,
		Payload:   payload,
	})
}
