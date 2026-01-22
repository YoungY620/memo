package prompts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemplatesMatchDocs(t *testing.T) {
	root := findRepoRoot(t)

	tests := []struct {
		name      string
		assetPath string
		docPath   string
		canonical func() string
	}{
		{
			name:      "initialize",
			assetPath: "core/internal/prompts/assets/initialize.md",
			docPath:   "docs/core/prompts/initialize.md",
			canonical: InitializeTemplate,
		},
		{
			name:      "watch",
			assetPath: "core/internal/prompts/assets/watch.md",
			docPath:   "docs/core/prompts/watch.md",
			canonical: WatchTemplate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			asset := readFile(t, filepath.Join(root, tc.assetPath))
			doc := readFile(t, filepath.Join(root, tc.docPath))

			if asset != doc {
				t.Fatalf("asset and doc differ for %s", tc.name)
			}
			if raw := tc.canonical(); raw != asset {
				t.Fatalf("embedded template mismatch for %s", tc.name)
			}
		})
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("pwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("go.mod not found from %s", dir)
	return ""
}
