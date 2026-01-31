# Feature: MCP JSON Query Interface

MCP server exposes `.memo/index/*.json` via safe JSON path query, starts alongside memo watcher.

## Usage

```bash
# Terminal 1: Start watcher (maintains index)
memo --path /project/dir

# Terminal 2: Start MCP server (read-only query)
memo --mcp --path /project/dir
```

- `--mcp` mode: Only provides index query service, does NOT create index
- If index not found, MCP mode exits with error
- Two processes share same `.memo/index` directory

## Modules

| Module | Responsibility |
|--------|----------------|
| `mcp/server` | MCP stdio server, tool registration |
| `mcp/query` | Path parser (state machine), query executor |
| `main` | `--mcp` flag switches between watcher/MCP mode |

## Architecture

```
 Terminal 1                          Terminal 2
┌───────────────────┐                ┌───────────────────┐
│  memo --path X    │                │ memo --mcp        │
│                   │                │      --path X     │
├───────────────────┤                ├───────────────────┤
│     Watcher       │                │   MCP Server      │
│  (file watching)  │                │    (stdio)        │
│                   │                ├─────────┬─────────┤
│     Analyser      │                │list_keys│get_val  │
│  (update index)   │                └─────────┴─────────┘
└─────────┬─────────┘                         │
          │       write                read   │
          ▼                                   ▼
     ┌─────────────────────────────────────┐
     │      .memo/index/*.json             │
     │ arch │ interface │ stories │ issues │
     └─────────────────────────────────────┘
```

## MCP Tool Definitions

### list_keys

```json
{
  "name": "memo_list_keys",
  "description": "List available keys at a path in .memo/index JSON files.\n\nSchema:\n- [arch]: {modules: [{name, description, interfaces, internal?}], relationships}\n- [interface]: {external: [{type, name, params, description}], internal: [...]}\n- [stories]: {stories: [{title, tags, content}]}\n- [issues]: {issues: [{tags, title, description, locations: [{file, keyword, line}]}]}\n\nReturns {type: 'dict'|'list', keys?: [...], length?: N}",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {"type": "string", "description": "Path like [arch][modules][0]"}
    },
    "required": ["path"]
  }
}
```

### get_value

```json
{
  "name": "memo_get_value",
  "description": "Get JSON value at a path in .memo/index files.\n\nSchema:\n- [arch]: {modules: [{name, description, interfaces, internal?}], relationships}\n- [interface]: {external: [{type, name, params, description}], internal: [...]}\n- [stories]: {stories: [{title, tags, content}]}\n- [issues]: {issues: [{tags, title, description, locations: [{file, keyword, line}]}]}\n\nReturns {value: '<JSON string>'}",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {"type": "string", "description": "Path like [arch][modules][0][name]"}
    },
    "required": ["path"]
  }
}
```

### Examples

```
list_keys({path: "[arch][modules]"})
  -> {type: "list", length: 3}

list_keys({path: "[arch][modules][0]"})
  -> {type: "dict", keys: ["name", "description", "interfaces"]}

get_value({path: "[arch][modules][0][name]"})
  -> {value: "\"main\""}
```

## Path Syntax

```
[file][key][index][key]...
  │     │     │     │
  ▼     ▼     ▼     ▼
file  key   idx   key
```

## Files

| File | Change |
|------|--------|
| `mcp/server.go` | New: MCP stdio server |
| `mcp/query.go` | New: Path parser + query executor |
| `mcp/query_test.go` | New: Security tests |
| `main.go` | Start MCP server in goroutine |

## Patch

```diff
// main.go
+ import "github.com/YoungY620/memo/mcp"

  var (
      mcpFlag = flag.Bool("mcp", false, "Run as MCP server (stdio)")
  )

  func main() {
+     // MCP server mode (read-only, index must exist)
+     if *mcpFlag {
+         indexDir := filepath.Join(workDir, ".memo", "index")
+         if _, err := os.Stat(indexDir); os.IsNotExist(err) {
+             log.Fatalf("Index not found: %s", indexDir)
+         }
+         return mcp.Serve(workDir)
+     }
      // ... existing watcher setup ...
  }
```

```diff
// mcp/server.go (new file)
+ func Serve(workDir string) error
```

```diff
// mcp/query.go (new file)
+ func ParsePath(path string) (file string, segments []PathSegment, error)
+ func ListKeys(indexDir, path string) (*ListKeysResult, error)
+ func GetValue(indexDir, path string) (*GetValueResult, error)
```

## TODO

- [x] `mcp/query.go`: Path parser (state machine)
- [x] `mcp/query.go`: ListKeys + GetValue executors
- [x] `mcp/query_test.go`: Security tests (injection, bounds)
- [x] `mcp/server.go`: MCP stdio server + two tools
- [x] `main.go`: Integrate MCP server startup
- [x] Integration test
