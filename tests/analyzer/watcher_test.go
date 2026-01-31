package analyzer_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/YoungY620/memo/analyzer"
)

func TestNewWatcher(t *testing.T) {
	tmpDir := t.TempDir()

	onChange := func(files []string) {
		// Callback for testing
		_ = files
	}

	watcher, err := analyzer.NewWatcher(tmpDir, []string{".git", "node_modules"}, 100, 1000, onChange)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	if watcher == nil {
		t.Error("Expected non-nil watcher")
	}
}

func TestWatcher_ScanAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	testFile1 := filepath.Join(tmpDir, "file1.txt")
	testFile2 := filepath.Join(tmpDir, "file2.txt")
	os.WriteFile(testFile1, []byte("content1"), 0644)
	os.WriteFile(testFile2, []byte("content2"), 0644)

	// Create ignored directory
	ignoredDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(ignoredDir, 0755)
	os.WriteFile(filepath.Join(ignoredDir, "ignored.txt"), []byte("ignored"), 0644)

	var mu sync.Mutex
	var receivedFiles []string
	onChange := func(files []string) {
		mu.Lock()
		receivedFiles = files
		mu.Unlock()
	}

	watcher, err := analyzer.NewWatcher(tmpDir, []string{".git"}, 50, 200, onChange)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Scan all files
	watcher.ScanAll()

	// Wait for debounce
	time.Sleep(100 * time.Millisecond)
	watcher.Flush()

	mu.Lock()
	defer mu.Unlock()

	if len(receivedFiles) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(receivedFiles), receivedFiles)
	}

	// Verify ignored files are not included
	for _, f := range receivedFiles {
		if filepath.Base(filepath.Dir(f)) == ".git" {
			t.Errorf("Ignored file should not be included: %s", f)
		}
	}
}

func TestWatcher_IgnorePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with various patterns
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file.log"), []byte("log"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "node_modules", "pkg.js"), []byte("js"), 0644)

	var mu sync.Mutex
	var receivedFiles []string
	onChange := func(files []string) {
		mu.Lock()
		receivedFiles = files
		mu.Unlock()
	}

	watcher, err := analyzer.NewWatcher(tmpDir, []string{"*.log", "node_modules"}, 50, 200, onChange)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	watcher.ScanAll()
	time.Sleep(100 * time.Millisecond)
	watcher.Flush()

	mu.Lock()
	defer mu.Unlock()

	// Should only have file.txt
	if len(receivedFiles) != 1 {
		t.Errorf("Expected 1 file, got %d: %v", len(receivedFiles), receivedFiles)
	}

	for _, f := range receivedFiles {
		if filepath.Ext(f) == ".log" {
			t.Errorf("Log file should be ignored: %s", f)
		}
	}
}

func TestWatcher_Close(t *testing.T) {
	tmpDir := t.TempDir()

	watcher, err := analyzer.NewWatcher(tmpDir, nil, 100, 1000, func(files []string) {})
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Should not error
	if err := watcher.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
