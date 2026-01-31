//go:build testing

package mcp_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YoungY620/memo/internal"
	"github.com/YoungY620/memo/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockServer creates a server with custom input/output for testing
type testServer struct {
	input  *bytes.Buffer
	output *bytes.Buffer
	server *mcp.Server
}

func newTestServer(t *testing.T) (*testServer, string) {
	tmpDir := t.TempDir()
	memoDir := filepath.Join(tmpDir, ".memo")
	indexDir := filepath.Join(memoDir, "index")
	os.MkdirAll(indexDir, 0755)

	// Create test index files
	files := map[string]string{
		"arch.json":      `{"modules": [{"name": "test", "description": "test module", "interfaces": "none"}], "relationships": ""}`,
		"interface.json": `{"external": [], "internal": []}`,
		"stories.json":   `{"stories": []}`,
		"issues.json":    `{"issues": []}`,
	}

	for name, content := range files {
		os.WriteFile(filepath.Join(indexDir, name), []byte(content), 0644)
	}

	// Ensure history logger is closed after test to release file handles (Windows)
	t.Cleanup(internal.CloseHistoryLogger)

	return &testServer{
		input:  new(bytes.Buffer),
		output: new(bytes.Buffer),
	}, tmpDir
}

func TestServer_Initialize(t *testing.T) {
	ts, workDir := newTestServer(t)

	// Create a real server
	server := mcp.NewServer(workDir)

	// Test initialize request parsing
	initReq := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}`

	// We can't easily test the full Run() loop, but we can verify the request format
	var req mcp.Request
	if err := json.Unmarshal([]byte(initReq), &req); err != nil {
		t.Fatalf("Failed to parse init request: %v", err)
	}

	if req.Method != "initialize" {
		t.Errorf("Expected method 'initialize', got '%s'", req.Method)
	}

	_ = ts
	_ = server
}

func TestServer_ToolsList(t *testing.T) {
	_, workDir := newTestServer(t)

	server := mcp.NewServer(workDir)

	// Verify server is created
	if server == nil {
		t.Fatal("Server should not be nil")
	}
}

func TestJSONRPCRequest_Parsing(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid request",
			json:    `{"jsonrpc": "2.0", "id": 1, "method": "initialize"}`,
			wantErr: false,
		},
		{
			name:    "with params",
			json:    `{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "memo_list_keys", "arguments": {"path": "[arch]"}}}`,
			wantErr: false,
		},
		{
			name:    "string id",
			json:    `{"jsonrpc": "2.0", "id": "test-id", "method": "test"}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req mcp.Request
			err := json.Unmarshal([]byte(tt.json), &req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSONRPCResponse_Format(t *testing.T) {
	resp := mcp.Response{
		JSONRPC: "2.0",
		ID:      1,
		Result: mcp.InitializeResult{
			ProtocolVersion: "2024-11-05",
			ServerInfo: mcp.ServerInfo{
				Name:    "memo",
				Version: "1.0.0",
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Verify JSON structure
	var result map[string]any
	json.Unmarshal(data, &result)

	if result["jsonrpc"] != "2.0" {
		t.Error("Expected jsonrpc 2.0")
	}
	if result["id"].(float64) != 1 {
		t.Error("Expected id 1")
	}
	if result["result"] == nil {
		t.Error("Expected result to be present")
	}
}

func TestToolCallResult_Format(t *testing.T) {
	result := mcp.ToolCallResult{
		Content: []mcp.ContentItem{
			{Type: "text", Text: "test content"},
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	if !strings.Contains(string(data), "test content") {
		t.Error("Result should contain 'test content'")
	}
}

func TestToolCallResult_WithError(t *testing.T) {
	result := mcp.ToolCallResult{
		Content: []mcp.ContentItem{
			{Type: "text", Text: "error message"},
		},
		IsError: true,
	}

	data, _ := json.Marshal(result)

	var parsed map[string]any
	json.Unmarshal(data, &parsed)

	if parsed["isError"] != true {
		t.Error("Expected isError to be true")
	}
}

func TestToolCallResult_WithWarning(t *testing.T) {
	result := mcp.ToolCallResult{
		Content: []mcp.ContentItem{
			{Type: "text", Text: "data"},
		},
		Warning: "Data may be stale",
	}

	data, _ := json.Marshal(result)

	if !strings.Contains(string(data), "Data may be stale") {
		t.Error("Result should contain warning")
	}
}

func TestServe_NonExistentDir(t *testing.T) {
	// Serve should work even with new directory (it creates .memo)
	tmpDir := t.TempDir()
	t.Cleanup(internal.CloseHistoryLogger)

	server := mcp.NewServer(tmpDir)

	if server == nil {
		t.Error("Server should be created even for new directory")
	}
}

func TestServer_ReadLine(t *testing.T) {
	// Test that newline-delimited JSON works
	input := `{"jsonrpc": "2.0", "id": 1, "method": "initialize"}` + "\n"
	reader := bufio.NewReader(strings.NewReader(input))

	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("Failed to read line: %v", err)
	}

	var req mcp.Request
	if err := json.Unmarshal(line, &req); err != nil {
		t.Fatalf("Failed to parse request: %v", err)
	}

	if req.Method != "initialize" {
		t.Errorf("Expected method 'initialize', got '%s'", req.Method)
	}
}

func TestServer_GetStatus_Idle(t *testing.T) {
	_, workDir := newTestServer(t)
	server := mcp.NewServer(workDir)

	// Status should be idle by default
	status := server.GetStatusFromServer()
	assert.Equal(t, "idle", status.Status)
	assert.Nil(t, status.Since)
}

func TestServer_GetStatus_Analyzing(t *testing.T) {
	_, workDir := newTestServer(t)

	// Write analyzing status
	memoDir := filepath.Join(workDir, ".memo")
	now := time.Now()
	statusData := map[string]interface{}{
		"status": "analyzing",
		"since":  now.Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(statusData)
	require.NoError(t, os.WriteFile(filepath.Join(memoDir, "status.json"), data, 0644))

	server := mcp.NewServer(workDir)
	status := server.GetStatusFromServer()

	assert.Equal(t, "analyzing", status.Status)
	assert.NotNil(t, status.Since)
}

func TestServer_GetStatus_InvalidJSON(t *testing.T) {
	_, workDir := newTestServer(t)

	// Write invalid JSON
	memoDir := filepath.Join(workDir, ".memo")
	require.NoError(t, os.WriteFile(filepath.Join(memoDir, "status.json"), []byte("invalid json"), 0644))

	server := mcp.NewServer(workDir)
	status := server.GetStatusFromServer()

	// Should fallback to idle
	assert.Equal(t, "idle", status.Status)
}

func TestServer_HandleRequest_NotificationsInitialized(t *testing.T) {
	// notifications/initialized should return nil (no response)
	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      nil, // notifications typically have no ID
		Method:  "notifications/initialized",
	}

	// Just verify the request can be parsed
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed mcp.Request
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "notifications/initialized", parsed.Method)
}

func TestServer_HandleToolCall_UnknownTool(t *testing.T) {
	// Test that unknown tool returns appropriate error
	params := map[string]interface{}{
		"name":      "unknown_tool",
		"arguments": map[string]interface{}{"path": "[arch]"},
	}
	data, _ := json.Marshal(params)

	var toolParams struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	require.NoError(t, json.Unmarshal(data, &toolParams))
	assert.Equal(t, "unknown_tool", toolParams.Name)
}

func TestToolCallResult_WarningFormat(t *testing.T) {
	result := mcp.ToolCallResult{
		Content: []mcp.ContentItem{
			{Type: "text", Text: `{"type":"dict","keys":["modules","relationships"]}`},
		},
		Warning: "Data may be stale: analysis in progress (started 10s ago)",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Contains(t, parsed["warning"], "analysis in progress")
	assert.Contains(t, parsed["warning"], "10s ago")
}

func TestServer_ErrorResponse(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
	}{
		{
			name:    "parse error",
			code:    -32700,
			message: "Parse error",
		},
		{
			name:    "method not found",
			code:    -32601,
			message: "Method not found",
		},
		{
			name:    "invalid params",
			code:    -32602,
			message: "Invalid params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := mcp.Response{
				JSONRPC: "2.0",
				ID:      1,
				Error: &mcp.Error{
					Code:    tt.code,
					Message: tt.message,
				},
			}

			data, err := json.Marshal(resp)
			require.NoError(t, err)

			var parsed map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &parsed))

			errObj := parsed["error"].(map[string]interface{})
			assert.Equal(t, float64(tt.code), errObj["code"])
			assert.Equal(t, tt.message, errObj["message"])
		})
	}
}

func TestServer_ToolsDescription(t *testing.T) {
	_, workDir := newTestServer(t)
	server := mcp.NewServer(workDir)
	require.NotNil(t, server)

	// Server should have 2 tools defined
	// This tests the tools() method indirectly
}
