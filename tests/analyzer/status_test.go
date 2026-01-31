package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YoungY620/memo/analyzer"
)

func TestSetStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .memo directory
	memoDir := filepath.Join(tmpDir, ".memo")
	if err := os.MkdirAll(memoDir, 0755); err != nil {
		t.Fatalf("Failed to create .memo dir: %v", err)
	}

	// Test setting idle status
	if err := analyzer.SetStatus(memoDir, "idle"); err != nil {
		t.Fatalf("SetStatus(idle) failed: %v", err)
	}

	// Verify file exists
	statusPath := filepath.Join(memoDir, "status.json")
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		t.Fatal("Status file was not created")
	}

	// Test setting analyzing status
	if err := analyzer.SetStatus(memoDir, "analyzing"); err != nil {
		t.Fatalf("SetStatus(analyzing) failed: %v", err)
	}

	status := analyzer.GetStatus(memoDir)
	if status.Status != "analyzing" {
		t.Errorf("Expected status 'analyzing', got '%s'", status.Status)
	}
	if status.Since == nil {
		t.Error("Expected non-nil Since for analyzing status")
	} else {
		// Since should be recent (within last 5 seconds)
		if time.Since(*status.Since) > 5*time.Second {
			t.Error("Since timestamp is too old")
		}
	}
}

func TestGetStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	memoDir := filepath.Join(tmpDir, ".memo")
	if err := os.MkdirAll(memoDir, 0755); err != nil {
		t.Fatalf("Failed to create .memo dir: %v", err)
	}

	// Test getting status when file doesn't exist
	status := analyzer.GetStatus(memoDir)
	if status.Status != "idle" {
		t.Errorf("Expected default 'idle' status, got '%s'", status.Status)
	}

	// Test getting status after setting
	analyzer.SetStatus(memoDir, "analyzing")
	status = analyzer.GetStatus(memoDir)
	if status.Status != "analyzing" {
		t.Errorf("Expected 'analyzing' status, got '%s'", status.Status)
	}

	// Test getting status with invalid JSON
	statusPath := filepath.Join(memoDir, "status.json")
	os.WriteFile(statusPath, []byte("invalid json"), 0644)
	status = analyzer.GetStatus(memoDir)
	if status.Status != "idle" {
		t.Errorf("Expected 'idle' for invalid JSON, got '%s'", status.Status)
	}
}
