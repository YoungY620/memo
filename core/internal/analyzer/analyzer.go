// Package analyzer provides analysis processing functionality, calls Kimi API to generate index
package analyzer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	kimi "github.com/MoonshotAI/kimi-agent-sdk/go"
	"github.com/MoonshotAI/kimi-agent-sdk/go/wire"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/buffer"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/config"
)

// Analyzer analysis processor
type Analyzer struct {
	cfg      *config.Config
	rootPath string
}

// New creates a new analyzer
func New(cfg *config.Config) *Analyzer {
	return &Analyzer{
		cfg:      cfg,
		rootPath: cfg.Watcher.Root,
	}
}

// Analyze analyzes changes and updates index
func (a *Analyzer) Analyze(ctx context.Context, changes []buffer.Change) error {
	if len(changes) == 0 {
		return nil
	}

	// Build prompt
	prompt, err := a.buildPrompt(changes)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	// Call Kimi
	response, err := a.callKimi(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to call Kimi: %w", err)
	}

	// Parse response and update index
	if err := a.updateIndex(response); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	return nil
}

// buildPrompt builds the analysis prompt
func (a *Analyzer) buildPrompt(changes []buffer.Change) (string, error) {
	var sb strings.Builder

	sb.WriteString(`You are a code analysis expert. Please analyze the following code changes and update the project semantic index.

## Index Specification

1. Index directory structure:
   - _index.md: Component relationship diagram (Mermaid UML) + submodule/reference summary navigation
   - _activities.json: Activity tracking (TODO/Bug/Issue etc.)
   - _reference/: Detailed content
   - _tags.json: Available tag list (root directory only)
   - _notes.json: Flash-notes (root directory only)

2. _index.md rules:
   - Prefer Mermaid diagrams for component relationships
   - Component-level abstraction, no white-box explanation
   - Details go to _reference/, referenced via summary

3. _activities.json format:
   {
     "<type>": {
       "items": [{ "content": "...", "tags": ["..."] }],
       "children": ["relative path to submodule"]
     }
   }

4. Submodule path mapping rules:
   - Source file src/core/watcher.go -> Index core/_reference/watcher.md
   - Source directory src/core/trigger/ -> Index core/trigger/_index.md

`)

	// Add current index structure
	sb.WriteString("## Current Index Structure\n\n")
	indexTree, err := a.getIndexTree()
	if err != nil {
		sb.WriteString("(Index directory does not exist or is empty, please create initial index)\n\n")
	} else {
		sb.WriteString("```\n")
		sb.WriteString(indexTree)
		sb.WriteString("\n```\n\n")
	}

	// Add change list
	sb.WriteString("## Changes\n\n")
	for _, change := range changes {
		relPath, _ := filepath.Rel(a.rootPath, change.Path)
		sb.WriteString(fmt.Sprintf("### %s [%s]\n\n", relPath, change.Type.String()))

		// If not delete operation, add file content
		if change.Type != buffer.ChangeDelete {
			content, err := os.ReadFile(change.Path)
			if err == nil {
				// Limit content length
				contentStr := string(content)
				if len(contentStr) > 5000 {
					contentStr = contentStr[:5000] + "\n... (content too long, truncated)"
				}
				sb.WriteString("```\n")
				sb.WriteString(contentStr)
				sb.WriteString("\n```\n\n")
			}
		}
	}

	// Add output format description
	sb.WriteString(`## Output Format

Please output files to create or update in the following format:

---FILE: path/to/file.md---
[file content]
---END---

---FILE: path/to/another.json---
[file content]
---END---

Notes:
1. Paths are relative to .kimi-index/ directory
2. If creating index for the first time, create _index.md, _tags.json, _notes.json, _activities.json
3. Each submodule directory needs _index.md and _activities.json
4. Files in _reference/ must be referenced in corresponding _index.md
5. All tags must be defined in root _tags.json
`)

	return sb.String(), nil
}

// getIndexTree gets current index directory tree
func (a *Analyzer) getIndexTree() (string, error) {
	indexPath := a.cfg.Index.Path
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return "", err
	}

	var sb strings.Builder
	err := filepath.Walk(indexPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		relPath, _ := filepath.Rel(indexPath, path)
		if relPath == "." {
			return nil
		}
		depth := strings.Count(relPath, string(filepath.Separator))
		indent := strings.Repeat("  ", depth)
		if info.IsDir() {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, info.Name()))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", indent, info.Name()))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

// callKimi calls Kimi API
func (a *Analyzer) callKimi(ctx context.Context, prompt string) (string, error) {
	session, err := kimi.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	turn, err := session.Prompt(ctx, wire.NewStringContent(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}

	var result strings.Builder
	for step := range turn.Steps {
		for msg := range step.Messages {
			if cp, ok := msg.(wire.ContentPart); ok && cp.Type == wire.ContentPartTypeText {
				result.WriteString(cp.Text.Value)
			}
		}
	}

	return result.String(), nil
}

// updateIndex parses response and updates index files
func (a *Analyzer) updateIndex(response string) error {
	// Ensure index directory exists
	if err := os.MkdirAll(a.cfg.Index.Path, 0755); err != nil {
		return err
	}

	// Parse ---FILE: path---...---END--- blocks
	filePattern := regexp.MustCompile(`(?s)---FILE:\s*(.+?)---\n(.*?)---END---`)
	matches := filePattern.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		relPath := strings.TrimSpace(match[1])
		content := strings.TrimSpace(match[2])

		// Build full path
		fullPath := filepath.Join(a.cfg.Index.Path, relPath)

		// Ensure parent directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}

// InitIndex initializes index directory (creates base structure)
func (a *Analyzer) InitIndex(ctx context.Context) error {
	indexPath := a.cfg.Index.Path

	// Create index directory
	if err := os.MkdirAll(indexPath, 0755); err != nil {
		return err
	}

	// Check if index already exists
	indexFile := filepath.Join(indexPath, "_index.md")
	if _, err := os.Stat(indexFile); err == nil {
		// Index already exists
		return nil
	}

	// Create initial files
	backticks := "```"
	files := map[string]string{
		"_index.md": fmt.Sprintf(`# Project Index

## Component Relationships

%smermaid
graph LR
    Root[Project Root]
%s

## Submodules

| Module | Description |
|--------|-------------|

## References

| File | Summary |
|------|---------|
`, backticks, backticks),
		"_tags.json":       `["todo", "bug", "feature", "refactor", "test", "docs"]`,
		"_notes.json":      `[]`,
		"_activities.json": `{}`,
	}

	for name, content := range files {
		path := filepath.Join(indexPath, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}
