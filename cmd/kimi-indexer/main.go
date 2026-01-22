package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/user/kimi-sdk-agent-indexer/core/config"
	"github.com/user/kimi-sdk-agent-indexer/core/logging"
)

func main() {
	var (
		configPath     = flag.String("config", ".kimi-indexer.yaml", "Path to configuration file")
		rootOverride   = flag.String("root", "", "Override watcher root directory")
		indexOverride  = flag.String("index", "", "Override index output directory (relative to root if not absolute)")
		schemaOverride = flag.String("schemas", "", "Override schema directory")
		printOnly      = flag.Bool("print-config", false, "Print resolved configuration and exit")
	)
	flag.Parse()

	logger := logging.New(logging.WithLevel(logging.LevelInfo))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Errorf("failed to load config: %v", err)
		os.Exit(1)
	}

	cfg.ApplyOverrides(*rootOverride, *indexOverride, *schemaOverride)

	if err := cfg.Normalize(); err != nil {
		logger.Errorf("failed to normalize config: %v", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		logger.Errorf("invalid configuration: %v", err)
		os.Exit(1)
	}

	if *printOnly {
		fmt.Print(cfg.PrettyYAML())
		return
	}

	logger.Infof("configuration loaded successfully")
	logger.Debugf("resolved configuration:\n%s", cfg.PrettyYAML())
	logger.Infof("watcher root: %s", cfg.Watcher.Root)
	logger.Infof("index path: %s", cfg.Index.Path)
	logger.Infof("schema directory: %s", cfg.SchemaDir)
	logger.Warnf("no session factory configured; watch service is not started in this build")
}
