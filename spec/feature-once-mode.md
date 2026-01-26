# Feature: Once Mode

Single scan mode: run initialization and analysis, then exit without entering watch mode.

## Architecture

```
Normal mode:  ScanAll() → pending → timer → Flush() → onChange
Once mode:    ScanAll() → pending → Flush() → onChange → exit
                                      ↑
                              Direct call, reuse same method
```

**Core change**: `flush()` → `Flush()` (make public), reused by both modes.

## Files

| File | Change |
|------|--------|
| `watcher.go` | `flush()` → `Flush()` |
| `main.go` | Add `--once` flag, call `Flush()` and exit in once mode |

## TODO

- [x] `watcher.go`: `flush()` → `Flush()`
- [x] `main.go`: Add `--once` flag and branch logic
- [x] Test verification
