package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitignoreMatcher_BasicPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .gitignore with basic patterns
	gitignoreContent := `*.log
build/
node_modules
`
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	matcher, err := NewGitignoreMatcher(tmpDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		ignored  bool
		describe string
	}{
		{filepath.Join(tmpDir, "debug.log"), true, "*.log should match"},
		{filepath.Join(tmpDir, "main.go"), false, "*.go should not match"},
		{filepath.Join(tmpDir, "build", "output.js"), true, "build/ should match"},
		{filepath.Join(tmpDir, "node_modules", "pkg", "index.js"), true, "node_modules should match"},
		{filepath.Join(tmpDir, "src", "app.js"), false, "src/ should not match"},
	}

	for _, tt := range tests {
		t.Run(tt.describe, func(t *testing.T) {
			result := matcher.Match(tt.path)
			if result != tt.ignored {
				t.Errorf("Match(%s) = %v, want %v", tt.path, result, tt.ignored)
			}
		})
	}
}

func TestGitignoreMatcher_NestedGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	// Root .gitignore
	rootGitignore := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(rootGitignore, []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create src directory with its own .gitignore
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcGitignore := filepath.Join(srcDir, ".gitignore")
	if err := os.WriteFile(srcGitignore, []byte("*.generated.go\n"), 0644); err != nil {
		t.Fatal(err)
	}

	matcher, err := NewGitignoreMatcher(tmpDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		ignored  bool
		describe string
	}{
		{filepath.Join(tmpDir, "debug.log"), true, "root *.log should match"},
		{filepath.Join(srcDir, "debug.log"), true, "root *.log should match in subdir"},
		{filepath.Join(srcDir, "api.generated.go"), true, "src *.generated.go should match"},
		{filepath.Join(srcDir, "main.go"), false, "src *.go should not match"},
		{filepath.Join(tmpDir, "types.generated.go"), false, "src pattern should not match in root"},
	}

	for _, tt := range tests {
		t.Run(tt.describe, func(t *testing.T) {
			result := matcher.Match(tt.path)
			if result != tt.ignored {
				t.Errorf("Match(%s) = %v, want %v", tt.path, result, tt.ignored)
			}
		})
	}
}

func TestGitignoreMatcher_NegationRules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore with negation rule
	gitignoreContent := `*.log
!important.log
`
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	matcher, err := NewGitignoreMatcher(tmpDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		ignored  bool
		describe string
	}{
		{filepath.Join(tmpDir, "debug.log"), true, "*.log should match"},
		{filepath.Join(tmpDir, "important.log"), false, "!important.log should NOT match (negated)"},
		{filepath.Join(tmpDir, "error.log"), true, "*.log should match"},
	}

	for _, tt := range tests {
		t.Run(tt.describe, func(t *testing.T) {
			result := matcher.Match(tt.path)
			if result != tt.ignored {
				t.Errorf("Match(%s) = %v, want %v", tt.path, result, tt.ignored)
			}
		})
	}
}

func TestGitignoreMatcher_GlobalPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// No .gitignore file, only global patterns
	globalPatterns := []string{".git", "*.tmp"}

	matcher, err := NewGitignoreMatcher(tmpDir, globalPatterns)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		ignored  bool
		describe string
	}{
		{filepath.Join(tmpDir, ".git", "config"), true, ".git should match"},
		{filepath.Join(tmpDir, "cache.tmp"), true, "*.tmp should match"},
		{filepath.Join(tmpDir, "main.go"), false, "main.go should not match"},
	}

	for _, tt := range tests {
		t.Run(tt.describe, func(t *testing.T) {
			result := matcher.Match(tt.path)
			if result != tt.ignored {
				t.Errorf("Match(%s) = %v, want %v", tt.path, result, tt.ignored)
			}
		})
	}
}

func TestGitignoreMatcher_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	matcher, err := NewGitignoreMatcher(tmpDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Empty gitignore should be skipped, so nothing matches
	path := filepath.Join(tmpDir, "test.log")
	if matcher.Match(path) {
		t.Errorf("Empty .gitignore should not match anything")
	}
}

func TestGitignoreMatcher_DynamicUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	matcher, err := NewGitignoreMatcher(tmpDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(tmpDir, "test.tmp")

	// Initially should not match
	if matcher.Match(testFile) {
		t.Errorf("Should not match before .gitignore is added")
	}

	// Add .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("*.tmp\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := matcher.AddGitignore(gitignorePath); err != nil {
		t.Fatal(err)
	}

	// Now should match
	if !matcher.Match(testFile) {
		t.Errorf("Should match after .gitignore is added")
	}

	// Remove .gitignore
	matcher.RemoveGitignore(gitignorePath)

	// Should not match again
	if matcher.Match(testFile) {
		t.Errorf("Should not match after .gitignore is removed")
	}
}
