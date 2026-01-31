package internal_test

import (
	"testing"
	"time"

	"github.com/YoungY620/memo/internal"
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
			internal.SetLogLevel(tc.level)
		})
	}
}

func TestLogFunctions(t *testing.T) {
	// These should not panic at any log level
	internal.SetLogLevel("debug")

	internal.LogError("test error %s", "arg")
	internal.LogNotice("test notice %s", "arg")
	internal.LogInfo("test info %s", "arg")
	internal.LogDebug("test debug %s", "arg")

	internal.SetLogLevel("error")
	internal.LogDebug("this should be suppressed")
}

func TestLineBuffer(t *testing.T) {
	lb := internal.NewLineBuffer(100 * time.Millisecond)

	// Test writing and flushing complete lines
	lb.Write("line1\nline2\n")
	result := lb.Flush(false)
	if result != "line1\nline2" {
		t.Errorf("Expected 'line1\\nline2', got '%s'", result)
	}

	// Test partial line (should not flush without force)
	lb.Write("partial")
	result = lb.Flush(false)
	if result != "" {
		t.Errorf("Expected empty string for partial line, got '%s'", result)
	}

	// Test force flush
	result = lb.Flush(true)
	if result != "partial" {
		t.Errorf("Expected 'partial', got '%s'", result)
	}

	// Buffer should be empty now
	result = lb.Flush(true)
	if result != "" {
		t.Errorf("Expected empty string after flush, got '%s'", result)
	}
}

func TestLineBuffer_Timeout(t *testing.T) {
	lb := internal.NewLineBuffer(50 * time.Millisecond)

	lb.Write("partial")

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	result := lb.Flush(false)
	if result != "partial" {
		t.Errorf("Expected 'partial' after timeout, got '%s'", result)
	}
}

func TestLineBuffer_MixedContent(t *testing.T) {
	lb := internal.NewLineBuffer(100 * time.Millisecond)

	lb.Write("line1\n")
	lb.Write("line2\npartial")

	result := lb.Flush(false)
	if result != "line1\nline2" {
		t.Errorf("Expected 'line1\\nline2', got '%s'", result)
	}

	// Partial should still be in buffer
	result = lb.Flush(true)
	if result != "partial" {
		t.Errorf("Expected 'partial', got '%s'", result)
	}
}
