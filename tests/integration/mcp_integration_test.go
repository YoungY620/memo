package integration_test

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupMCPTestEnv(t *testing.T) (string, string) {
	t.Helper()

	// Build binary
	binary := buildBinary(t)

	// Create test directory with index
	tmpDir := t.TempDir()
	indexDir := filepath.Join(tmpDir, ".memo", "index")
	_ = os.MkdirAll(indexDir, 0755)

	files := map[string]string{
		"arch.json":      `{"modules": [{"name": "test", "description": "test module", "interfaces": "none"}], "relationships": "test"}`,
		"interface.json": `{"external": [{"type": "cli", "name": "--test", "params": "none", "description": "test"}], "internal": []}`,
		"stories.json":   `{"stories": [{"title": "Test Story", "tags": ["test"], "content": "test content"}]}`,
		"issues.json":    `{"issues": []}`,
	}

	for name, content := range files {
		_ = os.WriteFile(filepath.Join(indexDir, name), []byte(content), 0644)
	}

	return binary, tmpDir
}

func TestMCPServer_Initialize(t *testing.T) {
	binary, tmpDir := setupMCPTestEnv(t)

	// Start MCP server
	cmd := exec.Command(binary, "-mcp", "-path", tmpDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	// Send initialize request
	initReq := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}` + "\n"
	_, _ = stdin.Write([]byte(initReq))

	// Read response
	reader := bufio.NewReader(stdout)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc 2.0, got: %v", resp["jsonrpc"])
	}

	result := resp["result"].(map[string]any)
	if result["protocolVersion"] == nil {
		t.Error("Expected protocolVersion in result")
	}

	serverInfo := result["serverInfo"].(map[string]any)
	if serverInfo["name"] != "memo" {
		t.Errorf("Expected server name 'memo', got: %v", serverInfo["name"])
	}
}

func TestMCPServer_ToolsList(t *testing.T) {
	binary, tmpDir := setupMCPTestEnv(t)

	cmd := exec.Command(binary, "-mcp", "-path", tmpDir)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	_ = cmd.Start()
	defer func() { _ = cmd.Process.Kill() }()

	// Initialize first
	_, _ = stdin.Write([]byte(`{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}` + "\n"))

	reader := bufio.NewReader(stdout)
	_, _ = reader.ReadBytes('\n') // Skip init response

	// Request tools list
	_, _ = stdin.Write([]byte(`{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}` + "\n"))

	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var resp map[string]any
	_ = json.Unmarshal(line, &resp)

	result := resp["result"].(map[string]any)
	tools := result["tools"].([]any)

	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]any)
		toolNames[toolMap["name"].(string)] = true
	}

	if !toolNames["memo_list_keys"] {
		t.Error("Expected memo_list_keys tool")
	}
	if !toolNames["memo_get_value"] {
		t.Error("Expected memo_get_value tool")
	}
}

func TestMCPServer_ToolCall(t *testing.T) {
	binary, tmpDir := setupMCPTestEnv(t)

	cmd := exec.Command(binary, "-mcp", "-path", tmpDir)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	_ = cmd.Start()
	defer func() { _ = cmd.Process.Kill() }()

	reader := bufio.NewReader(stdout)

	// Initialize
	_, _ = stdin.Write([]byte(`{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}` + "\n"))
	_, _ = reader.ReadBytes('\n')

	// Call memo_list_keys
	callReq := `{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "memo_list_keys", "arguments": {"path": "[arch]"}}}` + "\n"
	_, _ = stdin.Write([]byte(callReq))

	line, _ := reader.ReadBytes('\n')

	var resp map[string]any
	_ = json.Unmarshal(line, &resp)

	result := resp["result"].(map[string]any)
	content := result["content"].([]any)

	if len(content) == 0 {
		t.Error("Expected content in result")
	}

	// Parse the content
	contentItem := content[0].(map[string]any)
	text := contentItem["text"].(string)

	var listResult map[string]any
	_ = json.Unmarshal([]byte(text), &listResult)

	if listResult["type"] != "dict" {
		t.Errorf("Expected type 'dict', got: %v", listResult["type"])
	}
}

func TestMCPServer_GetValue(t *testing.T) {
	binary, tmpDir := setupMCPTestEnv(t)

	cmd := exec.Command(binary, "-mcp", "-path", tmpDir)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	_ = cmd.Start()
	defer func() { _ = cmd.Process.Kill() }()

	reader := bufio.NewReader(stdout)

	// Initialize
	_, _ = stdin.Write([]byte(`{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}` + "\n"))
	_, _ = reader.ReadBytes('\n')

	// Get value
	callReq := `{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "memo_get_value", "arguments": {"path": "[arch][modules][0][name]"}}}` + "\n"
	_, _ = stdin.Write([]byte(callReq))

	line, _ := reader.ReadBytes('\n')

	var resp map[string]any
	_ = json.Unmarshal(line, &resp)

	result := resp["result"].(map[string]any)
	content := result["content"].([]any)
	contentItem := content[0].(map[string]any)
	text := contentItem["text"].(string)

	var getValue map[string]any
	_ = json.Unmarshal([]byte(text), &getValue)

	if getValue["value"] != `"test"` {
		t.Errorf("Expected value '\"test\"', got: %v", getValue["value"])
	}
}

func TestMCPServer_InvalidMethod(t *testing.T) {
	binary, tmpDir := setupMCPTestEnv(t)

	cmd := exec.Command(binary, "-mcp", "-path", tmpDir)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	_ = cmd.Start()
	defer func() { _ = cmd.Process.Kill() }()

	reader := bufio.NewReader(stdout)

	// Send invalid method
	_, _ = stdin.Write([]byte(`{"jsonrpc": "2.0", "id": 1, "method": "invalid_method", "params": {}}` + "\n"))

	line, _ := reader.ReadBytes('\n')

	var resp map[string]any
	_ = json.Unmarshal(line, &resp)

	if resp["error"] == nil {
		t.Error("Expected error for invalid method")
	}

	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != -32601 {
		t.Errorf("Expected error code -32601, got: %v", errObj["code"])
	}
}

func TestMCPServer_Shutdown(t *testing.T) {
	binary, tmpDir := setupMCPTestEnv(t)

	cmd := exec.Command(binary, "-mcp", "-path", tmpDir)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	_ = cmd.Start()

	// Close stdin to trigger EOF
	stdin.Close()

	// Server should exit gracefully
	done := make(chan error)
	go func() {
		_, _ = io.ReadAll(stdout)
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("Server exited with: %v (expected)", err)
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Error("Server did not exit after stdin closed")
	}
}

func TestMCPServer_InvalidJSON(t *testing.T) {
	binary, tmpDir := setupMCPTestEnv(t)

	cmd := exec.Command(binary, "-mcp", "-path", tmpDir)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	_ = cmd.Start()
	defer func() { _ = cmd.Process.Kill() }()

	reader := bufio.NewReader(stdout)

	// Send invalid JSON
	_, _ = stdin.Write([]byte(`{invalid json}` + "\n"))

	line, _ := reader.ReadBytes('\n')

	var resp map[string]any
	_ = json.Unmarshal(line, &resp)

	if resp["error"] == nil {
		t.Error("Expected error for invalid JSON")
	}

	errObj := resp["error"].(map[string]any)
	if !strings.Contains(errObj["message"].(string), "Parse error") {
		t.Errorf("Expected parse error, got: %v", errObj["message"])
	}
}
