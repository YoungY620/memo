package analyzer

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/YoungY620/memo/internal"

	agent "github.com/MoonshotAI/kimi-agent-sdk/go"
	"github.com/MoonshotAI/kimi-agent-sdk/go/wire"
)

//go:embed prompts/*.md
var promptFS embed.FS

// sessionPrefix is the prefix for memo-generated session IDs.
// This distinguishes memo sessions from user interactive sessions.
const sessionPrefix = "memo-"

// maxFilesPerBatch is the threshold for splitting files into batches.
// When file count exceeds this, files are split by directory.
const maxFilesPerBatch = 100

func loadPrompt(name string) string {
	data, err := promptFS.ReadFile("prompts/" + name + ".md")
	if err != nil {
		internal.LogError("Failed to load prompt %s: %v", name, err)
		return ""
	}
	return string(data)
}

// AgentConfig holds the agent configuration
type AgentConfig struct {
	APIKey string
	Model  string
}

// Analyser performs code analysis using AI
type Analyser struct {
	agentCfg  AgentConfig
	indexDir  string
	workDir   string
	sessionID string
}

// generateSessionID creates a deterministic session ID based on work directory
// Format: <sessionPrefix><8-char-hash-of-workdir>
func generateSessionID(workDir string) string {
	hash := sha256.Sum256([]byte(workDir))
	shortHash := hex.EncodeToString(hash[:4]) // 8 hex chars
	return sessionPrefix + shortHash
}

// toRelativePaths converts absolute paths to relative paths based on workDir
func toRelativePaths(files []string, workDir string) []string {
	rel := make([]string, 0, len(files))
	for _, f := range files {
		r, err := filepath.Rel(workDir, f)
		if err != nil {
			r = f
		}
		rel = append(rel, r)
	}
	return rel
}

// splitIntoBatches splits files into batches by directory when count > threshold
func splitIntoBatches(files []string, threshold int) [][]string {
	if len(files) <= threshold {
		return [][]string{files}
	}

	// Group by first path component (top-level dir)
	groups := make(map[string][]string)
	for _, f := range files {
		parts := strings.SplitN(f, string(filepath.Separator), 2)
		dir := parts[0]
		groups[dir] = append(groups[dir], f)
	}

	var batches [][]string
	for dir, dirFiles := range groups {
		if len(dirFiles) <= threshold {
			batches = append(batches, dirFiles)
		} else {
			// Strip prefix, recurse, restore prefix
			var sub []string
			for _, f := range dirFiles {
				if idx := strings.Index(f, string(filepath.Separator)); idx >= 0 {
					sub = append(sub, f[idx+1:])
				} else {
					sub = append(sub, f)
				}
			}
			for _, b := range splitIntoBatches(sub, threshold) {
				restored := make([]string, len(b))
				for i, f := range b {
					restored[i] = filepath.Join(dir, f)
				}
				batches = append(batches, restored)
			}
		}
	}
	return batches
}

// mergeBatches merges small batches into larger ones using First Fit Decreasing algorithm.
// This solves the bin packing problem: minimize number of batches while respecting capacity.
func mergeBatches(batches [][]string, capacity int) [][]string {
	if len(batches) <= 1 {
		return batches
	}

	// Sort batches by size descending (FFD: First Fit Decreasing)
	sorted := make([][]string, len(batches))
	copy(sorted, batches)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i]) > len(sorted[j])
	})

	var merged [][]string

	for _, batch := range sorted {
		placed := false

		// Try to fit into existing merged batch (First Fit)
		for i := range merged {
			if len(merged[i])+len(batch) <= capacity {
				merged[i] = append(merged[i], batch...)
				placed = true
				break
			}
		}

		// No fit found, create new batch
		if !placed {
			newBatch := make([]string, len(batch))
			copy(newBatch, batch)
			merged = append(merged, newBatch)
		}
	}

	return merged
}

// NewAnalyser creates a new Analyser instance
func NewAnalyser(agentCfg AgentConfig, workDir string) *Analyser {
	sessionID := generateSessionID(workDir)
	internal.LogInfo("Using session ID: %s for workDir: %s", sessionID, workDir)

	return &Analyser{
		agentCfg:  agentCfg,
		indexDir:  filepath.Join(workDir, ".memo", "index"),
		workDir:   workDir,
		sessionID: sessionID,
	}
}

// Analyse performs analysis on the given changed files
func (a *Analyser) Analyse(ctx context.Context, changedFiles []string) error {
	// Convert to relative paths
	relFiles := toRelativePaths(changedFiles, a.workDir)

	// Split into batches if needed, then merge small batches
	batches := splitIntoBatches(relFiles, maxFilesPerBatch)
	batches = mergeBatches(batches, maxFilesPerBatch)
	internal.LogInfo("Starting analysis for %d files in %d batch(es)", len(changedFiles), len(batches))

	// Mark analysis in progress
	memoDir := filepath.Dir(a.indexDir)
	if err := SetStatus(memoDir, "analyzing"); err != nil {
		internal.LogError("Failed to set status: %v", err)
	}
	defer func() {
		if err := SetStatus(memoDir, "idle"); err != nil {
			internal.LogError("Failed to clear status: %v", err)
		}
	}()

	// Process each batch
	for i, batch := range batches {
		if err := a.analyseBatch(ctx, batch, i+1, len(batches)); err != nil {
			return fmt.Errorf("batch %d/%d failed: %w", i+1, len(batches), err)
		}
	}

	return nil
}

func (a *Analyser) analyseBatch(ctx context.Context, files []string, batchNum, totalBatches int) error {
	internal.LogInfo("Processing batch %d/%d (%d files)", batchNum, totalBatches, len(files))

	var session *agent.Session
	var err error

	// Use local MCP config to prevent loading ~/.kimi/mcp.json
	// (which may contain memo itself, causing infinite recursion)
	mcpFile := filepath.Join(a.workDir, ".memo", "mcp.json")

	// Use kimi defaults if agent config is not set
	if a.agentCfg.APIKey != "" && a.agentCfg.Model != "" {
		internal.LogDebug("Using configured model: %s", a.agentCfg.Model)
		session, err = agent.NewSession(
			agent.WithAPIKey(a.agentCfg.APIKey),
			agent.WithModel(a.agentCfg.Model),
			agent.WithWorkDir(a.workDir),
			agent.WithAutoApprove(),
			agent.WithMCPConfigFile(mcpFile),
			agent.WithSession(a.sessionID),
		)
	} else {
		internal.LogDebug("Using kimi default configuration")
		session, err = agent.NewSession(
			agent.WithWorkDir(a.workDir),
			agent.WithAutoApprove(),
			agent.WithMCPConfigFile(mcpFile),
			agent.WithSession(a.sessionID),
		)
	}
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Build initial prompt
	contextPrompt := loadPrompt("context")
	analysePrompt := loadPrompt("analyse")

	// Add batch info if multiple batches
	var batchInfo string
	if totalBatches > 1 {
		batchInfo = fmt.Sprintf("\n\n## Batch %d of %d\n\nThis is batch %d of %d. Previous batches have been processed. Focus on the files in this batch.", batchNum, totalBatches, batchNum, totalBatches)
	}

	filesInfo := "\n\nChanged files (relative to working directory):\n" + strings.Join(files, "\n")
	initialPrompt := contextPrompt + "\n\n" + analysePrompt + batchInfo + filesInfo

	// Send initial prompt
	internal.LogDebug("Batch %d/%d: sending initial prompt, files=%v", batchNum, totalBatches, files)
	start := time.Now()
	if err := a.runPrompt(ctx, session, initialPrompt); err != nil {
		internal.LogError("Batch %d/%d: initial prompt failed: %v", batchNum, totalBatches, err)
		return err
	}
	internal.LogDebug("Batch %d/%d: initial prompt completed, duration=%s", batchNum, totalBatches, time.Since(start))

	// Validation loop
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		internal.LogDebug("Validating .memo/index files (attempt %d/%d)", i+1, maxRetries)
		result := ValidateIndex(a.indexDir)
		if result.Valid {
			internal.LogInfo("Batch %d/%d validation passed", batchNum, totalBatches)
			return nil
		}

		errMsg := FormatValidationErrors(result)
		internal.LogError("Batch %d/%d: validation failed (attempt %d/%d): %s", batchNum, totalBatches, i+1, maxRetries, errMsg)

		// Send feedback prompt
		feedbackPrompt := loadPrompt("feedback")
		errorInfo := "Validation errors:\n" + FormatValidationErrors(result)
		fullFeedback := loadPrompt("context") + "\n\n" + feedbackPrompt + "\n\n" + errorInfo

		internal.LogDebug("Batch %d/%d: sending feedback prompt (attempt %d)", batchNum, totalBatches, i+1)
		if err := a.runPrompt(ctx, session, fullFeedback); err != nil {
			internal.LogError("Batch %d/%d: feedback prompt failed: %v", batchNum, totalBatches, err)
			return err
		}
	}

	return fmt.Errorf("validation failed after %d attempts", maxRetries)
}

// maxCheckpointFileSize is the per-file truncation limit for checkpoint content.
// Large files (especially full.jsonl transcripts) are truncated to this size.
const maxCheckpointFileSize = 200 * 1024 // 200KB

// maxCheckpointBatchSize is the target maximum total content size per batch.
// Leaves room for prompt templates and API overhead within the 4MB API limit.
const maxCheckpointBatchSize = 3 * 1024 * 1024 // 3MB

// truncateContent truncates content to maxLen bytes at a line boundary,
// appending a truncation notice.
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	// Find last newline before maxLen
	cut := strings.LastIndex(content[:maxLen], "\n")
	if cut < 0 {
		cut = maxLen
	}
	return content[:cut] + "\n\n... [truncated, showing first " + fmt.Sprintf("%dKB", cut/1024) + " of " + fmt.Sprintf("%dKB", len(content)/1024) + "]"
}

// formatCheckpointData formats a single checkpoint's data for the prompt,
// truncating large files. Returns the formatted string.
func formatCheckpointData(cp CheckpointData, index int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("### Checkpoint %d: %s (commit: %s)\n\n", index, cp.CheckpointID, cp.CommitSHA[:8]))
	for path, content := range cp.Files {
		content = truncateContent(content, maxCheckpointFileSize)
		b.WriteString(fmt.Sprintf("**File: %s**\n```\n%s\n```\n\n", path, content))
	}
	return b.String()
}

// batchCheckpoints groups checkpoints into batches where each batch's
// total formatted size stays under maxCheckpointBatchSize.
func batchCheckpoints(checkpoints []CheckpointData) [][]CheckpointData {
	var batches [][]CheckpointData
	var current []CheckpointData
	currentSize := 0

	for i, cp := range checkpoints {
		formatted := formatCheckpointData(cp, i+1)
		fmtSize := len(formatted)

		// If a single checkpoint exceeds the limit, it goes in its own batch
		if fmtSize >= maxCheckpointBatchSize {
			if len(current) > 0 {
				batches = append(batches, current)
				current = nil
				currentSize = 0
			}
			batches = append(batches, []CheckpointData{cp})
			continue
		}

		if currentSize+fmtSize > maxCheckpointBatchSize && len(current) > 0 {
			batches = append(batches, current)
			current = nil
			currentSize = 0
		}
		current = append(current, cp)
		currentSize += fmtSize
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}

// AnalyseCheckpoints analyses Entire.io checkpoint data and fuses insights
// into the existing 4 index files (arch, interface, stories, issues).
func (a *Analyser) AnalyseCheckpoints(ctx context.Context, checkpoints []CheckpointData) error {
	if len(checkpoints) == 0 {
		return nil
	}

	batches := batchCheckpoints(checkpoints)
	internal.LogInfo("Analyzing %d checkpoint(s) into index files in %d batch(es)", len(checkpoints), len(batches))

	// Mark status
	memoDir := filepath.Dir(a.indexDir)
	if err := SetStatus(memoDir, "analyzing"); err != nil {
		internal.LogError("Failed to set status: %v", err)
	}
	defer func() {
		if err := SetStatus(memoDir, "idle"); err != nil {
			internal.LogError("Failed to clear status: %v", err)
		}
	}()

	for i, batch := range batches {
		if err := a.analyseCheckpointBatch(ctx, batch, i+1, len(batches)); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyser) analyseCheckpointBatch(ctx context.Context, checkpoints []CheckpointData, batchNum, totalBatches int) error {
	internal.LogInfo("Processing checkpoint batch %d/%d (%d checkpoints)", batchNum, totalBatches, len(checkpoints))

	// Build prompt with checkpoint data
	contextPrompt := loadPrompt("context")
	analysePrompt := loadPrompt("analyse_checkpoints")

	var checkpointInfo strings.Builder
	checkpointInfo.WriteString("\n\n## Checkpoint Data\n\n")
	for i, cp := range checkpoints {
		checkpointInfo.WriteString(formatCheckpointData(cp, i+1))
	}

	var batchInfo string
	if totalBatches > 1 {
		batchInfo = fmt.Sprintf("\n\n## Batch %d of %d\n\nThis is batch %d of %d. Previous batches have been processed. Focus on the checkpoints in this batch.\n", batchNum, totalBatches, batchNum, totalBatches)
	}

	initialPrompt := contextPrompt + "\n\n" + analysePrompt + batchInfo + checkpointInfo.String()

	// Create agent session
	mcpFile := filepath.Join(a.workDir, ".memo", "mcp.json")

	var session *agent.Session
	var err error

	if a.agentCfg.APIKey != "" && a.agentCfg.Model != "" {
		session, err = agent.NewSession(
			agent.WithAPIKey(a.agentCfg.APIKey),
			agent.WithModel(a.agentCfg.Model),
			agent.WithWorkDir(a.workDir),
			agent.WithAutoApprove(),
			agent.WithMCPConfigFile(mcpFile),
			agent.WithSession(a.sessionID),
		)
	} else {
		session, err = agent.NewSession(
			agent.WithWorkDir(a.workDir),
			agent.WithAutoApprove(),
			agent.WithMCPConfigFile(mcpFile),
			agent.WithSession(a.sessionID),
		)
	}
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Send prompt
	internal.LogDebug("Checkpoint batch %d/%d: sending prompt for %d checkpoints", batchNum, totalBatches, len(checkpoints))
	start := time.Now()
	if err := a.runPrompt(ctx, session, initialPrompt); err != nil {
		return fmt.Errorf("checkpoint batch %d/%d failed: %w", batchNum, totalBatches, err)
	}
	internal.LogDebug("Checkpoint batch %d/%d completed, duration=%s", batchNum, totalBatches, time.Since(start))

	// Validation loop (reuse existing ValidateIndex for the 4 files)
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		result := ValidateIndex(a.indexDir)
		if result.Valid {
			internal.LogInfo("Checkpoint batch %d/%d validation passed", batchNum, totalBatches)
			return nil
		}

		errMsg := FormatValidationErrors(result)
		internal.LogError("Checkpoint batch %d/%d validation failed (attempt %d/%d): %s", batchNum, totalBatches, i+1, maxRetries, errMsg)

		feedbackPrompt := loadPrompt("feedback")
		errorInfo := "Validation errors:\n" + errMsg
		fullFeedback := loadPrompt("context") + "\n\n" + feedbackPrompt + "\n\n" + errorInfo

		if err := a.runPrompt(ctx, session, fullFeedback); err != nil {
			return fmt.Errorf("checkpoint batch %d/%d feedback failed: %w", batchNum, totalBatches, err)
		}
	}

	return fmt.Errorf("checkpoint batch %d/%d validation failed after %d attempts", batchNum, totalBatches, maxRetries)
}

func (a *Analyser) runPrompt(ctx context.Context, session *agent.Session, prompt string) error {
	turn, err := session.Prompt(ctx, wire.NewStringContent(prompt))
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	lb := internal.NewLineBuffer(500 * time.Millisecond)

	// Consume all messages
	for step := range turn.Steps {
		for msg := range step.Messages {
			switch m := msg.(type) {
			case wire.ApprovalRequest:
				internal.LogDebug("Auto-approving request")
				_ = m.Respond(wire.ApprovalRequestResponseApprove)
			case wire.ContentPart:
				if m.Type == wire.ContentPartTypeText && m.Text.Valid {
					lb.Write(m.Text.Value)
					if lines := lb.Flush(false); lines != "" {
						internal.LogDebug("Agent output: %s", lines)
					}
				}
			case wire.StatusUpdate:
				// StatusUpdate usually means a generation round is complete
				if lines := lb.Flush(true); lines != "" {
					internal.LogDebug("Agent output: %s", lines)
				}
			}
		}
		// Step ended, force flush remaining content
		if lines := lb.Flush(true); lines != "" {
			internal.LogDebug("Agent output: %s", lines)
		}
	}

	if err := turn.Err(); err != nil {
		return fmt.Errorf("turn error: %w", err)
	}

	return nil
}
