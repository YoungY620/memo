package validator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ValidationError represents the first failure encountered.
type ValidationError struct {
	Rule     string
	File     string
	Message  string
	Severity string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.Rule, e.File, e.Message)
}

// Config describes validator inputs.
type Config struct {
	IndexPath string
	SchemaDir string
}

// Validator checks index artifacts according to storage-design requirements.
type Validator struct {
	cfg Config
}

// New creates a validator instance.
func New(cfg Config) (*Validator, error) {
	if cfg.IndexPath == "" {
		return nil, errors.New("validator: index path is required")
	}
	if cfg.SchemaDir == "" {
		return nil, errors.New("validator: schema dir is required")
	}
	return &Validator{cfg: cfg}, nil
}

// Validate runs synchronously and returns the first failure.
func (v *Validator) Validate(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	requiredRootFiles := []string{"_index.md", "_tags.json", "_notes.json", "_activities.json"}
	for _, name := range requiredRootFiles {
		path := filepath.Join(v.cfg.IndexPath, name)
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return &ValidationError{
					Rule:    "STRUCT-00x",
					File:    name,
					Message: "required file missing",
				}
			}
			return fmt.Errorf("validator: stat %s: %w", name, err)
		}
	}

	if err := v.validateJSON("_tags.json", "tags.schema.json"); err != nil {
		return err
	}
	if err := v.validateJSON("_notes.json", "notes.schema.json"); err != nil {
		return err
	}
	if err := v.validateJSON("_activities.json", "activities.schema.json"); err != nil {
		return err
	}

	return v.walkSubmodules()
}

func (v *Validator) validateJSON(fileName, schemaName string) error {
	path := filepath.Join(v.cfg.IndexPath, fileName)
	schemaPath := filepath.Join(v.cfg.SchemaDir, schemaName)

	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
	docLoader := gojsonschema.NewReferenceLoader("file://" + path)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return fmt.Errorf("validator: schema validation %s: %w", fileName, err)
	}

	if !result.Valid() {
		errs := result.Errors()
		if len(errs) == 0 {
			return &ValidationError{
				Rule:    "JSON-000",
				File:    fileName,
				Message: "schema validation failed",
			}
		}
		first := errs[0]
		return &ValidationError{
			Rule:    "JSON-" + first.Type(),
			File:    fileName,
			Message: first.String(),
		}
	}

	return nil
}

func (v *Validator) walkSubmodules() error {
	return filepath.WalkDir(v.cfg.IndexPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path == v.cfg.IndexPath {
			return nil
		}

		rel, _ := filepath.Rel(v.cfg.IndexPath, path)
		if strings.HasPrefix(rel, "_") {
			// Skip internal directories like _reference.
			return nil
		}

		required := []string{"_index.md", "_activities.json"}
		for _, name := range required {
			full := filepath.Join(path, name)
			if _, err := os.Stat(full); errors.Is(err, fs.ErrNotExist) {
				return &ValidationError{
					Rule:    "STRUCT-submodule",
					File:    rel + "/" + name,
					Message: "missing required file",
				}
			}
		}
		return nil
	})
}

// ExtractArtifacts scans the validator index path for artifacts and returns their metadata.
func (v *Validator) ExtractArtifacts() ([]Artifact, error) {
	var artifacts []Artifact
	err := filepath.WalkDir(v.cfg.IndexPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(v.cfg.IndexPath, path)
		artifacts = append(artifacts, Artifact{Path: rel, Content: json.RawMessage(bytes)})
		return nil
	})
	return artifacts, err
}

// Artifact mirrors the LLM payload schema.
type Artifact struct {
	Path    string          `json:"path"`
	Action  string          `json:"action"`
	Content json.RawMessage `json:"content"`
}
