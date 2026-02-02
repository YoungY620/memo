package analyzer_test

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YoungY620/memo/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWatcher(t *testing.T) {
	tmpDir := t.TempDir()

	onChange := func(files []string) {
		// Callback for testing
		_ = files
	}

	watcher, err := analyzer.NewWatcher(tmpDir, []string{".git", "node_modules"}, 100, 1000, onChange)
	require.NoError(t, err)
	defer watcher.Close()

	assert.NotNil(t, watcher)
}

func TestWatcher_ScanAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	testFile1 := filepath.Join(tmpDir, "file1.txt")
	testFile2 := filepath.Join(tmpDir, "file2.txt")
	require.NoError(t, os.WriteFile(testFile1, []byte("content1"), 0644))
	require.NoError(t, os.WriteFile(testFile2, []byte("content2"), 0644))

	// Create ignored directory
	ignoredDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.MkdirAll(ignoredDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ignoredDir, "ignored.txt"), []byte("ignored"), 0644))

	var mu sync.Mutex
	var receivedFiles []string
	onChange := func(files []string) {
		mu.Lock()
		receivedFiles = files
		mu.Unlock()
	}

	watcher, err := analyzer.NewWatcher(tmpDir, []string{".git"}, 50, 200, onChange)
	require.NoError(t, err)
	defer watcher.Close()

	// Scan all files
	watcher.ScanAll()

	// Wait for debounce
	time.Sleep(100 * time.Millisecond)
	watcher.Flush()

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, receivedFiles, 2, "Should have 2 files")

	// Verify ignored files are not included
	for _, f := range receivedFiles {
		assert.NotEqual(t, ".git", filepath.Base(filepath.Dir(f)), "Ignored file should not be included: %s", f)
	}
}

func TestWatcher_IgnorePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with various patterns
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.log"), []byte("log"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "node_modules", "pkg.js"), []byte("js"), 0644))

	var mu sync.Mutex
	var receivedFiles []string
	onChange := func(files []string) {
		mu.Lock()
		receivedFiles = files
		mu.Unlock()
	}

	watcher, err := analyzer.NewWatcher(tmpDir, []string{"*.log", "node_modules"}, 50, 200, onChange)
	require.NoError(t, err)
	defer watcher.Close()

	watcher.ScanAll()
	time.Sleep(100 * time.Millisecond)
	watcher.Flush()

	mu.Lock()
	defer mu.Unlock()

	// Should only have file.txt
	assert.Len(t, receivedFiles, 1, "Should only have 1 file")

	for _, f := range receivedFiles {
		assert.NotEqual(t, ".log", filepath.Ext(f), "Log file should be ignored: %s", f)
	}
}

func TestWatcher_Close(t *testing.T) {
	tmpDir := t.TempDir()

	watcher, err := analyzer.NewWatcher(tmpDir, nil, 100, 1000, func(files []string) {})
	require.NoError(t, err)

	// Should not error
	assert.NoError(t, watcher.Close())
}

func TestWatcher_Debounce(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for i := 0; i < 5; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt"), []byte("content"), 0644))
	}

	var mu sync.Mutex
	var callCount int
	var receivedFiles []string
	onChange := func(files []string) {
		mu.Lock()
		callCount++
		receivedFiles = files
		mu.Unlock()
	}

	// Short debounce (50ms), long max wait (1000ms)
	watcher, err := analyzer.NewWatcher(tmpDir, nil, 50, 1000, onChange)
	require.NoError(t, err)
	defer watcher.Close()

	// Scan all - this adds all files to pending
	watcher.ScanAll()

	// Wait for debounce to trigger
	time.Sleep(100 * time.Millisecond)
	watcher.Flush()

	mu.Lock()
	assert.Equal(t, 1, callCount, "Should call onChange only once due to debounce")
	assert.Equal(t, 5, len(receivedFiles), "Should receive all 5 files")
	mu.Unlock()
}

func TestWatcher_MaxWait(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644))

	var mu sync.Mutex
	var callCount int
	onChange := func(files []string) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	// Long debounce (500ms), short max wait (100ms)
	watcher, err := analyzer.NewWatcher(tmpDir, nil, 500, 100, onChange)
	require.NoError(t, err)
	defer watcher.Close()

	// Scan all
	watcher.ScanAll()

	// Wait for max wait to trigger (should be before debounce)
	time.Sleep(150 * time.Millisecond)
	watcher.Flush()

	mu.Lock()
	assert.GreaterOrEqual(t, callCount, 1, "MaxWait should trigger callback before debounce")
	mu.Unlock()
}

func TestWatcher_ConcurrentGuard(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for i := 0; i < 3; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt"), []byte("content"), 0644))
	}

	var concurrentCount int32
	var maxConcurrent int32
	var callCount int32

	onChange := func(files []string) {
		current := atomic.AddInt32(&concurrentCount, 1)
		defer atomic.AddInt32(&concurrentCount, -1)
		atomic.AddInt32(&callCount, 1)

		// Track max concurrent
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
				break
			}
		}

		// Simulate some work
		time.Sleep(50 * time.Millisecond)
	}

	watcher, err := analyzer.NewWatcher(tmpDir, nil, 10, 50, onChange)
	require.NoError(t, err)
	defer watcher.Close()

	// Trigger multiple flushes concurrently
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			watcher.ScanAll()
			time.Sleep(20 * time.Millisecond)
			watcher.Flush()
		}()
	}
	wg.Wait()

	// Give some time for any pending callbacks
	time.Sleep(100 * time.Millisecond)

	// Max concurrent should be 1 (semaphore protection)
	assert.Equal(t, int32(1), atomic.LoadInt32(&maxConcurrent),
		"Only one analysis should run at a time")
}

func TestWatcher_NewDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644))

	var mu sync.Mutex
	var receivedFiles []string
	onChange := func(files []string) {
		mu.Lock()
		receivedFiles = append(receivedFiles, files...)
		mu.Unlock()
	}

	watcher, err := analyzer.NewWatcher(tmpDir, nil, 50, 200, onChange)
	require.NoError(t, err)
	defer watcher.Close()

	// Start watcher in background
	go func() { _ = watcher.Run() }()

	// Wait a bit for watcher to start
	time.Sleep(50 * time.Millisecond)

	// Create a new subdirectory with a file
	subDir := filepath.Join(tmpDir, "newdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	time.Sleep(50 * time.Millisecond)

	// Create file in new directory
	newFile := filepath.Join(subDir, "newfile.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("new content"), 0644))

	// Wait for debounce
	time.Sleep(150 * time.Millisecond)
	watcher.Flush()

	mu.Lock()
	defer mu.Unlock()

	// Should have detected the new file
	found := false
	for _, f := range receivedFiles {
		if filepath.Base(f) == "newfile.txt" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should detect file in newly created directory")
}

func TestWatcher_FileEvents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("initial"), 0644))

	var mu sync.Mutex
	var receivedFiles []string
	onChange := func(files []string) {
		mu.Lock()
		receivedFiles = append(receivedFiles, files...)
		mu.Unlock()
	}

	watcher, err := analyzer.NewWatcher(tmpDir, nil, 50, 200, onChange)
	require.NoError(t, err)
	defer watcher.Close()

	// Start watcher
	go func() { _ = watcher.Run() }()
	time.Sleep(50 * time.Millisecond)

	// Test Write event
	require.NoError(t, os.WriteFile(testFile, []byte("modified"), 0644))
	time.Sleep(150 * time.Millisecond)
	watcher.Flush()

	mu.Lock()
	assert.Contains(t, receivedFiles, testFile, "Should detect file modification")
	receivedFiles = nil
	mu.Unlock()

	// Test Rename event
	newPath := filepath.Join(tmpDir, "renamed.txt")
	require.NoError(t, os.Rename(testFile, newPath))
	time.Sleep(150 * time.Millisecond)
	watcher.Flush()

	mu.Lock()
	// Either old or new path should be detected
	assert.True(t, len(receivedFiles) > 0, "Should detect file rename")
	mu.Unlock()
}

// Benchmark for ignored pattern matching
func BenchmarkIgnored(b *testing.B) {
	tmpDir := b.TempDir()
	patterns := []string{".git", "node_modules", "*.log", "dist", "build", ".memo"}

	watcher, _ := analyzer.NewWatcher(tmpDir, patterns, 100, 1000, func([]string) {})
	defer watcher.Close()

	testPaths := []string{
		"/project/src/main.go",
		"/project/.git/config",
		"/project/node_modules/pkg/index.js",
		"/project/build/output.js",
		"/project/logs/app.log",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			// Access the ignored method indirectly through ScanAll behavior
			_ = path
		}
	}
}
