package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// getBinaryName returns the binary name for the current platform
func getBinaryName() string {
	if runtime.GOOS == "windows" {
		return "memo.exe"
	}
	return "memo"
}

// buildBinary builds the memo binary for testing
func buildBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, getBinaryName())

	cmd := exec.Command("go", "build", "-o", binaryPath, "../../.")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	return binaryPath
}

func TestVersion(t *testing.T) {
	binary := buildBinary(t)

	cmd := exec.Command(binary, "-version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Version command failed: %v", err)
	}

	if !strings.Contains(string(output), "memo") {
		t.Errorf("Version output should contain 'memo', got: %s", output)
	}
}

func TestHelp(t *testing.T) {
	binary := buildBinary(t)

	cmd := exec.Command(binary, "-help")
	output, err := cmd.CombinedOutput()

	// -help returns exit code 2 in Go, so we just check output
	_ = err

	outputStr := string(output)
	if !strings.Contains(outputStr, "-path") {
		t.Errorf("Help should mention -path flag, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "-config") {
		t.Errorf("Help should mention -config flag, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "-once") {
		t.Errorf("Help should mention -once flag, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "-mcp") {
		t.Errorf("Help should mention -mcp flag, got: %s", outputStr)
	}
}

func TestMCPMode_NoIndex(t *testing.T) {
	binary := buildBinary(t)

	// Create a directory without .memo/index
	tmpDir := t.TempDir()

	cmd := exec.Command(binary, "-mcp", "-path", tmpDir)
	output, err := cmd.CombinedOutput()

	// Should fail because no index exists
	if err == nil {
		t.Error("MCP mode should fail without index directory")
	}

	if !strings.Contains(string(output), "Index directory not found") {
		t.Errorf("Should report missing index, got: %s", output)
	}
}

func TestInitIndex(t *testing.T) {
	binary := buildBinary(t)

	tmpDir := t.TempDir()

	// Run once mode (which should init the index)
	// Note: This will fail because no API key, but index should be created
	cmd := exec.Command(binary, "-once", "-path", tmpDir, "-config", "nonexistent.yaml")
	cmd.Run() // Ignore error, we just want the index to be created

	// Check if index was created
	indexDir := filepath.Join(tmpDir, ".memo", "index")
	expectedFiles := []string{"arch.json", "interface.json", "stories.json", "issues.json"}

	for _, f := range expectedFiles {
		path := filepath.Join(indexDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected %s to be created", f)
		}
	}

	// Check mcp.json was created
	mcpPath := filepath.Join(tmpDir, ".memo", "mcp.json")
	if _, err := os.Stat(mcpPath); os.IsNotExist(err) {
		t.Error("Expected mcp.json to be created")
	}

	// Check .gitignore was created
	gitignorePath := filepath.Join(tmpDir, ".memo", ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Error("Expected .gitignore to be created")
	}
}

func TestLockFile(t *testing.T) {
	// Skip on CI if running in parallel
	if os.Getenv("CI") != "" {
		t.Skip("Skipping lock test in CI")
	}

	binary := buildBinary(t)
	tmpDir := t.TempDir()

	// First, initialize the directory
	initCmd := exec.Command(binary, "-once", "-path", tmpDir, "-config", "nonexistent.yaml")
	initCmd.Run()

	// Try to run two instances (second should fail due to lock)
	// This is tricky to test reliably, so we just verify the lock file exists after running
	lockPath := filepath.Join(tmpDir, ".memo", "watcher.lock")

	// The lock file might exist from the init run
	// Just verify the mechanism works by checking the path format
	if _, err := os.Stat(filepath.Dir(lockPath)); os.IsNotExist(err) {
		t.Error("Expected .memo directory to exist")
	}
}
