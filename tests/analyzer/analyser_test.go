//go:build testing

package analyzer_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YoungY620/memo/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests use exported test helpers from analyzer/export_testing.go
// Run with: go test -tags testing ./...

func TestGenerateSessionID(t *testing.T) {
	tests := []struct {
		name    string
		workDir string
		wantLen int // Expected length: "memo-" (5) + 8 hex chars = 13
	}{
		{
			name:    "normal path",
			workDir: "/path/to/project",
			wantLen: 13,
		},
		{
			name:    "root path",
			workDir: "/",
			wantLen: 13,
		},
		{
			name:    "empty path",
			workDir: "",
			wantLen: 13,
		},
		{
			name:    "windows-like path",
			workDir: "C:\\Users\\test\\project",
			wantLen: 13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test determinism - same input should produce same output
			id1 := analyzer.GenerateSessionID(tt.workDir)
			id2 := analyzer.GenerateSessionID(tt.workDir)
			assert.Equal(t, id1, id2, "Session ID should be deterministic")

			// Test prefix
			assert.True(t, strings.HasPrefix(id1, "memo-"), "Session ID should have memo- prefix")

			// Test length
			assert.Equal(t, tt.wantLen, len(id1), "Session ID should have correct length")
		})
	}

	// Test that different paths produce different IDs
	t.Run("different paths produce different IDs", func(t *testing.T) {
		id1 := analyzer.GenerateSessionID("/path/one")
		id2 := analyzer.GenerateSessionID("/path/two")
		assert.NotEqual(t, id1, id2, "Different paths should produce different IDs")
	})
}

func TestToRelativePaths(t *testing.T) {
	workDir := "/home/user/project"

	tests := []struct {
		name     string
		files    []string
		workDir  string
		expected []string
	}{
		{
			name:     "absolute paths",
			files:    []string{"/home/user/project/src/main.go", "/home/user/project/pkg/util.go"},
			workDir:  workDir,
			expected: []string{"src/main.go", "pkg/util.go"},
		},
		{
			name:     "already relative",
			files:    []string{"src/main.go"},
			workDir:  workDir,
			expected: []string{"src/main.go"},
		},
		{
			name:     "empty list",
			files:    []string{},
			workDir:  workDir,
			expected: []string{},
		},
		{
			name:     "mixed paths",
			files:    []string{"/home/user/project/a.go", "b.go"},
			workDir:  workDir,
			expected: []string{"a.go", "b.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.ToRelativePaths(tt.files, tt.workDir)
			assert.Equal(t, len(tt.expected), len(result))
			for i, expected := range tt.expected {
				// Normalize path separators for cross-platform compatibility
				assert.Equal(t, filepath.FromSlash(expected), filepath.FromSlash(result[i]))
			}
		})
	}
}

func TestSplitIntoBatches(t *testing.T) {
	tests := []struct {
		name       string
		files      []string
		threshold  int
		minBatches int
		maxBatches int
		totalFiles int
	}{
		{
			name:       "below threshold - no split",
			files:      []string{"a.go", "b.go", "c.go"},
			threshold:  100,
			minBatches: 1,
			maxBatches: 1,
			totalFiles: 3,
		},
		{
			name:       "exactly at threshold",
			files:      generateFiles("dir", 100),
			threshold:  100,
			minBatches: 1,
			maxBatches: 1,
			totalFiles: 100,
		},
		{
			name:       "above threshold single dir",
			files:      generateFiles("dir", 150),
			threshold:  100,
			minBatches: 1,   // Recursively splits by subdirectory
			maxBatches: 150, // Each file may become its own batch if no subdirs
			totalFiles: 150,
		},
		{
			name:       "above threshold multiple dirs",
			files:      append(generateFiles("dir1", 75), generateFiles("dir2", 75)...),
			threshold:  100,
			minBatches: 2,
			maxBatches: 2,
			totalFiles: 150,
		},
		{
			name:       "empty list",
			files:      []string{},
			threshold:  100,
			minBatches: 1,
			maxBatches: 1,
			totalFiles: 0,
		},
		{
			name: "deep nested structure",
			files: []string{
				"a/b/c/1.go", "a/b/c/2.go",
				"x/y/z/1.go", "x/y/z/2.go",
			},
			threshold:  100,
			minBatches: 1,
			maxBatches: 1,
			totalFiles: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := analyzer.SplitIntoBatches(tt.files, tt.threshold)

			// Verify batch count is within expected range
			assert.GreaterOrEqual(t, len(batches), tt.minBatches, "Should have at least minBatches")
			assert.LessOrEqual(t, len(batches), tt.maxBatches, "Should have at most maxBatches")

			// Verify total file count is preserved
			totalFiles := 0
			for _, batch := range batches {
				totalFiles += len(batch)
			}
			assert.Equal(t, tt.totalFiles, totalFiles, "Total files should be preserved")

			// Verify no empty batches (except for empty input)
			if tt.totalFiles > 0 {
				for i, batch := range batches {
					assert.NotEmpty(t, batch, "Batch %d should not be empty", i)
				}
			}
		})
	}
}

func TestSplitIntoBatches_LargeScale(t *testing.T) {
	// Test with a large number of files spread across directories
	var files []string
	for i := 0; i < 10; i++ {
		files = append(files, generateFiles(filepath.Join("module", string(rune('a'+i))), 50)...)
	}

	batches := analyzer.SplitIntoBatches(files, 100)

	// Should have multiple batches
	assert.Greater(t, len(batches), 1, "Should split into multiple batches")

	// Verify all files are present
	fileSet := make(map[string]bool)
	for _, batch := range batches {
		for _, f := range batch {
			require.False(t, fileSet[f], "File should not be duplicated: %s", f)
			fileSet[f] = true
		}
	}
	assert.Equal(t, 500, len(fileSet), "All 500 files should be present")
}

func TestMergeBatches(t *testing.T) {
	tests := []struct {
		name           string
		batches        [][]string
		capacity       int
		expectedCount  int
		maxBatchSize   int
	}{
		{
			name:          "empty batches",
			batches:       [][]string{},
			capacity:      100,
			expectedCount: 0,
			maxBatchSize:  0,
		},
		{
			name:          "single batch - no merge needed",
			batches:       [][]string{{"a.go", "b.go"}},
			capacity:      100,
			expectedCount: 1,
			maxBatchSize:  100,
		},
		{
			name: "many small batches - merge into one",
			batches: [][]string{
				{"a/1.go", "a/2.go"},
				{"b/1.go", "b/2.go"},
				{"c/1.go", "c/2.go"},
				{"d/1.go", "d/2.go"},
				{"e/1.go", "e/2.go"},
			},
			capacity:      100,
			expectedCount: 1,
			maxBatchSize:  100,
		},
		{
			name: "batches exceed capacity - merge into multiple",
			batches: [][]string{
				generateFiles("dir1", 30),
				generateFiles("dir2", 30),
				generateFiles("dir3", 30),
				generateFiles("dir4", 30),
				generateFiles("dir5", 30),
			},
			capacity:      100,
			expectedCount: 2, // 30+30+30=90, 30+30=60
			maxBatchSize:  100,
		},
		{
			name: "50 tiny batches - merge into one",
			batches: func() [][]string {
				var b [][]string
				for i := 0; i < 50; i++ {
					b = append(b, []string{fmt.Sprintf("dir%d/file.go", i), fmt.Sprintf("dir%d/file2.go", i)})
				}
				return b
			}(),
			capacity:      100,
			expectedCount: 1,
			maxBatchSize:  100,
		},
		{
			name: "mixed sizes - FFD optimization",
			batches: [][]string{
				generateFiles("big1", 60),
				generateFiles("big2", 60),
				generateFiles("small1", 20),
				generateFiles("small2", 20),
				generateFiles("small3", 20),
				generateFiles("small4", 20),
			},
			capacity:      100,
			expectedCount: 2, // FFD: 60+20+20=100, 60+20+20=100
			maxBatchSize:  100,
		},
		{
			name: "batch at capacity - no merge possible",
			batches: [][]string{
				generateFiles("dir1", 100),
				generateFiles("dir2", 100),
			},
			capacity:      100,
			expectedCount: 2,
			maxBatchSize:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := analyzer.MergeBatches(tt.batches, tt.capacity)

			// Verify batch count
			assert.Equal(t, tt.expectedCount, len(merged), "Should have expected number of batches")

			// Verify no batch exceeds capacity
			for i, batch := range merged {
				assert.LessOrEqual(t, len(batch), tt.maxBatchSize, "Batch %d should not exceed capacity", i)
			}

			// Verify all files are preserved
			originalFiles := make(map[string]bool)
			for _, batch := range tt.batches {
				for _, f := range batch {
					originalFiles[f] = true
				}
			}

			mergedFiles := make(map[string]bool)
			for _, batch := range merged {
				for _, f := range batch {
					assert.False(t, mergedFiles[f], "File should not be duplicated: %s", f)
					mergedFiles[f] = true
				}
			}

			assert.Equal(t, len(originalFiles), len(mergedFiles), "All files should be preserved")
		})
	}
}

func TestSplitAndMerge_Integration(t *testing.T) {
	// Test the full pipeline: split then merge
	// 50 directories with 3 files each = 150 files total (exceeds threshold of 100)
	var files []string
	for i := 0; i < 50; i++ {
		dir := fmt.Sprintf("dir%d", i)
		files = append(files, filepath.Join(dir, "a.go"), filepath.Join(dir, "b.go"), filepath.Join(dir, "c.go"))
	}

	// Split will create 50 small batches (each dir has 3 files < threshold)
	batches := analyzer.SplitIntoBatches(files, 100)
	assert.Equal(t, 50, len(batches), "Split should create 50 batches")

	// Merge should combine them into 2 batches (150 files, capacity 100)
	merged := analyzer.MergeBatches(batches, 100)
	assert.Equal(t, 2, len(merged), "Merge should combine into 2 batches")

	// Verify all 150 files preserved
	totalFiles := 0
	for _, batch := range merged {
		totalFiles += len(batch)
	}
	assert.Equal(t, 150, totalFiles, "All 150 files should be preserved")
}

func BenchmarkMergeBatches(b *testing.B) {
	// Create 100 small batches of 5 files each
	var batches [][]string
	for i := 0; i < 100; i++ {
		batches = append(batches, generateFiles(fmt.Sprintf("dir%d", i), 5))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.MergeBatches(batches, 100)
	}
}

func TestLoadPrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		wantNil bool
	}{
		{
			name:    "analyse prompt",
			prompt:  "analyse",
			wantNil: false,
		},
		{
			name:    "context prompt",
			prompt:  "context",
			wantNil: false,
		},
		{
			name:    "feedback prompt",
			prompt:  "feedback",
			wantNil: false,
		},
		{
			name:    "nonexistent prompt",
			prompt:  "nonexistent",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.LoadPrompt(tt.prompt)
			if tt.wantNil {
				assert.Empty(t, result, "Nonexistent prompt should return empty string")
			} else {
				assert.NotEmpty(t, result, "Existing prompt should return content")
			}
		})
	}
}

// Helper function to generate file paths
func generateFiles(dir string, count int) []string {
	files := make([]string, count)
	for i := 0; i < count; i++ {
		files[i] = filepath.Join(dir, fmt.Sprintf("file%d.go", i))
	}
	return files
}

// Benchmark tests
func BenchmarkSplitIntoBatches_Small(b *testing.B) {
	files := generateFiles("dir", 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.SplitIntoBatches(files, 100)
	}
}

func BenchmarkSplitIntoBatches_Large(b *testing.B) {
	var files []string
	for i := 0; i < 20; i++ {
		files = append(files, generateFiles(filepath.Join("module", string(rune('a'+i))), 100)...)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.SplitIntoBatches(files, 100)
	}
}

func BenchmarkGenerateSessionID(b *testing.B) {
	workDir := "/path/to/some/project/directory"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.GenerateSessionID(workDir)
	}
}
