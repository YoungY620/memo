package internal_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YoungY620/memo/internal"
)

func TestNewHistoryLogger(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	os.MkdirAll(memoDir, 0755)

	logger, err := internal.NewHistoryLogger(memoDir, "test")
	if err != nil {
		t.Fatalf("Failed to create history logger: %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Error("Expected non-nil logger")
	}

	// History file should exist
	historyPath := filepath.Join(memoDir, ".history")
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Error("History file was not created")
	}
}

func TestHistoryLogger_Log(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	os.MkdirAll(memoDir, 0755)

	logger, err := internal.NewHistoryLogger(memoDir, "test")
	if err != nil {
		t.Fatalf("Failed to create history logger: %v", err)
	}

	// Log some entries
	logger.Log(internal.HistoryEntry{
		Type:    "request",
		Method:  "test_method",
		ID:      1,
		Message: "test message",
	})

	logger.LogInfo("info message")
	logger.LogDebug("debug message")
	logger.LogError("error message", nil)

	logger.Close()

	// Read and verify the log file
	historyPath := filepath.Join(memoDir, ".history")
	data, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("Failed to read history file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 4 {
		t.Errorf("Expected 4 log entries, got %d", len(lines))
	}

	// Verify first entry
	var entry internal.HistoryEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	if entry.Type != "request" {
		t.Errorf("Expected type 'request', got '%s'", entry.Type)
	}
	if entry.Source != "test" {
		t.Errorf("Expected source 'test', got '%s'", entry.Source)
	}
	if entry.Seq != 1 {
		t.Errorf("Expected seq 1, got %d", entry.Seq)
	}
}

func TestHistoryLogger_NilSafe(t *testing.T) {
	var logger *internal.HistoryLogger

	// These should not panic
	logger.Log(internal.HistoryEntry{Type: "test"})
	logger.LogInfo("test")
	logger.LogDebug("test")
	logger.LogError("test", nil)
	logger.Close()
}

func TestHistoryLogger_ErrorWithErr(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	os.MkdirAll(memoDir, 0755)

	logger, _ := internal.NewHistoryLogger(memoDir, "test")
	defer logger.Close()

	// Log error with actual error
	logger.LogError("something failed", os.ErrNotExist)

	logger.Close()

	historyPath := filepath.Join(memoDir, ".history")
	data, _ := os.ReadFile(historyPath)

	if !strings.Contains(string(data), "file does not exist") {
		t.Error("Error message should contain the error text")
	}
}
