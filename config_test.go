package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadGitignore(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitignore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .gitignore file
	gitignoreContent := `# Comment line
dist/
build/
*.exe

# Dependencies
node_modules/
vendor/

!negation_should_be_ignored

# With leading slash
/root_only

*.log
`
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write .gitignore: %v", err)
	}

	// Test LoadGitignore
	patterns, err := LoadGitignore(tmpDir)
	if err != nil {
		t.Fatalf("LoadGitignore failed: %v", err)
	}

	expected := []string{
		"dist",
		"build",
		"*.exe",
		"node_modules",
		"vendor",
		"root_only",
		"*.log",
	}

	if !reflect.DeepEqual(patterns, expected) {
		t.Errorf("Patterns mismatch.\nGot:      %v\nExpected: %v", patterns, expected)
	}
}

func TestLoadGitignore_NoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gitignore_test_empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// No .gitignore file
	patterns, err := LoadGitignore(tmpDir)
	if err != nil {
		t.Fatalf("LoadGitignore should not error on missing file: %v", err)
	}
	if patterns != nil {
		t.Errorf("Expected nil patterns for missing .gitignore, got: %v", patterns)
	}
}

func TestMergeGitignore(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "merge_gitignore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .gitignore file
	gitignoreContent := `dist/
node_modules/
*.log
newpattern/
`
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write .gitignore: %v", err)
	}

	// Create config with some existing patterns
	cfg := &Config{
		Watch: WatchConfig{
			IgnorePatterns: []string{".git", "node_modules", ".memo", "*.log"},
		},
	}

	// Merge
	if err := cfg.MergeGitignore(tmpDir); err != nil {
		t.Fatalf("MergeGitignore failed: %v", err)
	}

	// Check that new patterns were added but duplicates were not
	expected := []string{".git", "node_modules", ".memo", "*.log", "dist", "newpattern"}
	if !reflect.DeepEqual(cfg.Watch.IgnorePatterns, expected) {
		t.Errorf("Patterns mismatch.\nGot:      %v\nExpected: %v", cfg.Watch.IgnorePatterns, expected)
	}
}
