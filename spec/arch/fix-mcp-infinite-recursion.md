# Fix: MCP Infinite Recursion

Fix infinite recursion when `memo --mcp-with-watcher` is configured in `~/.kimi/mcp.json`.

## Problem

When user adds memo to Kimi CLI's MCP config:

```json
// ~/.kimi/mcp.json
{
  "mcpServers": {
    "memo": {
      "command": "memo",
      "args": ["--mcp-with-watcher"]
    }
  }
}
```

The following infinite recursion occurs:

```
┌─────────────────────────────────────────────────────────────────┐
│  User starts Kimi CLI                                           │
│       │                                                         │
│       ▼                                                         │
│  Kimi CLI loads ~/.kimi/mcp.json                                │
│       │                                                         │
│       ▼                                                         │
│  Starts memo --mcp-with-watcher                                 │
│       │                                                         │
│       ├──▶ MCP server (OK)                                      │
│       │                                                         │
│       └──▶ Watcher subprocess                                   │
│                 │                                               │
│                 ▼                                                │
│            File change detected                                 │
│                 │                                               │
│                 ▼                                                │
│            Analyser.Analyse() creates Kimi session              │
│                 │                                               │
│                 ▼                                                │
│            Session loads ~/.kimi/mcp.json  ◀─────┐              │
│                 │                                 │              │
│                 ▼                                 │              │
│            Starts memo --mcp-with-watcher        │              │
│                 │                                 │              │
│                 └──▶ Watcher subprocess ─────────┘              │
│                                                                 │
│            ∞ INFINITE RECURSION until OOM                       │
└─────────────────────────────────────────────────────────────────┘
```

## Root Cause

The watcher's Kimi session (created in `analyser.go`) loads the default MCP config from `~/.kimi/mcp.json`, which contains memo itself, causing recursive spawning.

## Solution

Create `.memo/mcp.json` during memo initialization, and use it when creating watcher's Kimi session to override the default `~/.kimi/mcp.json`.

Key insight: Kimi CLI only loads `~/.kimi/mcp.json` when no `--mcp-config-file` is specified. By specifying a local config file, we prevent loading the global config.

This also enables future extensibility: users can add custom MCP servers to `.memo/mcp.json` for the watcher session if needed.

```
Watcher session with empty MCP config:
┌─────────────────────────────────────────────────────────────────┐
│  Analyser.Analyse()                                             │
│       │                                                         │
│       ▼                                                         │
│  agent.NewSession(                                              │
│      ...,                                                       │
│      agent.WithMCPConfigFile(".memo/mcp.json"),  // local MCP config    │
│  )                                                              │
│       │                                                         │
│       ▼                                                         │
│  Session starts with NO MCP servers                             │
│       │                                                         │
│       ▼                                                         │
│  Analysis completes ✓ (no recursion)                            │
└─────────────────────────────────────────────────────────────────┘
```

## Why `.memo/mcp.json`?

| Location | Pros | Cons |
|----------|------|------|
| `.memo/mcp.json` ✓ | Auto-created, user can customize, auto-cleanup | None |
| `.local/empty-mcp.json` | Separated from memo data | Requires extra dir, not always present |
| `os.TempDir()` | No project pollution | Recreate on every startup |

## Files

| File | Change |
|------|--------|
| `main.go` | Create `.memo/mcp.json` in `initIndex()` |
| `analyser.go` | Add `WithMCPConfigFile` option to session creation |

## Patch

```diff
// main.go
  func initIndex(indexDir string) error {
      if err := os.MkdirAll(indexDir, 0755); err != nil {
          return err
      }

      files := map[string]string{
          "arch.json":      `{"modules": [], "relationships": ""}`,
          "interface.json": `{"external": [], "internal": []}`,
          "stories.json":   `{"stories": []}`,
          "issues.json":    `{"issues": []}`,
      }

      for name, content := range files {
          path := filepath.Join(indexDir, name)
          if _, err := os.Stat(path); os.IsNotExist(err) {
              logDebug("Creating %s", path)
              if err := os.WriteFile(path, []byte(content), 0644); err != nil {
                  return err
              }
          }
      }

+     // Create local MCP config file for watcher sessions
+     // This prevents loading ~/.kimi/mcp.json (which may contain memo itself)
+     // Users can customize this file to add MCP servers for watcher sessions
+     memoDir := filepath.Dir(indexDir)
+     mcpFile := filepath.Join(memoDir, "mcp.json")
+     if _, err := os.Stat(mcpFile); os.IsNotExist(err) {
+         logDebug("Creating %s", mcpFile)
+         if err := os.WriteFile(mcpFile, []byte("{}"), 0644); err != nil {
+             return err
+         }
+     }

      return nil
  }
```

```diff
// analyser.go
  func (a *Analyser) Analyse(ctx context.Context, changedFiles []string) error {
      logInfo("Starting analysis for %d files", len(changedFiles))

      var session *agent.Session
      var err error

+     // Use local MCP config to prevent loading ~/.kimi/mcp.json
+     // (which may contain memo itself, causing infinite recursion)
+     mcpFile := filepath.Join(a.workDir, ".memo", "mcp.json")

      // Use kimi defaults if agent config is not set
      if a.cfg.Agent.APIKey != "" && a.cfg.Agent.Model != "" {
          logDebug("Using configured model: %s", a.cfg.Agent.Model)
          session, err = agent.NewSession(
              agent.WithAPIKey(a.cfg.Agent.APIKey),
              agent.WithModel(a.cfg.Agent.Model),
              agent.WithWorkDir(a.workDir),
              agent.WithAutoApprove(),
+             agent.WithMCPConfigFile(mcpFile),
          )
      } else {
          logDebug("Using kimi default configuration")
          session, err = agent.NewSession(
              agent.WithWorkDir(a.workDir),
              agent.WithAutoApprove(),
+             agent.WithMCPConfigFile(mcpFile),
          )
      }
      ...
  }
```

## Result

```
.memo/
├── index/
│   ├── arch.json
│   ├── interface.json
│   ├── stories.json
│   └── issues.json
└── mcp.json          # New: local MCP config (default: {})
```

## TODO

- [x] `main.go`: Create `.memo/mcp.json` in `initIndex()`
- [x] `analyser.go`: Add `WithMCPConfigFile` to both session creation paths
- [ ] `README.md`: Document `.memo/mcp.json` for user customization
- [ ] Test: Verify no recursion when memo is in `~/.kimi/mcp.json`
- [ ] Test: Verify watcher analysis still works correctly
