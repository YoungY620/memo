# Feature: MCP with Watcher Mode

Add `--mcp-with-watcher` flag to run MCP server + Watcher subprocess together.

## Modules

| Module | Responsibility |
|--------|----------------|
| `main` | Mode switching, subprocess management |

## Architecture

```
Default mode (no flag):
┌─────────────────────────────────────┐
│              memo                   │
│         Watcher only                │
└─────────────────────────────────────┘

--mcp mode:
┌─────────────────────────────────────┐
│           memo --mcp                │
│       MCP Server only (stdio)       │
└─────────────────────────────────────┘

--mcp-with-watcher mode:
┌─────────────────────────────────────────────┐
│           memo --mcp-with-watcher             │
│                (main process)                 │
│                                               │
│  1. spawn subprocess: memo --path <workDir>   │
│  2. run MCP server (stdio)                    │
│  3. on exit: kill subprocess                  │
└───────────────────┬─────────────────────────┘
                    │ spawn
                    ▼
          ┌───────────────────┐
          │ memo --path X     │
          │  (subprocess)     │
          │  logs → /dev/null │
          └───────────────────┘
```

## Flags

| Flag | Behavior |
|------|----------|
| (none) | Watcher only (default, existing behavior) |
| `--mcp` | MCP server only (existing) |
| `--mcp-with-watcher` | MCP server + Watcher subprocess (new) |
| `--once` | One-time scan + analysis, then exit |

## Files

| File | Change |
|------|--------|
| `main.go` | Add `--mcp-with-watcher` flag, subprocess spawn logic |

## Patch

```diff
// main.go
  var (
      mcpFlag = flag.Bool("mcp", false, "Run as MCP server")
+     mcpWithWatcherFlag = flag.Bool("mcp-with-watcher", false, "MCP server + Watcher")
  )

  func main() {
+     // MCP with Watcher mode
+     if *mcpWithWatcherFlag {
+         // Spawn watcher subprocess
+         cmd := exec.Command(os.Args[0], "--path", workDir)
+         cmd.Stdout = nil
+         cmd.Stderr = nil
+         cmd.Start()
+         defer cmd.Process.Kill()
+
+         // Run MCP server
+         mcp.Serve(workDir)
+         return
+     }
+
      // MCP only mode
      if *mcpFlag {
          mcp.Serve(workDir)
          return
      }

      // Default: Watcher only (existing behavior)
      ...
  }
```

## TODO

- [x] `main.go`: Add `--mcp-with-watcher` flag
- [x] `main.go`: Spawn watcher subprocess
- [x] `main.go`: Kill subprocess on exit
- [x] Test `--mcp-with-watcher` mode
- [x] Test `--mcp` mode still works
- [x] Test default mode still works
