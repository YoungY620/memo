// Package analyzer provides analysis processing, calls Kimi API to generate index
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

// Analyzer 分析处理器
type Analyzer struct {
	cfg      *config.Config
	rootPath string
}

// New 创建新的分析器
func New(cfg *config.Config) *Analyzer {
	return &Analyzer{
		cfg:      cfg,
		rootPath: cfg.Watcher.Root,
	}
}

// Analyze 分析变更并更新索引
func (a *Analyzer) Analyze(ctx context.Context, changes []buffer.Change) error {
	if len(changes) == 0 {
		return nil
	}

	// 构建 prompt
	prompt, err := a.buildPrompt(changes)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	// 调用 Kimi
	response, err := a.callKimi(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to call Kimi: %w", err)
	}

	// 解析响应并更新索引
	if err := a.updateIndex(response); err != nil {
		return fmt.Errorf("failed to update index: %w", err)
	}

	return nil
}

// buildPrompt 构建分析 prompt
func (a *Analyzer) buildPrompt(changes []buffer.Change) (string, error) {
	var sb strings.Builder

	sb.WriteString(`You are a code analysis expert. Please analyze the following code changes and update the project semantic index.

## Index specification

1. index directory结构:
   - _index.md: 组件关系图（Mermaid UML）+ 子模块/引用摘要导航
   - _activities.json: 活动追踪（TODO/Bug/Issue 等）
   - _reference/: 详细内容
   - _tags.json: 可用 tag 列表（仅根目录）
   - _notes.json: flash-notes（仅根目录）

2. _index.md 规则:
   - 优先使用 Mermaid 图表示组件关系
   - 组件级抽象，不做白盒讲解
   - 详情放 _reference/，通过摘要引用

3. _activities.json 格式:
   {
     "<type>": {
       "items": [{ "content": "...", "tags": ["..."] }],
       "children": ["子模块相对路径"]
     }
   }

4. 子模块路径映射规则:
   - 源文件 src/core/watcher.go -> 索引 core/_reference/watcher.md
   - 源目录 src/core/trigger/ -> 索引 core/trigger/_index.md

`)

	// 添加Current index structure
	sb.WriteString("## Current index structure\n\n")
	indexTree, err := a.getIndexTree()
	if err != nil {
		sb.WriteString("（index directory不存在或为空，请创建初始索引）\n\n")
	} else {
		sb.WriteString("```\n")
		sb.WriteString(indexTree)
		sb.WriteString("\n```\n\n")
	}

	// 添加变更列表
	sb.WriteString("## Changes\n\n")
	for _, change := range changes {
		relPath, _ := filepath.Rel(a.rootPath, change.Path)
		sb.WriteString(fmt.Sprintf("### %s [%s]\n\n", relPath, change.Type.String()))

		// 如果不是删除操作，添加文件内容
		if change.Type != buffer.ChangeDelete {
			content, err := os.ReadFile(change.Path)
			if err == nil {
				// 限制内容长度
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

	// 添加Output format说明
	sb.WriteString(`## Output format

请按以下格式输出需要创建或更新的文件：

---FILE: path/to/file.md---
[文件内容]
---END---

---FILE: path/to/another.json---
[文件内容]
---END---

注意：
1. 路径相对于 .kimi-index/ 目录
2. 如果是首次创建索引，需要创建 _index.md, _tags.json, _notes.json, _activities.json
3. 每个子模块目录需要 _index.md 和 _activities.json
4. _reference/ 中的文件需要在对应 _index.md 中引用
5. 所有 tag 必须在根目录 _tags.json 中定义
`)

	return sb.String(), nil
}

// getIndexTree 获取当前index directory树
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

// callKimi 调用 Kimi API
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

// updateIndex 解析响应并更新索引文件
func (a *Analyzer) updateIndex(response string) error {
	// 确保index directory存在
	if err := os.MkdirAll(a.cfg.Index.Path, 0755); err != nil {
		return err
	}

	// 解析 ---FILE: path---...---END--- 块
	filePattern := regexp.MustCompile(`(?s)---FILE:\s*(.+?)---\n(.*?)---END---`)
	matches := filePattern.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		relPath := strings.TrimSpace(match[1])
		content := strings.TrimSpace(match[2])

		// 构建完整路径
		fullPath := filepath.Join(a.cfg.Index.Path, relPath)

		// 确保父目录存在
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// 写入文件
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}

// InitIndex initialize index directory（创建基础结构）
func (a *Analyzer) InitIndex(ctx context.Context) error {
	indexPath := a.cfg.Index.Path

	// 创建index directory
	if err := os.MkdirAll(indexPath, 0755); err != nil {
		return err
	}

	// 检查是否已有索引
	indexFile := filepath.Join(indexPath, "_index.md")
	if _, err := os.Stat(indexFile); err == nil {
		// 索引已存在
		return nil
	}

	// 创建初始文件
	backticks := "```"
	files := map[string]string{
		"_index.md": fmt.Sprintf(`# Project Index

## 组件关系

%smermaid
graph LR
    Root[项目根目录]
%s

## 子模块

| 模块 | 简介 |
|------|------|

## 引用

| 文件 | 摘要 |
|------|------|
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
