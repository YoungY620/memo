# Fix: Large Codebase Context Overflow

## Problem

When the codebase is very large (e.g., includes dependencies like `node_modules`, `vendor`, or complex monorepos), the file path list alone can exceed the LLM context window limit.

**Current behavior:**
- All changed file paths are passed to the analyze prompt in one batch
- Each file path is an absolute path like `/path/to/dev/working/very-long-project-name/src/components/deeply/nested/path/to/file.tsx`
- For large codebases with thousands of files, this list can consume most of the context window

**Example scenario:**
```
Initial scan: 15,000 files
Average path length: 80 characters
Total path data: ~1.2MB (just paths, no content)
Context window: 128K tokens (~500KB text)
Result: Context overflow before any analysis begins
```

## Solution

### 1. Use Relative Paths

Refactor the analyze prompt template to:
- Provide the project working directory as an absolute path once
- Use relative paths for all file references

**Before:**
```
Changed files:
- /path/to/dev/working/my-project/src/components/Button.tsx
- /path/to/dev/working/my-project/src/components/Input.tsx
- /path/to/dev/working/my-project/src/utils/helpers.ts
```

**After:**
```
Working directory: /path/to/dev/working/my-project

Changed files:
- src/components/Button.tsx
- src/components/Input.tsx
- src/utils/helpers.ts
```

### 2. Context Window Check

Before sending the analyze request, estimate token usage:

```go
// Rough estimation: 1 token ≈ 4 characters for code/paths
func estimateTokens(text string) int {
    return len(text) / 4
}

// Check if file list exceeds threshold
const contextLimit = 128000      // tokens
const reservedForPrompt = 20000  // tokens for system prompt, schema, etc.
const reservedForResponse = 30000 // tokens for model response
const availableForFiles = contextLimit - reservedForPrompt - reservedForResponse

func needsBatching(files []string, workDir string) bool {
    totalLen := 0
    for _, f := range files {
        relPath, _ := filepath.Rel(workDir, f)
        totalLen += len(relPath) + 3 // "+ 3" for "- " prefix and newline
    }
    return estimateTokens(totalLen) > availableForFiles
}
```

### 3. Batched Analysis Flow

When file count exceeds threshold, split into batches:

```
┌─────────────────────────────────────────────────────────────┐
│                    Changed Files (N)                         │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
                   ┌─────────────────┐
                   │ Exceeds limit?  │
                   └─────────────────┘
                     │           │
                    Yes          No
                     │           │
                     ▼           ▼
          ┌──────────────┐   ┌──────────────┐
          │ Split into   │   │ Single batch │
          │ batches      │   │ analysis     │
          └──────────────┘   └──────────────┘
                │
                ▼
    ┌───────────────────────────────────┐
    │  Batch 1    Batch 2    Batch N    │
    │  (files     (files     (files     │
    │   1-500)    501-1000)  ...)       │
    └───────────────────────────────────┘
                │
                ▼
    ┌───────────────────────────────────┐
    │  Analyze each batch sequentially  │
    │  - Read current index state       │
    │  - Process batch files            │
    │  - Update index incrementally     │
    └───────────────────────────────────┘
```

### 4. Batch Processing Strategy

**Option A: Sequential batches with incremental updates**
- Process batch 1 → update index
- Process batch 2 → update index (reading previous state)
- ...continue until all batches processed

**Option B: Parallel batches with merge**
- Process all batches in parallel
- Merge index updates at the end
- Risk: Conflicts in overlapping module definitions

**Recommendation:** Option A (sequential) for simplicity and consistency.

### 5. Implementation Changes

#### 5.1 Update `prompts/analyse.md`

Add working directory context:

```markdown
# Analysis Task

**Working Directory:** {{.WorkDir}}

Files in the codebase have changed. All paths below are relative to the working directory.

## Changed Files (Batch {{.BatchNum}}/{{.TotalBatches}})

{{range .Files}}
- {{.}}
{{end}}

{{if gt .TotalBatches 1}}
**Note:** This is batch {{.BatchNum}} of {{.TotalBatches}}. Focus only on files in this batch. Previous batches have already been processed.
{{end}}
```

#### 5.2 Update `analyser.go`

```go
type AnalysisBatch struct {
    Files       []string
    BatchNum    int
    TotalBatches int
    WorkDir     string
}

func (a *Analyser) Analyse(ctx context.Context, changedFiles []string) error {
    // Convert to relative paths
    relFiles := make([]string, 0, len(changedFiles))
    for _, f := range changedFiles {
        rel, err := filepath.Rel(a.workDir, f)
        if err != nil {
            rel = f // fallback to absolute
        }
        relFiles = append(relFiles, rel)
    }
    
    // Check if batching needed
    batches := a.splitIntoBatches(relFiles)
    
    for i, batch := range batches {
        err := a.analyseBatch(ctx, AnalysisBatch{
            Files:        batch,
            BatchNum:     i + 1,
            TotalBatches: len(batches),
            WorkDir:      a.workDir,
        })
        if err != nil {
            return fmt.Errorf("batch %d/%d failed: %w", i+1, len(batches), err)
        }
    }
    return nil
}

func (a *Analyser) splitIntoBatches(files []string) [][]string {
    const maxFilesPerBatch = 500  // Conservative limit
    const maxCharsPerBatch = 50000 // ~12.5K tokens
    
    var batches [][]string
    var currentBatch []string
    var currentChars int
    
    for _, f := range files {
        if len(currentBatch) >= maxFilesPerBatch || 
           currentChars+len(f) > maxCharsPerBatch {
            if len(currentBatch) > 0 {
                batches = append(batches, currentBatch)
            }
            currentBatch = nil
            currentChars = 0
        }
        currentBatch = append(currentBatch, f)
        currentChars += len(f) + 3
    }
    
    if len(currentBatch) > 0 {
        batches = append(batches, currentBatch)
    }
    
    return batches
}
```

#### 5.3 Update prompt loading

```go
func (a *Analyser) buildAnalysePrompt(batch AnalysisBatch) string {
    tmpl := loadPrompt("analyse")
    
    var buf bytes.Buffer
    t := template.Must(template.New("analyse").Parse(tmpl))
    t.Execute(&buf, batch)
    
    return buf.String()
}
```

## Configuration

Add optional config for tuning:

```yaml
analysis:
  max_files_per_batch: 500
  max_chars_per_batch: 50000
  context_reserve_tokens: 50000  # Reserved for prompt + response
```

## Testing

1. **Small repo (<100 files):** Single batch, no change in behavior
2. **Medium repo (100-500 files):** Single batch with relative paths
3. **Large repo (500-5000 files):** Multiple batches, sequential processing
4. **Huge repo (>5000 files):** Many batches, verify index consistency

## Future Improvements

1. **Smart batching by module:** Group related files into same batch
2. **Parallel batch processing:** With proper merge strategy
3. **Incremental context:** Only send diff from previous batch
4. **File content preview:** Include first N lines of each file for better context
