# Feature: Concurrent Analysis Guard

Prevent concurrent analysis when multiple Flush() calls overlap.

## Problem

```
Timer1 → Flush() → Analyse() [running 3 min...]
                      ↑
Timer2 → Flush() → Analyse() [starts while Timer1 still running]
                      ↑
                  CONFLICT: both writing to index files
```

## Modules

| Module | Responsibility |
|--------|----------------|
| `watcher` | Add mutex to prevent concurrent Flush/Analyse |

## Architecture

```
Current (problematic):
Timer1 ──→ Flush() ──→ Analyse() ────────────────→
Timer2 ────────→ Flush() ──→ Analyse() ──→  (CONFLICT!)

Proposed (with guard):
Timer1 ──→ Flush() ──→ Analyse() ────────────────→
Timer2 ────────→ Flush() [blocked, wait] ──→ Analyse() ──→
```

## Files

| File | Change |
|------|--------|
| `watcher.go` | Add `analysing sync.Mutex` to prevent concurrent analysis |

## Patch

```diff
// watcher.go
type Watcher struct {
    // ...
+   analysing sync.Mutex
}

func (w *Watcher) Flush() {
+   w.analysing.Lock()
+   defer w.analysing.Unlock()
+
    w.mu.Lock()
    // ... collect files ...
    w.mu.Unlock()

    if len(files) > 0 && w.onChange != nil {
        w.onChange(files)
    }
}
```

## TODO

- [ ] `watcher.go`: Add `analysing sync.Mutex`
- [ ] Test concurrent flush scenario
