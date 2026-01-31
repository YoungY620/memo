package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HistoryLogger logs events to .memo/.history for debugging
type HistoryLogger struct {
	file   *os.File
	mu     sync.Mutex
	seqNum int64
	source string
}

// HistoryEntry represents a single log entry
type HistoryEntry struct {
	Seq       int64  `json:"seq"`
	Timestamp string `json:"ts"`
	Source    string `json:"src"`              // "mcp" or "watcher"
	Type      string `json:"type"`             // "request", "response", "error", "info", "debug"
	Method    string `json:"method,omitempty"` // for mcp requests
	ID        any    `json:"id,omitempty"`     // for mcp request/response correlation
	Params    any    `json:"params,omitempty"`
	Result    any    `json:"result,omitempty"`
	Error     any    `json:"error,omitempty"`
	Duration  string `json:"duration,omitempty"`
	Message   string `json:"msg,omitempty"`
}

// NewHistoryLogger creates a new history logger with given source
func NewHistoryLogger(memoDir, source string) (*HistoryLogger, error) {
	historyPath := filepath.Join(memoDir, ".history")
	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	return &HistoryLogger{file: f, source: source}, nil
}

// Log writes an entry to the history file
func (h *HistoryLogger) Log(entry HistoryEntry) {
	if h == nil || h.file == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	h.seqNum++
	entry.Seq = h.seqNum
	entry.Timestamp = time.Now().Format(time.RFC3339Nano)
	entry.Source = h.source

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = h.file.Write(data)
	_, _ = h.file.Write([]byte("\n"))
}

// LogError logs an error
func (h *HistoryLogger) LogError(message string, err error) {
	entry := HistoryEntry{Type: "error", Message: message}
	if err != nil {
		entry.Error = err.Error()
	}
	h.Log(entry)
}

// LogInfo logs an informational message
func (h *HistoryLogger) LogInfo(format string, v ...any) {
	msg := format
	if len(v) > 0 {
		msg = fmt.Sprintf(format, v...)
	}
	h.Log(HistoryEntry{Type: "info", Message: msg})
}

// LogDebug logs a debug message
func (h *HistoryLogger) LogDebug(format string, v ...any) {
	msg := format
	if len(v) > 0 {
		msg = fmt.Sprintf(format, v...)
	}
	h.Log(HistoryEntry{Type: "debug", Message: msg})
}

// Close closes the history file
func (h *HistoryLogger) Close() error {
	if h != nil && h.file != nil {
		return h.file.Close()
	}
	return nil
}
