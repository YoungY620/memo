# Feature: Scan Mode

Single scan mode: run initialization and analysis, then exit without entering watch mode.

## Architecture

```
Watch mode:  ScanAll() → pending → timer → Flush() → onChange → loop
Scan mode:   ScanAll() → pending → Flush() → onChange → exit
                                      ↑
                              Direct call, reuse same method
```

**Core change**: Implemented as `memo scan` subcommand using Cobra CLI framework.

## Commands

```bash
memo scan                # scan all files once and exit
memo scan -p /path       # scan specific directory
memo scan -c config.yaml # use custom config
```

## Files

| File | Description |
|------|-------------|
| `cmd/scan.go` | Scan command implementation |
| `cmd/common.go` | Shared logic with watch command |

## Implementation

The scan command reuses the watcher infrastructure but:
1. Calls `ScanAll()` to queue all files
2. Immediately calls `Flush()` to process them
3. Exits after completion (no event loop)

## TODO

- [x] Implement `cmd/scan.go`
- [x] Share initialization logic with watch via `cmd/common.go`
- [x] Update tests and documentation
