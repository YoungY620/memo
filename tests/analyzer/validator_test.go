package analyzer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YoungY620/memo/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Empty(t, analyzer.FormatValidationErrors(validResult))

	// Test with errors
	invalidResult := analyzer.ValidationResult{
		Valid:  false,
		Errors: []string{"error1", "error2"},
	}
	formatted := analyzer.FormatValidationErrors(invalidResult)
	assert.NotEmpty(t, formatted)
	assert.Contains(t, formatted, "error1")
	assert.Contains(t, formatted, "error2")
}

func TestValidateIndex_DirNotExist(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "nonexistent", "index")

	result := analyzer.ValidateIndex(nonExistentDir)
	assert.False(t, result.Valid, "Should be invalid for non-existent directory")
	assert.NotEmpty(t, result.Errors)
}

func TestValidateIndex_AllFilesSchema(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		content string
		valid   bool
	}{
		{
			name:    "arch.json valid",
			file:    "arch.json",
			content: `{"modules": [{"name": "test", "description": "test", "interfaces": "none"}], "relationships": ""}`,
			valid:   true,
		},
		{
			name:    "arch.json missing modules",
			file:    "arch.json",
			content: `{"relationships": ""}`,
			valid:   false,
		},
		{
			name:    "interface.json valid",
			file:    "interface.json",
			content: `{"external": [], "internal": []}`,
			valid:   true,
		},
		{
			name:    "interface.json missing external",
			file:    "interface.json",
			content: `{"internal": []}`,
			valid:   false,
		},
		{
			name:    "stories.json valid",
			file:    "stories.json",
			content: `{"stories": []}`,
			valid:   true,
		},
		{
			name:    "stories.json missing stories",
			file:    "stories.json",
			content: `{}`,
			valid:   false,
		},
		{
			name:    "issues.json valid",
			file:    "issues.json",
			content: `{"issues": []}`,
			valid:   true,
		},
		{
			name:    "issues.json missing issues",
			file:    "issues.json",
			content: `{"other": []}`,
			valid:   false,
		},
	}

	baseFiles := map[string]string{
		"arch.json":      `{"modules": [], "relationships": ""}`,
		"interface.json": `{"external": [], "internal": []}`,
		"stories.json":   `{"stories": []}`,
		"issues.json":    `{"issues": []}`,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			indexDir := filepath.Join(dir, ".memo", "index")
			require.NoError(t, os.MkdirAll(indexDir, 0755))

			// Write all base files
			for name, content := range baseFiles {
				path := filepath.Join(indexDir, name)
				require.NoError(t, os.WriteFile(path, []byte(content), 0644))
			}

			// Override the specific file being tested
			testPath := filepath.Join(indexDir, tt.file)
			require.NoError(t, os.WriteFile(testPath, []byte(tt.content), 0644))

			result := analyzer.ValidateIndex(indexDir)

			if tt.valid {
				assert.True(t, result.Valid, "Expected valid for %s: %v", tt.name, result.Errors)
			} else {
				assert.False(t, result.Valid, "Expected invalid for %s", tt.name)
				// Error should mention the file
				errStr := strings.Join(result.Errors, "\n")
				assert.Contains(t, errStr, tt.file)
			}
		})
	}
}

func TestValidateIndex_EmptyFiles(t *testing.T) {
	dir := t.TempDir()
	indexDir := filepath.Join(dir, ".memo", "index")
	require.NoError(t, os.MkdirAll(indexDir, 0755))

	// Create empty files
	files := []string{"arch.json", "interface.json", "stories.json", "issues.json"}
	for _, f := range files {
		require.NoError(t, os.WriteFile(filepath.Join(indexDir, f), []byte(""), 0644))
	}

	result := analyzer.ValidateIndex(indexDir)
	assert.False(t, result.Valid, "Empty files should be invalid")
}
