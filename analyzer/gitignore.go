package analyzer

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/YoungY620/memo/internal"
	ignore "github.com/sabhiram/go-gitignore"
)

// GitignoreMatcher handles nested .gitignore files like Git does
type GitignoreMatcher struct {
	rootPath     string
	globalIgnore *ignore.GitIgnore
	dirIgnores   map[string]*ignore.GitIgnore
	mu           sync.RWMutex
}

// NewGitignoreMatcher creates a new matcher and scans all .gitignore files
func NewGitignoreMatcher(root string, globalPatterns []string) (*GitignoreMatcher, error) {
	m := &GitignoreMatcher{
		rootPath:   root,
		dirIgnores: make(map[string]*ignore.GitIgnore),
	}

	// Create global matcher from config patterns
	if len(globalPatterns) > 0 {
		m.globalIgnore = ignore.CompileIgnoreLines(globalPatterns...)
	}

	// Scan all .gitignore files in the repository
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			internal.LogWarning("Skipping path %s: %v", path, err)
			return nil
		}

		// Only process files named .gitignore
		if !d.IsDir() && d.Name() == ".gitignore" {
			// Skip empty files
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if info.Size() == 0 {
				internal.LogDebug("Skipping empty .gitignore: %s", path)
				return nil
			}

			// Compile this .gitignore file
			gi, err := ignore.CompileIgnoreFile(path)
			if err != nil {
				internal.LogWarning("Failed to parse %s: %v", path, err)
				return nil
			}

			// Store in map (key is directory path)
			dir := filepath.Dir(path)
			m.dirIgnores[dir] = gi
			internal.LogDebug("Loaded .gitignore: %s", path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	internal.LogInfo("Loaded %d .gitignore files", len(m.dirIgnores))
	return m, nil
}

// Match checks if a path should be ignored
// Following Git's rules:
// 1. Global patterns (from config.yaml) always apply
// 2. Walk from file directory up to root, checking each directory's .gitignore
// 3. Each .gitignore's rules are matched relative to its directory
func (m *GitignoreMatcher) Match(absPath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check global patterns first
	if m.globalIgnore != nil {
		relPath, err := filepath.Rel(m.rootPath, absPath)
		if err == nil && m.globalIgnore.MatchesPath(relPath) {
			return true
		}
	}

	// Walk from file directory up to root
	dir := filepath.Dir(absPath)
	for {
		// Check if current directory has a .gitignore
		if gi, ok := m.dirIgnores[dir]; ok {
			// Calculate path relative to this .gitignore
			relPath, err := filepath.Rel(dir, absPath)
			if err == nil && gi.MatchesPath(relPath) {
				return true
			}
		}

		// Reached root directory, stop
		if dir == m.rootPath {
			break
		}

		// Move up one level
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return false
}

// AddGitignore dynamically adds or updates a .gitignore file
func (m *GitignoreMatcher) AddGitignore(gitignorePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	gi, err := ignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(gitignorePath)
	m.dirIgnores[dir] = gi

	internal.LogInfo("Reloaded .gitignore: %s", gitignorePath)
	return nil
}

// RemoveGitignore removes a deleted .gitignore file from the matcher
func (m *GitignoreMatcher) RemoveGitignore(gitignorePath string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir := filepath.Dir(gitignorePath)
	delete(m.dirIgnores, dir)

	internal.LogInfo("Removed .gitignore: %s", gitignorePath)
}
