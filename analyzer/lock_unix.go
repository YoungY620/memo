//go:build unix

package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const lockFileName = "watcher.lock"

// TryLock attempts to acquire an exclusive lock on .memo/watcher.lock
// Returns the lock file handle if successful, nil and error if already locked
func TryLock(memoDir string) (*os.File, error) {
	lockPath := filepath.Join(memoDir, lockFileName)

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try non-blocking exclusive lock
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("another watcher is already running on this directory")
	}

	// Write PID to lock file (for debugging)
	f.Truncate(0)
	f.Seek(0, 0)
	fmt.Fprintf(f, "%d\n", os.Getpid())
	f.Sync()

	return f, nil
}

// Unlock releases the lock and closes the file
func Unlock(f *os.File) {
	if f != nil {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}
}
