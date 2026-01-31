package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YoungY620/memo/analyzer"
)

func setupTestIndex(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	indexDir := filepath.Join(dir, ".memo", "index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"arch.json": `{
			"modules": [
				{"name": "main", "description": "entry point", "interfaces": "CLI"}
			],
			"relationships": "main -> config"
		}`,
		"interface.json": `{
			"external": [{"type": "cli", "name": "--help", "params": "none", "description": "show help"}],
			"internal": []
		}`,
		"stories.json": `{
			"stories": [{"title": "User Login", "tags": ["auth"], "content": "User logs in with credentials"}]
		}`,
		"issues.json": `{
			"issues": [{"tags": ["todo"], "title": "Fix bug", "description": "Fix the bug", "locations": [{"file": "main.go", "keyword": "TODO", "line": 10}]}]
		}`,
	}

	for name, content := range files {
		path := filepath.Join(indexDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return indexDir
}

func TestValidateIndex_Valid(t *testing.T) {
	indexDir := setupTestIndex(t)

	result := analyzer.ValidateIndex(indexDir)
	if !result.Valid {
		t.Errorf("Expected valid index, got errors: %v", result.Errors)
	}
}

func TestValidateIndex_MissingFile(t *testing.T) {
	dir := t.TempDir()
	indexDir := filepath.Join(dir, ".memo", "index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Only create some files
	os.WriteFile(filepath.Join(indexDir, "arch.json"), []byte(`{"modules": [], "relationships": ""}`), 0644)

	result := analyzer.ValidateIndex(indexDir)
	if result.Valid {
		t.Error("Expected invalid result for missing files")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for missing files")
	}
}

func TestValidateIndex_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	indexDir := filepath.Join(dir, ".memo", "index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"arch.json":      `invalid json`,
		"interface.json": `{"external": [], "internal": []}`,
		"stories.json":   `{"stories": []}`,
		"issues.json":    `{"issues": []}`,
	}

	for name, content := range files {
		path := filepath.Join(indexDir, name)
		os.WriteFile(path, []byte(content), 0644)
	}

	result := analyzer.ValidateIndex(indexDir)
	if result.Valid {
		t.Error("Expected invalid result for invalid JSON")
	}
}

func TestValidateIndex_SchemaMismatch(t *testing.T) {
	dir := t.TempDir()
	indexDir := filepath.Join(dir, ".memo", "index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files with wrong schema
	files := map[string]string{
		"arch.json":      `{"wrong": "schema"}`,
		"interface.json": `{"external": [], "internal": []}`,
		"stories.json":   `{"stories": []}`,
		"issues.json":    `{"issues": []}`,
	}

	for name, content := range files {
		path := filepath.Join(indexDir, name)
		os.WriteFile(path, []byte(content), 0644)
	}

	result := analyzer.ValidateIndex(indexDir)
	if result.Valid {
		t.Error("Expected invalid result for schema mismatch")
	}
}

func TestFormatValidationErrors(t *testing.T) {
	// Test with valid result
	validResult := analyzer.ValidationResult{Valid: true, Errors: nil}
	if analyzer.FormatValidationErrors(validResult) != "" {
		t.Error("Expected empty string for valid result")
	}

	// Test with errors
	invalidResult := analyzer.ValidationResult{
		Valid:  false,
		Errors: []string{"error1", "error2"},
	}
	formatted := analyzer.FormatValidationErrors(invalidResult)
	if formatted == "" {
		t.Error("Expected non-empty string for invalid result")
	}
}
