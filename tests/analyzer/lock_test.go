package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YoungY620/memo/analyzer"
)

func TestTryLock(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .memo directory
	memoDir := filepath.Join(tmpDir, ".memo")
	if err := os.MkdirAll(memoDir, 0755); err != nil {
		t.Fatalf("Failed to create .memo dir: %v", err)
	}

	// First lock should succeed
	lock1, err := analyzer.TryLock(memoDir)
	if err != nil {
		t.Fatalf("First lock failed: %v", err)
	}
	defer analyzer.Unlock(lock1)

	// Verify lock file exists
	lockPath := filepath.Join(memoDir, "watcher.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	if len(data) == 0 {
		t.Error("Lock file is empty, expected PID")
	}

	// Second lock should fail
	lock2, err := analyzer.TryLock(memoDir)
	if err == nil {
		analyzer.Unlock(lock2)
		t.Fatal("Second lock should have failed")
	}

	// After unlock, lock should succeed again
	analyzer.Unlock(lock1)
	lock3, err := analyzer.TryLock(memoDir)
	if err != nil {
		t.Fatalf("Lock after unlock failed: %v", err)
	}
	analyzer.Unlock(lock3)
}

func TestUnlockNil(t *testing.T) {
	// Should not panic
	analyzer.Unlock(nil)
}
