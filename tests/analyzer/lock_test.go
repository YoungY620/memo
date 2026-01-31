package analyzer_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/YoungY620/memo/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryLock(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create .memo directory
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	// First lock should succeed
	lock1, err := analyzer.TryLock(memoDir)
	require.NoError(t, err, "First lock should succeed")
	defer analyzer.Unlock(lock1)

	// Verify lock file exists
	lockPath := filepath.Join(memoDir, "watcher.lock")
	data, err := os.ReadFile(lockPath)
	require.NoError(t, err, "Lock file should exist")
	assert.NotEmpty(t, data, "Lock file should contain PID")

	// Second lock should fail
	lock2, err := analyzer.TryLock(memoDir)
	if err == nil {
		analyzer.Unlock(lock2)
		t.Fatal("Second lock should have failed")
	}
	assert.Error(t, err, "Second lock should fail")
	assert.Contains(t, err.Error(), "already running")

	// After unlock, lock should succeed again
	analyzer.Unlock(lock1)
	lock3, err := analyzer.TryLock(memoDir)
	require.NoError(t, err, "Lock after unlock should succeed")
	analyzer.Unlock(lock3)
}

func TestUnlockNil(t *testing.T) {
	// Should not panic
	assert.NotPanics(t, func() {
		analyzer.Unlock(nil)
	})
}

func TestTryLock_DirNotExist(t *testing.T) {
	// Try to lock a non-existent directory
	nonExistentDir := filepath.Join(t.TempDir(), "nonexistent", ".memo")

	lock, err := analyzer.TryLock(nonExistentDir)
	assert.Error(t, err, "Should fail when directory doesn't exist")
	assert.Nil(t, lock)
}

func TestTryLock_PIDWritten(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	lock, err := analyzer.TryLock(memoDir)
	require.NoError(t, err)
	defer analyzer.Unlock(lock)

	// Read and verify PID
	lockPath := filepath.Join(memoDir, "watcher.lock")
	data, err := os.ReadFile(lockPath)
	require.NoError(t, err)

	// Parse PID from file
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	require.NoError(t, err, "Lock file should contain valid PID")

	// PID should be our process
	assert.Equal(t, os.Getpid(), pid, "Lock file should contain current process PID")
}

func TestTryLock_MultipleUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	lock, err := analyzer.TryLock(memoDir)
	require.NoError(t, err)

	// Multiple unlocks should not panic
	assert.NotPanics(t, func() {
		analyzer.Unlock(lock)
		analyzer.Unlock(lock) // Second unlock on same lock
	})
}

func TestTryLock_LockFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	lock, err := analyzer.TryLock(memoDir)
	require.NoError(t, err)
	defer analyzer.Unlock(lock)

	// Verify lock file has correct permissions
	lockPath := filepath.Join(memoDir, "watcher.lock")
	info, err := os.Stat(lockPath)
	require.NoError(t, err)

	// File should be readable and writable
	mode := info.Mode()
	assert.True(t, mode.IsRegular(), "Lock file should be a regular file")
}
