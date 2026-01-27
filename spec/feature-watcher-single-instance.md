# Feature: Watcher Single Instance & Analysis Status

Prevent multiple watcher processes and provide analysis status awareness to MCP clients.

## Problem

1. **Multiple watchers** on same path cause data corruption
2. **MCP clients** don't know when data is being modified, may read stale/inconsistent data

## Solution

Two mechanisms:

1. **Watcher Lock**: `flock` on `.memo/watcher.lock` - prevents multiple watchers
2. **Analysis Status**: `.memo/status.json` - signals ongoing analysis to MCP clients

```
.memo/
├── index/
│   └── *.json
├── watcher.lock    ← flock for single instance
├── status.json     ← analysis status for MCP awareness
└── mcp.json
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Watcher Process                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Startup:                                                       │
│    TryLock(watcher.lock) ──failed──▶ Exit with error            │
│           │                                                     │
│         success                                                 │
│           ▼                                                     │
│    Run watcher loop                                             │
│           │                                                     │
│  On file change:                                                │
│           │                                                     │
│    SetStatus("analyzing") ◀─────────┐                           │
│           │                         │                           │
│           ▼                         │                           │
│    analyser.Analyse()               │  status.json:             │
│           │                         │  {"status": "analyzing",  │
│           ▼                         │   "since": "..."}         │
│    SetStatus("idle")                │                           │
│                                     │                           │
└─────────────────────────────────────┼───────────────────────────┘
                                      │
┌─────────────────────────────────────┼───────────────────────────┐
│                         MCP Server                              │
├─────────────────────────────────────┼───────────────────────────┤
│                                     │                           │
│  On tool call:                      │                           │
│    1. Read status.json ◀────────────┘                           │
│    2. Read index/*.json                                         │
│    3. Return result with status warning if analyzing            │
│                                                                 │
│  Response when analyzing:                                       │
│    {                                                            │
│      "content": [...],                                          │
│      "warning": "Data may be stale, analysis in progress"       │
│    }                                                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Modules

| Module | Responsibility |
|--------|----------------|
| `lock.go` | Watcher lock: `TryLock()`, `Unlock()` |
| `status.go` | Analysis status: `SetStatus()`, `GetStatus()` |
| `analyser.go` | Call `SetStatus()` before/after analysis |
| `mcp/server.go` | Check status, add warning to response |

## Files

| File | Change |
|------|--------|
| `lock.go` | New: watcher single instance lock |
| `status.go` | New: analysis status management |
| `analyser.go` | Set status before/after `Analyse()` |
| `mcp/server.go` | Check status, add warning field to response |
| `main.go` | Acquire watcher lock on startup |

## Interface

```go
// lock.go

// TryLock attempts to acquire exclusive lock on .memo/watcher.lock
func TryLock(memoDir string) (*os.File, error)

// Unlock releases the lock
func Unlock(f *os.File)
```

```go
// status.go

type Status struct {
    Status string    `json:"status"`          // "idle" | "analyzing"
    Since  time.Time `json:"since,omitempty"` // when analysis started
}

// SetStatus writes status to .memo/status.json
func SetStatus(memoDir string, status string) error

// GetStatus reads status from .memo/status.json
// Returns "idle" if file doesn't exist or is invalid
func GetStatus(memoDir string) Status
```

## Patch

### lock.go (new file)

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "syscall"
)

const lockFileName = "watcher.lock"

func TryLock(memoDir string) (*os.File, error) {
    lockPath := filepath.Join(memoDir, lockFileName)
    
    f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return nil, fmt.Errorf("failed to open lock file: %w", err)
    }
    
    err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
    if err != nil {
        f.Close()
        return nil, fmt.Errorf("another watcher is already running on this directory")
    }
    
    f.Truncate(0)
    f.Seek(0, 0)
    fmt.Fprintf(f, "%d\n", os.Getpid())
    f.Sync()
    
    return f, nil
}

func Unlock(f *os.File) {
    if f != nil {
        syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
        f.Close()
    }
}
```

### status.go (new file)

```go
package main

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"
)

const statusFileName = "status.json"

type Status struct {
    Status string     `json:"status"`
    Since  *time.Time `json:"since,omitempty"`
}

func SetStatus(memoDir string, status string) error {
    path := filepath.Join(memoDir, statusFileName)
    
    s := Status{Status: status}
    if status == "analyzing" {
        now := time.Now()
        s.Since = &now
    }
    
    data, err := json.Marshal(s)
    if err != nil {
        return err
    }
    
    return os.WriteFile(path, data, 0644)
}

func GetStatus(memoDir string) Status {
    path := filepath.Join(memoDir, statusFileName)
    
    data, err := os.ReadFile(path)
    if err != nil {
        return Status{Status: "idle"}
    }
    
    var s Status
    if err := json.Unmarshal(data, &s); err != nil {
        return Status{Status: "idle"}
    }
    
    return s
}
```

### analyser.go

```diff
  func (a *Analyser) Analyse(ctx context.Context, changedFiles []string) error {
      logInfo("Starting analysis for %d files", len(changedFiles))
+     
+     // Mark analysis in progress
+     memoDir := filepath.Dir(a.indexDir)
+     if err := SetStatus(memoDir, "analyzing"); err != nil {
+         logError("Failed to set status: %v", err)
+     }
+     defer func() {
+         if err := SetStatus(memoDir, "idle"); err != nil {
+             logError("Failed to clear status: %v", err)
+         }
+     }()

      var session *agent.Session
      // ... rest of analysis ...
  }
```

### mcp/server.go

```diff
+ import "time"

  type ToolCallResult struct {
      Content []ContentItem `json:"content"`
      IsError bool          `json:"isError,omitempty"`
+     Warning string        `json:"warning,omitempty"`
  }

+ type Status struct {
+     Status string     `json:"status"`
+     Since  *time.Time `json:"since,omitempty"`
+ }
+ 
+ func (s *Server) getStatus() Status {
+     path := filepath.Join(filepath.Dir(s.indexDir), "status.json")
+     data, err := os.ReadFile(path)
+     if err != nil {
+         return Status{Status: "idle"}
+     }
+     var status Status
+     if err := json.Unmarshal(data, &status); err != nil {
+         return Status{Status: "idle"}
+     }
+     return status
+ }

  func (s *Server) handleToolCall(id any, params *ToolCallParams) *Response {
      // ... existing code ...

+     // Check analysis status
+     var warning string
+     status := s.getStatus()
+     if status.Status == "analyzing" {
+         warning = "Data may be stale: analysis in progress"
+         if status.Since != nil {
+             warning += fmt.Sprintf(" (started %s ago)", time.Since(*status.Since).Round(time.Second))
+         }
+     }

      resultJSON, _ := json.Marshal(result)
      return &Response{
          JSONRPC: "2.0",
          ID:      id,
          Result: ToolCallResult{
              Content: []ContentItem{{Type: "text", Text: string(resultJSON)}},
+             Warning: warning,
          },
      }
  }
```

### main.go

```diff
  func main() {
      // ... flag parsing, MCP modes ...
      
      // Initialize .memo/index directory
      indexDir := filepath.Join(workDir, ".memo", "index")
      if err := initIndex(indexDir); err != nil {
          log.Fatalf("[ERROR] Failed to initialize .memo/index: %v", err)
      }
      
+     // Acquire single instance lock (watcher mode only)
+     memoDir := filepath.Join(workDir, ".memo")
+     lockFile, err := TryLock(memoDir)
+     if err != nil {
+         log.Fatalf("[ERROR] %v", err)
+     }
+     defer Unlock(lockFile)
+     
+     // Ensure status is idle on startup and exit
+     SetStatus(memoDir, "idle")
+     defer SetStatus(memoDir, "idle")
      
      // Create analyser
      analyser := NewAnalyser(cfg, workDir)
      // ... rest ...
  }
```

## Behavior

### Watcher Lock

| Scenario | Result |
|----------|--------|
| First watcher starts | Acquires lock, runs normally |
| Second watcher starts (same path) | Fails: "another watcher is already running" |
| Watcher exits/crashes | OS releases lock automatically |
| `--mcp` mode | No lock needed (read-only) |

### Analysis Status

| Scenario | status.json | MCP Response |
|----------|-------------|--------------|
| Watcher idle | `{"status":"idle"}` | Normal response |
| Analysis running | `{"status":"analyzing","since":"..."}` | Response + warning |
| No watcher running | File missing or stale | Normal response |

### MCP Response Example

```json
// Normal (idle)
{
  "content": [{"type": "text", "text": "{\"type\":\"dict\",\"keys\":[...]}"}]
}

// During analysis
{
  "content": [{"type": "text", "text": "{\"type\":\"dict\",\"keys\":[...]}"}],
  "warning": "Data may be stale: analysis in progress (started 5s ago)"
}
```

## TODO

- [x] Create `lock.go` with `TryLock()`, `Unlock()`
- [x] Create `status.go` with `SetStatus()`, `GetStatus()`
- [x] Update `analyser.go` to set status before/after analysis
- [x] Update `mcp/server.go` to check status and add warning
- [x] Update `main.go` to acquire lock and init status
- [x] Test: two watchers on same path → second fails
- [x] Test: MCP returns warning during analysis
- [x] Test: status resets to idle after analysis completes
- [x] Test: status resets to idle after watcher crash
