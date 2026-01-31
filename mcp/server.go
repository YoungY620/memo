package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/YoungY620/memo/internal"
)

// JSON-RPC 2.0 structures
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP structures
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct{}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
	Warning string        `json:"warning,omitempty"`
}

// Status represents the analysis status from status.json
type Status struct {
	Status string     `json:"status"`
	Since  *time.Time `json:"since,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Server is the MCP server
type Server struct {
	indexDir string
	memoDir  string
	reader   *bufio.Reader
	writer   io.Writer
	history  *internal.HistoryLogger
}

// NewServer creates a new MCP server
func NewServer(workDir string) *Server {
	memoDir := filepath.Join(workDir, ".memo")
	// Ensure .memo directory exists
	os.MkdirAll(memoDir, 0755)

	h, _ := internal.NewHistoryLogger(memoDir, "mcp") // ignore error, logging is optional

	return &Server{
		indexDir: filepath.Join(memoDir, "index"),
		memoDir:  memoDir,
		reader:   bufio.NewReader(os.Stdin),
		writer:   os.Stdout,
		history:  h,
	}
}

// logRequest logs an incoming MCP request
func (s *Server) logRequest(req *Request) {
	if s.history == nil {
		return
	}
	var params any
	if len(req.Params) > 0 {
		json.Unmarshal(req.Params, &params)
	}
	s.history.Log(internal.HistoryEntry{
		Type:   "request",
		Method: req.Method,
		ID:     req.ID,
		Params: params,
	})
}

// logResponse logs an outgoing MCP response
func (s *Server) logResponse(resp *Response, duration time.Duration) {
	if s.history == nil {
		return
	}
	entry := internal.HistoryEntry{
		Type:     "response",
		ID:       resp.ID,
		Duration: duration.String(),
	}
	if resp.Error != nil {
		entry.Error = resp.Error
	} else {
		entry.Result = resp.Result
	}
	s.history.Log(entry)
}

// getStatus reads the analysis status from status.json
func (s *Server) getStatus() Status {
	path := filepath.Join(s.memoDir, "status.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Status{Status: "idle"}
	}
	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		return Status{Status: "idle"}
	}
	return status
}

// tool descriptions with schema
const schemaDesc = `Schema:
- [arch]: {modules: [{name, description, interfaces, internal?}], relationships}
- [interface]: {external: [{type, name, params, description}], internal: [...]}
- [stories]: {stories: [{title, tags, content}]}
- [issues]: {issues: [{tags, title, description, locations: [{file, keyword, line}]}]}`

const whenToUse = `**IMPORTANT: Always check project state via memo BEFORE doing anything.**

No matter what task you are working on - coding, debugging, refactoring, or answering questions - you MUST first use memo to understand the current project state. This ensures you have accurate, up-to-date context before making any changes or decisions.

**When to use:**
- BEFORE writing any code: understand the architecture and existing patterns
- BEFORE debugging: check known issues and module relationships
- BEFORE refactoring: understand interfaces and dependencies
- BEFORE answering questions: get accurate project overview

**Why memo instead of reading files directly:**
1. Pre-indexed: AI has already analyzed the entire codebase
2. Structured: Information organized by architecture, interfaces, stories, issues
3. Efficient: No need to scan hundreds of files
4. Accurate: Includes relationships, design decisions, and known issues`

func (s *Server) tools() []Tool {
	return []Tool{
		{
			Name:        "memo_list_keys",
			Description: fmt.Sprintf("%s\n\n**Function:** List available keys at a path in .memo/index JSON files.\n\n%s\n\nReturns {type: 'dict'|'list', keys?: [...], length?: N}", whenToUse, schemaDesc),
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "Path like [arch][modules][0]"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "memo_get_value",
			Description: fmt.Sprintf("%s\n\n**Function:** Get JSON value at a path in .memo/index files.\n\n%s\n\nReturns {value: '<JSON string>'}", whenToUse, schemaDesc),
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {Type: "string", Description: "Path like [arch][modules][0][name]"},
				},
				Required: []string{"path"},
			},
		},
	}
}

// Run starts the MCP server
func (s *Server) Run() error {
	if s.history != nil {
		s.history.LogInfo("MCP server started")
		defer s.history.Close()
		defer s.history.LogInfo("MCP server stopped")
	}

	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			if s.history != nil {
				s.history.LogError("read error", err)
			}
			return err
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			if s.history != nil {
				s.history.LogError("parse error", err)
			}
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		// Log request
		s.logRequest(&req)

		start := time.Now()
		resp := s.handleRequest(&req)
		duration := time.Since(start)

		if resp != nil {
			// Log response
			s.logResponse(resp, duration)
			s.sendResponse(resp)
		}
	}
}

func (s *Server) handleRequest(req *Request) *Response {
	switch req.Method {
	case "initialize":
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo: ServerInfo{
					Name:    "memo",
					Version: "1.0.0",
				},
				Capabilities: Capabilities{
					Tools: &ToolsCapability{},
				},
			},
		}

	case "notifications/initialized":
		// No response needed for notifications
		return nil

	case "tools/list":
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  ToolsListResult{Tools: s.tools()},
		}

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.errorResponse(req.ID, -32602, "Invalid params")
		}
		return s.handleToolCall(req.ID, &params)

	default:
		return s.errorResponse(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) handleToolCall(id any, params *ToolCallParams) *Response {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		return s.errorResponse(id, -32602, "Invalid arguments")
	}

	var result any
	var err error

	switch params.Name {
	case "memo_list_keys":
		result, err = ListKeys(s.indexDir, args.Path)
	case "memo_get_value":
		result, err = GetValue(s.indexDir, args.Path)
	default:
		return s.errorResponse(id, -32602, fmt.Sprintf("Unknown tool: %s", params.Name))
	}

	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      id,
			Result: ToolCallResult{
				Content: []ContentItem{{Type: "text", Text: err.Error()}},
				IsError: true,
			},
		}
	}

	// Check analysis status
	var warning string
	status := s.getStatus()
	if status.Status == "analyzing" {
		warning = "Data may be stale: analysis in progress"
		if status.Since != nil {
			warning += fmt.Sprintf(" (started %s ago)", time.Since(*status.Since).Round(time.Second))
		}
	}

	resultJSON, _ := json.Marshal(result)
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: ToolCallResult{
			Content: []ContentItem{{Type: "text", Text: string(resultJSON)}},
			Warning: warning,
		},
	}
}

func (s *Server) errorResponse(id any, code int, message string) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
}

func (s *Server) sendError(id any, code int, message string) {
	s.sendResponse(s.errorResponse(id, code, message))
}

func (s *Server) sendResponse(resp *Response) {
	data, _ := json.Marshal(resp)
	fmt.Fprintln(s.writer, string(data))
}

// Serve starts an MCP server for the given work directory
func Serve(workDir string) error {
	server := NewServer(workDir)
	return server.Run()
}
