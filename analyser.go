package main

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	agent "github.com/MoonshotAI/kimi-agent-sdk/go"
	"github.com/MoonshotAI/kimi-agent-sdk/go/wire"
)

//go:embed prompts/*.md
var promptFS embed.FS

func loadPrompt(name string) string {
	data, err := promptFS.ReadFile("prompts/" + name + ".md")
	if err != nil {
		logError("Failed to load prompt %s: %v", name, err)
		return ""
	}
	return string(data)
}

type Analyser struct {
	cfg      *Config
	indexDir string
	workDir  string
}

func NewAnalyser(cfg *Config, workDir string) *Analyser {
	return &Analyser{
		cfg:      cfg,
		indexDir: filepath.Join(workDir, ".memo", "index"),
		workDir:  workDir,
	}
}

func (a *Analyser) Analyse(ctx context.Context, changedFiles []string) error {
	logInfo("Starting analysis for %d files", len(changedFiles))

	var session *agent.Session
	var err error

	// Use kimi defaults if agent config is not set
	if a.cfg.Agent.APIKey != "" && a.cfg.Agent.Model != "" {
		logDebug("Using configured model: %s", a.cfg.Agent.Model)
		session, err = agent.NewSession(
			agent.WithAPIKey(a.cfg.Agent.APIKey),
			agent.WithModel(a.cfg.Agent.Model),
			agent.WithWorkDir(a.workDir),
			agent.WithAutoApprove(),
		)
	} else {
		logDebug("Using kimi default configuration")
		session, err = agent.NewSession(
			agent.WithWorkDir(a.workDir),
			agent.WithAutoApprove(),
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

	// Consume all messages
	for step := range turn.Steps {
		for msg := range step.Messages {
			switch m := msg.(type) {
			case wire.ApprovalRequest:
				logDebug("Auto-approving request")
				m.Respond(wire.ApprovalRequestResponseApprove)
			case wire.ContentPart:
				if m.Type == wire.ContentPartTypeText && m.Text.Valid {
					logDebug("Agent output: %s", truncate(m.Text.Value, 100))
				}
			}
		}
	}

	if err := turn.Err(); err != nil {
		return fmt.Errorf("turn error: %w", err)
	}

	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
