package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YoungY620/memo/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetStatus(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	// Test setting idle status
	err := analyzer.SetStatus(memoDir, "idle")
	require.NoError(t, err)

	// Verify file exists
	statusPath := filepath.Join(memoDir, "status.json")
	_, err = os.Stat(statusPath)
	assert.NoError(t, err, "Status file should be created")

	// Test setting analyzing status
	err = analyzer.SetStatus(memoDir, "analyzing")
	require.NoError(t, err)

	status := analyzer.GetStatus(memoDir)
	assert.Equal(t, "analyzing", status.Status)
	assert.NotNil(t, status.Since, "Analyzing status should have Since timestamp")
	assert.WithinDuration(t, time.Now(), *status.Since, 5*time.Second)
}

func TestGetStatus(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	// Test getting status when file doesn't exist
	status := analyzer.GetStatus(memoDir)
	assert.Equal(t, "idle", status.Status, "Default status should be idle")

	// Test getting status after setting
	require.NoError(t, analyzer.SetStatus(memoDir, "analyzing"))
	status = analyzer.GetStatus(memoDir)
	assert.Equal(t, "analyzing", status.Status)

	// Test getting status with invalid JSON
	statusPath := filepath.Join(memoDir, "status.json")
	require.NoError(t, os.WriteFile(statusPath, []byte("invalid json"), 0644))
	status = analyzer.GetStatus(memoDir)
	assert.Equal(t, "idle", status.Status, "Invalid JSON should fallback to idle")
}

func TestSetStatus_DirNotExist(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "nonexistent", ".memo")

	err := analyzer.SetStatus(nonExistentDir, "idle")
	assert.Error(t, err, "Should fail when directory doesn't exist")
}

func TestSetStatus_IdleHasNoSince(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	// First set analyzing (has Since)
	require.NoError(t, analyzer.SetStatus(memoDir, "analyzing"))
	status := analyzer.GetStatus(memoDir)
	assert.NotNil(t, status.Since)

	// Then set idle (should not have Since)
	require.NoError(t, analyzer.SetStatus(memoDir, "idle"))
	status = analyzer.GetStatus(memoDir)
	assert.Equal(t, "idle", status.Status)
	assert.Nil(t, status.Since, "Idle status should not have Since timestamp")
}

func TestGetStatus_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	require.NoError(t, os.MkdirAll(memoDir, 0755))

	// Write empty file
	statusPath := filepath.Join(memoDir, "status.json")
	require.NoError(t, os.WriteFile(statusPath, []byte(""), 0644))

	status := analyzer.GetStatus(memoDir)
	assert.Equal(t, "idle", status.Status, "Empty file should fallback to idle")
}
