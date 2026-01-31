package analyzer

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"path/filepath"
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

	// Split into batches if needed
	batches := splitIntoBatches(relFiles, maxFilesPerBatch)
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
				m.Respond(wire.ApprovalRequestResponseApprove)
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
