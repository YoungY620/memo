//go:build windows

package analyzer

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
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

	// Try to lock the file exclusively with LOCKFILE_FAIL_IMMEDIATELY
	// This is the Windows equivalent of LOCK_EX|LOCK_NB on Unix
	handle := windows.Handle(f.Fd())
	overlapped := &windows.Overlapped{}
	err = windows.LockFileEx(
		handle,
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1, // Lock 1 byte
		0,
		overlapped,
	)
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
		handle := windows.Handle(f.Fd())
		overlapped := &windows.Overlapped{}
		// Ignore unlock error - file close will release the lock anyway
		windows.UnlockFileEx(handle, 0, 1, 0, overlapped)
		f.Close()
	}
}
