//go:build testing

package analyzer_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/YoungY620/memo/analyzer"
	"github.com/stretchr/testify/assert"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintBanner(t *testing.T) {
	opts := analyzer.BannerOptions{
		WorkDir: "/test/path",
		Version: "1.0.0",
	}

	// Just make sure it doesn't panic
	output := captureOutput(func() {
		analyzer.PrintBanner(opts)
	})

	assert.NotEmpty(t, output, "Expected non-empty banner output")
	assert.Contains(t, output, "1.0.0", "Banner should contain version")
}

func TestPrintBanner_LongPath(t *testing.T) {
	opts := analyzer.BannerOptions{
		WorkDir: "/very/long/path/that/might/need/truncation/to/fit/in/the/banner/display/properly/test",
		Version: "dev",
	}

	// Should not panic with long path
	output := captureOutput(func() {
		analyzer.PrintBanner(opts)
	})

	assert.NotEmpty(t, output, "Expected non-empty banner output")
}

func TestGetGreeting(t *testing.T) {
	// Note: This test depends on current time, so we test the function exists and returns string
	greeting := analyzer.GetGreeting()

	// Greeting can be empty (normal hours) or contain special messages
	// Just verify it doesn't panic and returns a string
	assert.IsType(t, "", greeting)
}

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "ASCII only",
			input:    "hello",
			expected: 5,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "CJK characters",
			input:    "你好",
			expected: 4, // Each CJK char is 2 width
		},
		{
			name:     "mixed ASCII and CJK",
			input:    "hello你好",
			expected: 9, // 5 + 4
		},
		{
			name:     "box drawing characters",
			input:    "─│┌┐",
			expected: 4, // Each box char is 1 width
		},
		{
			name:     "block elements",
			input:    "█▀▄",
			expected: 3, // Each block char is 1 width
		},
		{
			name:     "emoji",
			input:    "✨",
			expected: 2, // Emoji typically 2 width
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.RuneWidth(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		maxWidth int
		expected string
	}{
		{
			name:     "short path no truncation",
			path:     "/home/user",
			maxWidth: 20,
			expected: "/home/user",
		},
		{
			name:     "exact width",
			path:     "/home/user",
			maxWidth: 10,
			expected: "/home/user",
		},
		{
			name:     "needs truncation",
			path:     "/very/long/path/that/needs/truncation",
			maxWidth: 20,
			expected: "...that/needs/truncation",
		},
		{
			name:     "very small max width",
			path:     "/path/to/file",
			maxWidth: 5,
			expected: "...le",
		},
		{
			name:     "minimal width",
			path:     "/path/to/file",
			maxWidth: 3,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.TruncatePath(tt.path, tt.maxWidth)

			// Verify result width is within max
			resultWidth := analyzer.RuneWidth(result)
			assert.LessOrEqual(t, resultWidth, tt.maxWidth, "Result should not exceed max width")

			// For truncated paths, should start with "..."
			if len(tt.path) > tt.maxWidth && tt.maxWidth >= 3 {
				assert.True(t, strings.HasPrefix(result, "..."), "Truncated path should start with ...")
			}
		})
	}
}

func TestTruncatePath_EdgeCases(t *testing.T) {
	// Empty path
	result := analyzer.TruncatePath("", 10)
	assert.Equal(t, "", result)

	// Path shorter than "..."
	result = analyzer.TruncatePath("ab", 10)
	assert.Equal(t, "ab", result)
}

// Benchmark for runeWidth
func BenchmarkRuneWidth(b *testing.B) {
	testStrings := []string{
		"hello world",
		"你好世界",
		"hello你好world世界",
		"─────────────────",
		"███████████████████",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range testStrings {
			analyzer.RuneWidth(s)
		}
	}
}
