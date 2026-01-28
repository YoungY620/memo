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

## Solution

Use a capacity-1 channel as a semaphore:
- When channel is full (analysis running): skip flush, let pending files accumulate
- When channel is empty (idle): allow flush and acquire the slot
- After analysis completes: release the slot, accumulated changes will be flushed by next timer

## Architecture

```
Current (problematic):
Timer1 ──→ Flush() ──→ Analyse() ────────────────→
Timer2 ────────→ Flush() ──→ Analyse() ──→  (CONFLICT!)

Proposed (with guard):
Timer1 ──→ Flush() ──→ [acquire slot] ──→ Analyse() ──→ [release slot]
Timer2 ────────→ Flush() ──→ [slot full, skip] ──→ (files stay in pending)
Timer3 ──────────────────────────────────────→ Flush() ──→ [acquire] ──→ Analyse()
                                                              ↑
                                                   (processes all accumulated files)
```

## Design Details

### Semaphore Channel

```go
type Watcher struct {
    // ... existing fields ...
    sem chan struct{}  // capacity 1, acts as binary semaphore
}
```

### Non-blocking Acquire

```go
// tryAcquire attempts to acquire the semaphore without blocking
// Returns true if acquired, false if analysis is already running
func (w *Watcher) tryAcquire() bool {
    select {
    case w.sem <- struct{}{}:
        return true
    default:
        return false
    }
}
```

### Release

```go
// release releases the semaphore after analysis completes
func (w *Watcher) release() {
    <-w.sem
}
```

### Flush Logic

```go
func (w *Watcher) Flush() {
    // Try to acquire semaphore (non-blocking)
    if !w.tryAcquire() {
        logDebug("Analysis in progress, skipping flush (files remain in pending)")
        return
    }
    defer w.release()

    // Collect pending files
    w.mu.Lock()
    // ... stop timers, collect files, reset pending ...
    w.mu.Unlock()

    // Run analysis
    if len(files) > 0 && w.onChange != nil {
        w.onChange(files)
    }
}
```

## Behavior

| Scenario | Behavior |
|----------|----------|
| Flush when idle | Acquire slot → run analysis → release slot |
| Flush when analyzing | Skip flush, files stay in pending, log debug message |
| Timer fires after skip | Will flush all accumulated files (including previously skipped) |
| Rapid file changes during analysis | All changes accumulate in pending, batch processed after current analysis |

## Benefits

1. **Non-blocking**: Timer callbacks return immediately, no goroutine pile-up
2. **Batching**: Skipped flushes result in larger batches = fewer AI calls
3. **Simple**: Single channel, no complex mutex logic
4. **Safe**: Guaranteed single analysis at a time

## Files

| File | Change |
|------|--------|
| `watcher.go` | Add `sem chan struct{}` and guard logic in Flush() |

## Patch

```diff
// watcher.go

type Watcher struct {
    debounceMs, maxWaitMs int
    ignorePatterns        []string
    onChange              func([]string)
    watcher               *fsnotify.Watcher
    rootPath              string

    mu                sync.Mutex
    pending           map[string]struct{}
    debounce, maxWait *time.Timer
+   sem               chan struct{}  // capacity 1 semaphore for analysis guard
}

func NewWatcher(root string, ignore []string, debounceMs, maxWaitMs int, onChange func([]string)) (*Watcher, error) {
    fsw, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }
    w := &Watcher{
        rootPath:       root,
        ignorePatterns: ignore,
        debounceMs:     debounceMs,
        maxWaitMs:      maxWaitMs,
        onChange:       onChange,
        watcher:        fsw,
        pending:        make(map[string]struct{}),
+       sem:            make(chan struct{}, 1),
    }
    // ...
}

func (w *Watcher) Flush() {
+   // Non-blocking acquire: skip if analysis already running
+   select {
+   case w.sem <- struct{}{}:
+       // acquired
+   default:
+       logDebug("Analysis in progress, skipping flush (files remain in pending)")
+       return
+   }
+   defer func() { <-w.sem }()  // release on exit
+
    w.mu.Lock()
    if w.debounce != nil {
        w.debounce.Stop()
        w.debounce = nil
    }
    if w.maxWait != nil {
        w.maxWait.Stop()
        w.maxWait = nil
    }
    files := make([]string, 0, len(w.pending))
    for f := range w.pending {
        files = append(files, f)
    }
    w.pending = make(map[string]struct{})
    w.mu.Unlock()

    if len(files) > 0 && w.onChange != nil {
        w.onChange(files)
    }
}
```

## TODO

- [x] `watcher.go`: Add `sem chan struct{}` field
- [x] `watcher.go`: Initialize `sem` in NewWatcher with `make(chan struct{}, 1)`
- [x] `watcher.go`: Add semaphore guard at start of Flush()
- [ ] Test: verify concurrent flush calls don't cause conflicts
- [ ] Test: verify skipped flush files are processed in next flush
