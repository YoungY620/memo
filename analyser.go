package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	agent "github.com/MoonshotAI/kimi-agent-sdk/go"
	"github.com/MoonshotAI/kimi-agent-sdk/go/wire"
)

//go:embed prompts/*.md
var promptFS embed.FS

// sessionPrefix is the prefix for memo-generated session IDs.
// This distinguishes memo sessions from user interactive sessions.
const sessionPrefix = "memo-"

func loadPrompt(name string) string {
	data, err := promptFS.ReadFile("prompts/" + name + ".md")
	if err != nil {
		logError("Failed to load prompt %s: %v", name, err)
		return ""
	}
	return string(data)
}

type Analyser struct {
	cfg       *Config
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

func NewAnalyser(cfg *Config, workDir string) *Analyser {
	sessionID := generateSessionID(workDir)
	logInfo("Using session ID: %s for workDir: %s", sessionID, workDir)

	return &Analyser{
		cfg:       cfg,
		indexDir:  filepath.Join(workDir, ".memo", "index"),
		workDir:   workDir,
		sessionID: sessionID,
	}
}

func (a *Analyser) Analyse(ctx context.Context, changedFiles []string) error {
	logInfo("Starting analysis for %d files", len(changedFiles))

	var session *agent.Session
	var err error

	// Use local MCP config to prevent loading ~/.kimi/mcp.json
	// (which may contain memo itself, causing infinite recursion)
	mcpFile := filepath.Join(a.workDir, ".memo", "mcp.json")

	// Use kimi defaults if agent config is not set
	if a.cfg.Agent.APIKey != "" && a.cfg.Agent.Model != "" {
		logDebug("Using configured model: %s", a.cfg.Agent.Model)
		session, err = agent.NewSession(
			agent.WithAPIKey(a.cfg.Agent.APIKey),
			agent.WithModel(a.cfg.Agent.Model),
			agent.WithWorkDir(a.workDir),
			agent.WithAutoApprove(),
			agent.WithMCPConfigFile(mcpFile),
			agent.WithSession(a.sessionID),
		)
	} else {
		logDebug("Using kimi default configuration")
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

	filesInfo := "Changed files:\n" + strings.Join(changedFiles, "\n")
	initialPrompt := contextPrompt + "\n\n" + analysePrompt + "\n\n" + filesInfo

	// Send initial prompt
	logDebug("Sending initial analysis prompt")
	if err := a.runPrompt(ctx, session, initialPrompt); err != nil {
		return err
	}

	// Validation loop
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		logDebug("Validating .memo/index files (attempt %d/%d)", i+1, maxRetries)
		result := ValidateIndex(a.indexDir)
		if result.Valid {
			logInfo("Validation passed")
			return nil
		}

		logError("Validation failed (attempt %d/%d): %s", i+1, maxRetries, FormatValidationErrors(result))

		// Send feedback prompt
		feedbackPrompt := loadPrompt("feedback")
		errorInfo := "Validation errors:\n" + FormatValidationErrors(result)
		fullFeedback := loadPrompt("context") + "\n\n" + feedbackPrompt + "\n\n" + errorInfo

		logDebug("Sending feedback prompt")
		if err := a.runPrompt(ctx, session, fullFeedback); err != nil {
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

	lb := NewLineBuffer(500 * time.Millisecond)

	// Consume all messages
	for step := range turn.Steps {
		for msg := range step.Messages {
			switch m := msg.(type) {
			case wire.ApprovalRequest:
				logDebug("Auto-approving request")
				m.Respond(wire.ApprovalRequestResponseApprove)
			case wire.ContentPart:
				if m.Type == wire.ContentPartTypeText && m.Text.Valid {
					lb.Write(m.Text.Value)
					if lines := lb.Flush(false); lines != "" {
						logDebug("Agent output: %s", lines)
					}
				}
			case wire.StatusUpdate:
				// StatusUpdate usually means a generation round is complete
				if lines := lb.Flush(true); lines != "" {
					logDebug("Agent output: %s", lines)
				}
			}
		}
		// Step ended, force flush remaining content
		if lines := lb.Flush(true); lines != "" {
			logDebug("Agent output: %s", lines)
		}
	}

	if err := turn.Err(); err != nil {
		return fmt.Errorf("turn error: %w", err)
	}

	return nil
}
