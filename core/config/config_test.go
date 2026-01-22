package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsDefault(t *testing.T) {
	cfg, err := Load("does-not-exist.yaml")
	if err != nil {
		t.Fatalf("load default: %v", err)
	}
	if cfg.Watcher.Root != "." {
		t.Fatalf("expected default watcher root '.', got %q", cfg.Watcher.Root)
	}
	if cfg.Index.Path != ".kimi-indexer" {
		t.Fatalf("expected default index path '.kimi-indexer', got %q", cfg.Index.Path)
	}
}

func TestLoadAndNormalize(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(cfgPath, []byte(`
watcher:
  root: repo
  ignoreGlobs: [".git"]
index:
  path: ".custom-index"
schemaDir: schema-files
`), 0o644)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := cfg.Normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	if !filepath.IsAbs(cfg.Watcher.Root) {
		t.Fatalf("root not absolute: %s", cfg.Watcher.Root)
	}
	if !filepath.IsAbs(cfg.Index.Path) {
		t.Fatalf("index not absolute: %s", cfg.Index.Path)
	}
	if !filepath.IsAbs(cfg.SchemaDir) {
		t.Fatalf("schemaDir not absolute: %s", cfg.SchemaDir)
	}
}
