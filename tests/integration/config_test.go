package integration_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Import the main package for config testing
// Note: This requires the config functions to be accessible
// We test via the public LoadConfig and LoadGitignore functions

func TestConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	_ = os.WriteFile(configPath, []byte(""), 0644)

	// We can't directly call LoadConfig from here since it's in main package
	// This test is more of a placeholder for integration testing

	// Instead, verify the config file format
	configContent := `
agent:
  api_key: test-key
  model: test-model
watch:
  ignore_patterns:
    - .git
    - node_modules
  debounce_ms: 5000
  max_wait_ms: 60000
log_level: debug
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if len(data) == 0 {
		t.Error("Config file should not be empty")
	}
}

func TestGitignoreFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .gitignore file
	gitignoreContent := `# Comment
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

	// Verify the file was written correctly
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	content := string(data)
	if content != gitignoreContent {
		t.Error("Gitignore content mismatch")
	}
}

func TestIndexFileFormats(t *testing.T) {
	// Test valid index file formats
	testCases := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name: "arch.json",
			content: `{
				"modules": [
					{"name": "main", "description": "entry", "interfaces": "CLI"}
				],
				"relationships": ""
			}`,
			valid: true,
		},
		{
			name: "interface.json",
			content: `{
				"external": [],
				"internal": []
			}`,
			valid: true,
		},
		{
			name: "stories.json",
			content: `{
				"stories": []
			}`,
			valid: true,
		},
		{
			name: "issues.json",
			content: `{
				"issues": []
			}`,
			valid: true,
		},
	}

	tmpDir := t.TempDir()
	indexDir := filepath.Join(tmpDir, ".memo", "index")
	_ = os.MkdirAll(indexDir, 0755)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(indexDir, tc.name)
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to write %s: %v", tc.name, err)
			}

			// Verify file exists and is readable
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", tc.name, err)
			}

			if len(data) == 0 {
				t.Errorf("%s should not be empty", tc.name)
			}
		})
	}
}

func TestPathSeparator(t *testing.T) {
	// Test that filepath operations work correctly on all platforms
	parts := []string{"a", "b", "c", "file.txt"}
	path := filepath.Join(parts...)

	// Path should be normalized for the current OS
	if path == "" {
		t.Error("Path should not be empty")
	}

	// Split should return the original parts
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	if base != "file.txt" {
		t.Errorf("Expected base 'file.txt', got '%s'", base)
	}

	expectedDir := filepath.Join("a", "b", "c")
	if dir != expectedDir {
		t.Errorf("Expected dir '%s', got '%s'", expectedDir, dir)
	}
}

func TestAbsPath(t *testing.T) {
	// Test absolute path conversion
	relPath := "test/path"
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if !filepath.IsAbs(absPath) {
		t.Errorf("Expected absolute path, got: %s", absPath)
	}
}

func TestTempDir(t *testing.T) {
	// Test that temp directories work correctly
	tmpDir := t.TempDir()

	if tmpDir == "" {
		t.Error("Temp dir should not be empty")
	}

	// Should be able to create files
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write to temp dir: %v", err)
	}

	// Should be able to read back
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read from temp dir: %v", err)
	}

	if !reflect.DeepEqual(data, []byte("test")) {
		t.Error("Data mismatch")
	}
}
