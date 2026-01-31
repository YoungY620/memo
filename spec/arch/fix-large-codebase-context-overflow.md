# Fix: Large Codebase Context Overflow

Prevent context window overflow when analyzing large codebases by using relative paths and batched file processing.

## Status

- [x] Implemented

## Problem

| Issue | Impact |
|-------|--------|
| Absolute paths consume tokens | `/path/to/project/very-long-path/...` repeated N times |
| All files sent in one batch | 15,000 files × 80 chars = 1.2MB (exceeds 128K context) |
| No overflow detection | Analysis fails silently or produces incomplete results |

**Example:**
```
Files: 15,000
Avg path: 80 characters  
Total: ~1.2MB text (just paths)
Context: 128K tokens (~500KB)
Result: Overflow before analysis begins
```

## Solution

| Component | Description |
|-----------|-------------|
| Relative paths | Provide workDir once, use relative paths for files |
| Token estimation | `len(text) / 4` ≈ tokens |
| Batching | Split files when exceeding threshold |
| Sequential processing | Process batch → update index → next batch |

## Modules

| Module | Responsibility |
|--------|----------------|
| `analyser.go` | Batch splitting, relative path conversion |
| `prompts/analyse.md` | Template with workDir and batch info |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Changed Files (N)                         │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │ len(files) >        │
                 │ threshold?          │
                 └─────────────────────┘
                   │               │
                  Yes              No
                   │               │
                   ▼               ▼
        ┌──────────────────┐  ┌──────────────────┐
        │ Split by        │  │ Single batch     │
        │ directory       │  │ (no split)       │
        │ (recursive)     │  └──────────────────┘
        └──────────────────┘
                   │
                   ▼
        ┌──────────────────┐
        │ For each batch:  │
        │ 1. Build prompt  │
        │ 2. Send to LLM   │
        │ 3. Update index  │
        └──────────────────┘
```

### Recursive Directory Splitting Strategy

Only split when file count exceeds threshold. Split by top-level directories first, then recursively split any directory that still exceeds the threshold.

```
Threshold: 100 files

Input: 250 files across the project
  src/api/      (80 files)  → ✔ under threshold, batch as-is
  src/db/       (120 files) → ✘ over threshold, split further
    src/db/models/  (60 files)  → ✔ batch
    src/db/queries/ (60 files)  → ✔ batch
  pkg/          (30 files)  → ✔ under threshold, batch as-is
  root files    (20 files)  → ✔ under threshold, batch as-is

Result: 4 batches
  Batch 1: src/api/* (80 files)
  Batch 2: src/db/models/* (60 files)
  Batch 3: src/db/queries/* (60 files)
  Batch 4: pkg/* + root files (50 files)
```

### Algorithm

```
splitIntoBatches(files, threshold):
  if len(files) <= threshold:
    return [files]  // single batch, no split needed
  
  // Group by top-level directory
  groups = groupByTopDir(files)
  
  batches = []
  for dir, dirFiles in groups:
    if len(dirFiles) <= threshold:
      batches.append(dirFiles)
    else:
      // Recursively split this directory
      subBatches = splitIntoBatches(dirFiles, threshold)
      batches.extend(subBatches)
  
  return batches
```

Benefits:
- No unnecessary splitting for small changesets
- Related files stay together when possible
- Deep directories split only when needed
- Preserves module boundaries

## Files

| File | Change |
|------|--------|
| `analyser.go` | Add `splitIntoBatches()`, `estimateTokens()`, update `Analyse()` |
| `prompts/analyse.md` | Add `{{.WorkDir}}`, batch number, relative paths |

## Patch

```diff
// analyser.go

+ const (
+     maxFilesPerBatch = 500
+     maxCharsPerBatch = 50000  // ~12.5K tokens
+     contextReserve   = 50000  // tokens for prompt + response
+ )

+ type AnalysisBatch struct {
+     Files        []string
+     BatchNum     int
+     TotalBatches int
+     WorkDir      string
+ }

+ func estimateTokens(text string) int {
+     return len(text) / 4
+ }

+ func (a *Analyser) toRelativePaths(files []string) []string {
+     rel := make([]string, 0, len(files))
+     for _, f := range files {
+         r, err := filepath.Rel(a.workDir, f)
+         if err != nil {
+             r = f
+         }
+         rel = append(rel, r)
+     }
+     return rel
+ }

+ // splitIntoBatches splits files into batches by directory when count > threshold
+ func splitIntoBatches(files []string, threshold int) [][]string {
+     if len(files) <= threshold {
+         return [][]string{files}
+     }
+     
+     // Group by first path component (top-level dir)
+     groups := make(map[string][]string)
+     for _, f := range files {
+         dir := strings.SplitN(f, "/", 2)[0]
+         groups[dir] = append(groups[dir], f)
+     }
+     
+     var batches [][]string
+     for dir, dirFiles := range groups {
+         if len(dirFiles) <= threshold {
+             batches = append(batches, dirFiles)
+         } else {
+             // Strip prefix, recurse, restore prefix
+             var sub []string
+             for _, f := range dirFiles {
+                 if i := strings.Index(f, "/"); i >= 0 {
+                     sub = append(sub, f[i+1:])
+                 } else {
+                     sub = append(sub, f)
+                 }
+             }
+             for _, b := range splitIntoBatches(sub, threshold) {
+                 var restored []string
+                 for _, f := range b {
+                     restored = append(restored, dir+"/"+f)
+                 }
+                 batches = append(batches, restored)
+             }
+         }
+     }
+     return batches
+ }

  func (a *Analyser) Analyse(ctx context.Context, changedFiles []string) error {
+     // Convert to relative paths
+     relFiles := a.toRelativePaths(changedFiles)
+     
+     // Split into batches if needed
+     batches := a.splitIntoBatches(relFiles)
+     
+     for i, batch := range batches {
+         if err := a.analyseBatch(ctx, AnalysisBatch{
+             Files:        batch,
+             BatchNum:     i + 1,
+             TotalBatches: len(batches),
+             WorkDir:      a.workDir,
+         }); err != nil {
+             return fmt.Errorf("batch %d/%d: %w", i+1, len(batches), err)
+         }
+     }
+     return nil
  }
```

```diff
// prompts/analyse.md

  # Analysis Task

+ **Working Directory:** {{.WorkDir}}
+ 
  Files in the codebase have changed.
+ All paths below are relative to the working directory.

+ ## Changed Files (Batch {{.BatchNum}}/{{.TotalBatches}})

- Changed files:
  {{range .Files}}
  - {{.}}
  {{end}}

+ {{if gt .TotalBatches 1}}
+ **Note:** This is batch {{.BatchNum}} of {{.TotalBatches}}.
+ Previous batches have been processed.
+ {{end}}
```

## Configuration

```yaml
analysis:
  max_files_per_batch: 500
  max_chars_per_batch: 50000
  context_reserve_tokens: 50000
```

## TODO

- [x] Implement `splitIntoBatches()` 
- [x] Implement `toRelativePaths()`
- [x] Update `Analyse()` to use batching
- [x] Add batch info to prompt dynamically
- [ ] Test with large codebase
