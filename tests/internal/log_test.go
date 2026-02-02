package internal_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YoungY620/memo/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetLogLevel(t *testing.T) {
	testCases := []struct {
		level    string
		expected string
	}{
		{"error", "error"},
		{"ERROR", "error"},
		{"notice", "notice"},
		{"info", "info"},
		{"debug", "debug"},
		{"invalid", "info"}, // default fallback
		{"", "info"},        // empty defaults to info
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			// SetLogLevel should not panic
			assert.NotPanics(t, func() {
				internal.SetLogLevel(tc.level)
			})
		})
	}
}

func TestLogFunctions(t *testing.T) {
	// These should not panic at any log level
	internal.SetLogLevel("debug")

	assert.NotPanics(t, func() {
		internal.LogError("test error %s", "arg")
		internal.LogNotice("test notice %s", "arg")
		internal.LogInfo("test info %s", "arg")
		internal.LogDebug("test debug %s", "arg")
	})

	internal.SetLogLevel("error")
	assert.NotPanics(t, func() {
		internal.LogDebug("this should be suppressed")
	})
}

func TestInitHistoryLogger(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	// Initialize should not panic
	assert.NotPanics(t, func() {
		internal.InitHistoryLogger(memoDir, "test")
	})

	// Close should not panic
	assert.NotPanics(t, func() {
		internal.CloseHistoryLogger()
	})

	// Double close should not panic
	assert.NotPanics(t, func() {
		internal.CloseHistoryLogger()
	})
}

func TestInitHistoryLogger_InvalidDir(t *testing.T) {
	// Initialize with non-existent dir should not panic (just logs error)
	assert.NotPanics(t, func() {
		internal.InitHistoryLogger("/nonexistent/path/.memo", "test")
	})
}

func TestLineBuffer(t *testing.T) {
	lb := internal.NewLineBuffer(100 * time.Millisecond)

	// Test writing and flushing complete lines
	lb.Write("line1\nline2\n")
	result := lb.Flush(false)
	assert.Equal(t, "line1\nline2", result)

	// Test partial line (should not flush without force)
	lb.Write("partial")
	result = lb.Flush(false)
	assert.Empty(t, result, "Partial line should not flush")

	// Test force flush
	result = lb.Flush(true)
	assert.Equal(t, "partial", result)

	// Buffer should be empty now
	result = lb.Flush(true)
	assert.Empty(t, result)
}

func TestLineBuffer_Timeout(t *testing.T) {
	lb := internal.NewLineBuffer(50 * time.Millisecond)

	lb.Write("partial")

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	result := lb.Flush(false)
	assert.Equal(t, "partial", result, "Should flush after timeout")
}

func TestLineBuffer_MixedContent(t *testing.T) {
	lb := internal.NewLineBuffer(100 * time.Millisecond)

	lb.Write("line1\n")
	lb.Write("line2\npartial")

	result := lb.Flush(false)
	assert.Equal(t, "line1\nline2", result)

	// Partial should still be in buffer
	result = lb.Flush(true)
	assert.Equal(t, "partial", result)
}

func TestLineBuffer_Empty(t *testing.T) {
	lb := internal.NewLineBuffer(100 * time.Millisecond)

	// Flush empty buffer
	result := lb.Flush(false)
	assert.Empty(t, result)

	result = lb.Flush(true)
	assert.Empty(t, result)
}

func TestLineBuffer_OnlyNewlines(t *testing.T) {
	lb := internal.NewLineBuffer(100 * time.Millisecond)

	lb.Write("\n\n\n")
	result := lb.Flush(false)
	assert.Equal(t, "\n\n", result)
}

func TestLineBuffer_TrailingNewline(t *testing.T) {
	lb := internal.NewLineBuffer(100 * time.Millisecond)

	lb.Write("content\n")
	result := lb.Flush(false)
	assert.Equal(t, "content", result)

	// Buffer should be empty now
	result = lb.Flush(true)
	assert.Empty(t, result)
}
