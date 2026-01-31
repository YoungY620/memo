package mcp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YoungY620/memo/mcp"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantFile string
		wantSegs int
		wantErr  bool
	}{
		// Valid paths
		{"simple", "[arch]", "arch", 0, false},
		{"with key", "[arch][modules]", "arch", 1, false},
		{"with index", "[arch][modules][0]", "arch", 2, false},
		{"deep path", "[arch][modules][0][name]", "arch", 3, false},
		{"all files", "[interface][external]", "interface", 1, false},
		{"stories", "[stories][stories]", "stories", 1, false},
		{"issues", "[issues][issues]", "issues", 1, false},

		// Escape sequences
		{"escape bracket", "[arch][key\\[0\\]]", "arch", 1, false},
		{"escape backslash", "[arch][key\\\\name]", "arch", 1, false},

		// Invalid paths
		{"empty", "", "", 0, true},
		{"no brackets", "arch", "", 0, true},
		{"unclosed", "[arch", "", 0, true},
		{"double open", "[arch[[", "", 0, true},
		{"unexpected close", "arch]", "", 0, true},
		{"empty segment", "[arch][]", "", 0, true},
		{"invalid file", "[invalid][key]", "", 0, true},
		{"index first", "[0][key]", "", 0, true},
		{"trailing escape", "[arch][key\\", "", 0, true},
		{"invalid escape", "[arch][key\\x]", "", 0, true},
		{"control char", "[arch][key\x00]", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, segs, err := mcp.ParsePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if file != tt.wantFile {
					t.Errorf("ParsePath(%q) file = %v, want %v", tt.path, file, tt.wantFile)
				}
				if len(segs) != tt.wantSegs {
					t.Errorf("ParsePath(%q) segments = %d, want %d", tt.path, len(segs), tt.wantSegs)
				}
			}
		})
	}
}

func TestParsePathEscaping(t *testing.T) {
	// Test that escaped brackets are handled correctly
	_, segs, err := mcp.ParsePath("[arch][key\\[0\\]]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segs) != 1 || segs[0].Key != "key[0]" {
		t.Errorf("expected key 'key[0]', got %+v", segs)
	}

	// Test escaped backslash
	_, segs, err = mcp.ParsePath("[arch][path\\\\to\\\\file]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segs) != 1 || segs[0].Key != "path\\to\\file" {
		t.Errorf("expected key 'path\\to\\file', got %+v", segs)
	}
}

func TestKeyValidation(t *testing.T) {
	// Key too long
	longKey := "[arch][" + string(make([]byte, 101)) + "]"
	_, _, err := mcp.ParsePath(longKey)
	if err == nil {
		t.Error("expected error for long key")
	}
}

// Setup test index files
func setupTestIndex(t *testing.T) string {
	dir := t.TempDir()
	indexDir := filepath.Join(dir, ".memo", "index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"arch.json": `{
			"modules": [
				{"name": "main", "description": "entry point"},
				{"name": "config", "description": "configuration"}
			],
			"relationships": "main -> config"
		}`,
		"interface.json": `{
			"external": [{"type": "cli", "name": "--help"}],
			"internal": []
		}`,
		"stories.json": `{
			"stories": [{"title": "User Login", "tags": ["auth"], "content": "..."}]
		}`,
		"issues.json": `{
			"issues": [{"tags": ["todo"], "title": "Fix bug", "description": "...", "locations": []}]
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

func TestListKeys(t *testing.T) {
	indexDir := setupTestIndex(t)

	tests := []struct {
		name     string
		path     string
		wantType string
		wantLen  int
		wantErr  bool
	}{
		{"root dict", "[arch]", "dict", 2, false},
		{"array", "[arch][modules]", "list", 2, false},
		{"nested dict", "[arch][modules][0]", "dict", 2, false},
		{"invalid path", "[arch][nonexistent]", "", 0, true},
		{"out of bounds", "[arch][modules][99]", "", 0, true},
		{"scalar value", "[arch][modules][0][name]", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mcp.ListKeys(indexDir, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListKeys(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.Type != tt.wantType {
					t.Errorf("ListKeys(%q) type = %v, want %v", tt.path, result.Type, tt.wantType)
				}
				if result.Type == "list" && result.Length != tt.wantLen {
					t.Errorf("ListKeys(%q) length = %v, want %v", tt.path, result.Length, tt.wantLen)
				}
				if result.Type == "dict" && len(result.Keys) != tt.wantLen {
					t.Errorf("ListKeys(%q) keys count = %v, want %v", tt.path, len(result.Keys), tt.wantLen)
				}
			}
		})
	}
}

func TestGetValue(t *testing.T) {
	indexDir := setupTestIndex(t)

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"string value", "[arch][modules][0][name]", `"main"`, false},
		{"object value", "[arch][modules][0]", "", false}, // just check no error
		{"array value", "[arch][modules]", "", false},
		{"invalid path", "[arch][nonexistent]", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mcp.GetValue(indexDir, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValue(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.want != "" && result.Value != tt.want {
				t.Errorf("GetValue(%q) = %v, want %v", tt.path, result.Value, tt.want)
			}
		})
	}
}
