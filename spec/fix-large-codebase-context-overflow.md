# Fix: Large Codebase Context Overflow

Prevent context window overflow when analyzing large codebases by using relative paths and batched file processing.

## Status

- [ ] Not implemented
- [ ] Pending

## Problem

| Issue | Impact |
|-------|--------|
| Absolute paths consume tokens | `......` repeated N times |
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
                 │ Convert to relative │
                 │ paths               │
                 └─────────────────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │ estimateTokens() >  │
                 │ threshold?          │
                 └─────────────────────┘
                   │               │
                  Yes              No
                   │               │
                   ▼               ▼
        ┌──────────────────┐  ┌──────────────────┐
        │ splitIntoBatches │  │ Single batch     │
        └──────────────────┘  └──────────────────┘
                   │
                   ▼
        ┌──────────────────┐
        │ For each batch:  │
        │ 1. Build prompt  │
        │ 2. Send to LLM   │
        │ 3. Update index  │
        └──────────────────┘
```

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

+ func (a *Analyser) splitIntoBatches(files []string) [][]string {
+     var batches [][]string
+     var batch []string
+     var chars int
+     
+     for _, f := range files {
+         if len(batch) >= maxFilesPerBatch || chars+len(f) > maxCharsPerBatch {
+             if len(batch) > 0 {
+                 batches = append(batches, batch)
+             }
+             batch = nil
+             chars = 0
+         }
+         batch = append(batch, f)
+         chars += len(f) + 3
+     }
+     if len(batch) > 0 {
+         batches = append(batches, batch)
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

- [ ] Add `AnalysisBatch` struct
- [ ] Implement `estimateTokens()` function
- [ ] Implement `toRelativePaths()` function
- [ ] Implement `splitIntoBatches()` function
- [ ] Update `Analyse()` to use batching
- [ ] Update `prompts/analyse.md` template
- [ ] Add configuration options
- [ ] Test with small repo (<100 files)
- [ ] Test with large repo (>1000 files)
