package internal_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/YoungY620/memo/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHistoryLogger(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	logger, err := internal.NewHistoryLogger(memoDir, "test")
	require.NoError(t, err)
	defer logger.Close()

	assert.NotNil(t, logger)

	// History file should exist
	historyPath := filepath.Join(memoDir, ".history")
	_, err = os.Stat(historyPath)
	assert.NoError(t, err, "History file should be created")
}

func TestHistoryLogger_Log(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	logger, err := internal.NewHistoryLogger(memoDir, "test")
	require.NoError(t, err)

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
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 4, "Should have 4 log entries")

	// Verify first entry
	var entry internal.HistoryEntry
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &entry))

	assert.Equal(t, "request", entry.Type)
	assert.Equal(t, "test", entry.Source)
	assert.Equal(t, int64(1), entry.Seq)
}

func TestHistoryLogger_NilSafe(t *testing.T) {
	var logger *internal.HistoryLogger

	// These should not panic
	assert.NotPanics(t, func() {
		logger.Log(internal.HistoryEntry{Type: "test"})
		logger.LogInfo("test")
		logger.LogDebug("test")
		logger.LogError("test", nil)
		logger.Close()
	})
}

func TestHistoryLogger_ErrorWithErr(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	logger, _ := internal.NewHistoryLogger(memoDir, "test")

	// Log error with actual error
	logger.LogError("something failed", os.ErrNotExist)

	logger.Close()

	historyPath := filepath.Join(memoDir, ".history")
	data, _ := os.ReadFile(historyPath)

	assert.Contains(t, string(data), "file does not exist")
}

func TestNewHistoryLogger_DirNotExist(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "nonexistent", ".memo")

	logger, err := internal.NewHistoryLogger(nonExistentDir, "test")
	assert.Error(t, err, "Should fail when directory doesn't exist")
	assert.Nil(t, logger)
}

func TestHistoryLogger_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	logger, err := internal.NewHistoryLogger(memoDir, "test")
	require.NoError(t, err)
	defer logger.Close()

	// Concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 10
	numLogs := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numLogs; j++ {
				logger.LogInfo("message from goroutine %d, log %d", id, j)
			}
		}(i)
	}
	wg.Wait()
	logger.Close()

	// Verify all entries were written
	historyPath := filepath.Join(memoDir, ".history")
	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Equal(t, numGoroutines*numLogs, len(lines), "All log entries should be written")

	// Verify each line is valid JSON
	for i, line := range lines {
		var entry internal.HistoryEntry
		err := json.Unmarshal([]byte(line), &entry)
		assert.NoError(t, err, "Line %d should be valid JSON", i)
	}
}

func TestHistoryLogger_SeqMonotonic(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	logger, err := internal.NewHistoryLogger(memoDir, "test")
	require.NoError(t, err)

	// Log multiple entries
	for i := 0; i < 10; i++ {
		logger.LogInfo("message %d", i)
	}
	logger.Close()

	// Read and verify sequence numbers
	historyPath := filepath.Join(memoDir, ".history")
	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var lastSeq int64 = 0

	for i, line := range lines {
		var entry internal.HistoryEntry
		require.NoError(t, json.Unmarshal([]byte(line), &entry))

		assert.Greater(t, entry.Seq, lastSeq, "Seq should be monotonically increasing at line %d", i)
		lastSeq = entry.Seq
	}
}

func TestHistoryLogger_LogInfoFormat(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	logger, err := internal.NewHistoryLogger(memoDir, "test")
	require.NoError(t, err)

	// Test formatted logging
	logger.LogInfo("Value: %d, String: %s", 42, "hello")
	logger.Close()

	historyPath := filepath.Join(memoDir, ".history")
	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "Value: 42, String: hello")
}
