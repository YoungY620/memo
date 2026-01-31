# Fix: Session ID Pollution

Prevent memo from creating multiple Kimi sessions by using a deterministic session ID based on work directory.

## Status

- [x] Implemented
- [x] Tested

## Problem

| Issue | Impact |
|-------|--------|
| Each analysis creates new session | Hundreds of sessions accumulate over time |
| Sessions named by timestamp | Hard to identify memo vs user sessions |
| No session reuse | Wastes storage and clutters session list |

**Before:**
```
~/.kimi/sessions/<workdir-hash>/
├── 2026-01-26-001/    # memo
├── 2026-01-26-002/    # memo
├── 2026-01-26-003/    # user
├── ...                # 100+ sessions
```

## Solution

| Component | Description |
|-----------|-------------|
| Session ID format | `memo-<8-char-sha256-hash>` |
| Hash input | Absolute path of work directory |
| Behavior | Same project always uses same session |

**After:**
```
~/.kimi/sessions/<workdir-hash>/
├── 2026-01-26-001/    # user
├── memo-aef71e24/     # all memo analyses
```

## Modules

| Module | Responsibility |
|--------|----------------|
| `analyser.go` | Generate and use deterministic session ID |
| `agent.WithSession()` | Kimi SDK option to specify session ID |

## Architecture

```
┌─────────────────┐
│   NewAnalyser   │
└────────┬────────┘
         │ workDir
         ▼
┌─────────────────┐
│generateSessionID│──▶ "memo-" + sha256(workDir)[:8]
└────────┬────────┘
         │ sessionID
         ▼
┌─────────────────┐
│    Analyser     │
│  {sessionID}    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ agent.NewSession│
│ WithSession(id) │
└─────────────────┘
```

## Files

| File | Change |
|------|--------|
| `analyser.go` | Add `sessionID` field, `generateSessionID()` func, use `WithSession()` |

## Patch

```diff
// analyser.go

  import (
+     "crypto/sha256"
+     "encoding/hex"
      // ...
  )

+ // sessionPrefix is the prefix for memo-generated session IDs.
+ // This distinguishes memo sessions from user interactive sessions.
+ const sessionPrefix = "memo-"

  type Analyser struct {
      cfg       *Config
      indexDir  string
      workDir   string
+     sessionID string
  }

+ // generateSessionID creates a deterministic session ID based on work directory
+ // Format: <sessionPrefix><8-char-hash-of-workdir>
+ func generateSessionID(workDir string) string {
+     hash := sha256.Sum256([]byte(workDir))
+     shortHash := hex.EncodeToString(hash[:4])
+     return sessionPrefix + shortHash
+ }

  func NewAnalyser(cfg *Config, workDir string) *Analyser {
+     sessionID := generateSessionID(workDir)
+     logInfo("Using session ID: %s for workDir: %s", sessionID, workDir)

      return &Analyser{
          cfg:       cfg,
          indexDir:  filepath.Join(workDir, ".memo", "index"),
          workDir:   workDir,
+         sessionID: sessionID,
      }
  }

  func (a *Analyser) Analyse(...) error {
      session, err := agent.NewSession(
          agent.WithWorkDir(a.workDir),
          agent.WithAutoApprove(),
          agent.WithMCPConfigFile(mcpFile),
+         agent.WithSession(a.sessionID),
      )
      // ...
  }
```

## Verification

| Test | Command | Result |
|------|---------|--------|
| Session ID format | `memo --once 2>&1 \| grep session` | `memo-aef71e24` ✓ |
| Session created | `find ~/.kimi/sessions -name "memo-*"` | Directory exists ✓ |
| Session reused | Run twice, check timestamp | Same session updated ✓ |
| Kimi CLI compat | `kimi --session memo-xxx -p "test"` | Works ✓ |

**Test output:**
```
WorkDir: /path/to/your/project
Session: memo-aef71e24
Path:    ~/.kimi/sessions/<workdir-hash>/memo-aef71e24/
```

## TODO

- [x] Add `sessionID` field to Analyser struct
- [x] Implement `generateSessionID()` function
- [x] Use `agent.WithSession()` in both NewSession calls
- [x] Test first run (session creation)
- [x] Test second run (session reuse)
- [x] Verify Kimi CLI compatibility
