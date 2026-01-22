package prompts

import (
	"bytes"
	"text/template"
)

import _ "embed"

var (
	//go:embed assets/initialize.md
	initializeTemplateRaw string

	//go:embed assets/watch.md
	watchTemplateRaw string
)

// WatchTemplateData contains fields referenced by docs/core/prompts/watch.md.
type WatchTemplateData struct {
	WorkspaceRoot     string
	ChangeBatchID     string
	ChangedFiles      string
	ChangedFileBlobs  string
	RelatedIndexFiles string
	StorageSpecPath   string
}

// ValidatorFeedbackData contains fields for the validator feedback section.
type ValidatorFeedbackData struct {
	Attempt int
	Error   string
}

// RenderInitialize renders the initialize template with the provided data.
func RenderInitialize(data any) (string, error) {
	return renderTemplate(initializeTemplateRaw, data)
}

// RenderWatch renders the watch template.
func RenderWatch(data WatchTemplateData) (string, error) {
	return renderTemplate(watchTemplateRaw, data)
}

// AppendValidatorFeedback renders the validator feedback snippet.
func AppendValidatorFeedback(data ValidatorFeedbackData) (string, error) {
const validatorSnippet = `
Validator feedback (attempt {{.Attempt}}/100):
{{.Error}}

Use the bash tool to repair the reported issue, adjust any dependent files, and verify the validator will pass before replying. Summarize the commands you executed and note remaining blockers, if any.
`
	return renderTemplate(validatorSnippet, data)
}

func renderTemplate(raw string, data any) (string, error) {
	tpl, err := template.New("prompt").Delims("{{", "}}").Parse(raw)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// InitializeTemplate returns the raw template string (useful for diagnostics).
func InitializeTemplate() string {
	return initializeTemplateRaw
}

// WatchTemplate returns the raw template string (useful for diagnostics).
func WatchTemplate() string {
	return watchTemplateRaw
}
