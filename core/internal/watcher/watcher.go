// Package watcher provides file monitoring functionality
package watcher

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/config"
)

// EventType file change event type
type EventType int

const (
	EventCreate EventType = iota
	EventModify
	EventDelete
	EventRename
)

func (e EventType) String() string {
	switch e {
	case EventCreate:
		return "create"
	case EventModify:
		return "modify"
	case EventDelete:
		return "delete"
	case EventRename:
		return "rename"
	default:
		return "unknown"
	}
}

// Event file change event
type Event struct {
	Path string
	Type EventType
}

// Watcher file monitor
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	cfg       *config.WatcherConfig
	events    chan Event
	done      chan struct{}
	rootPath  string
	ignoreMap map[string]bool
	extMap    map[string]bool
}

// New creates a new file watcher
func New(cfg *config.WatcherConfig) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	rootPath, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher: fsWatcher,
		cfg:       cfg,
		events:    make(chan Event, 100),
		done:      make(chan struct{}),
		rootPath:  rootPath,
		ignoreMap: make(map[string]bool),
		extMap:    make(map[string]bool),
	}

	// Build ignore map
	for _, pattern := range cfg.Ignore {
		w.ignoreMap[pattern] = true
	}

	// Build extension map
	for _, ext := range cfg.Extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		w.extMap[ext] = true
	}

	return w, nil
}

// Start starts monitoring
func (w *Watcher) Start() error {
	// Recursively add directories
	err := filepath.Walk(w.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore inaccessible paths
		}
		if info.IsDir() {
			if w.shouldIgnore(path) {
				return filepath.SkipDir
			}
			return w.fsWatcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Start event processing
	go w.loop()
	return nil
}

// Events returns the event channel
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Stop stops monitoring
func (w *Watcher) Stop() error {
	close(w.done)
	return w.fsWatcher.Close()
}

// loop event processing loop
func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			close(w.events)
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case _, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			// Ignore errors, continue monitoring
		}
	}
}

// handleEvent handles a single fsnotify event
func (w *Watcher) handleEvent(e fsnotify.Event) {
	path := e.Name

	// Check if should ignore
	if w.shouldIgnore(path) {
		return
	}

	// Check extension
	if !w.shouldWatch(path) {
		return
	}

	var eventType EventType
	switch {
	case e.Op&fsnotify.Create != 0:
		eventType = EventCreate
		// If directory, add to watch
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			_ = w.fsWatcher.Add(path)
			return // Don't send directory create event
		}
	case e.Op&fsnotify.Write != 0:
		eventType = EventModify
	case e.Op&fsnotify.Remove != 0:
		eventType = EventDelete
	case e.Op&fsnotify.Rename != 0:
		eventType = EventRename
	default:
		return
	}

	// Send event
	select {
	case w.events <- Event{Path: path, Type: eventType}:
	default:
		// Channel full, discard event
	}
}

// shouldIgnore checks if path should be ignored
func (w *Watcher) shouldIgnore(path string) bool {
	relPath, err := filepath.Rel(w.rootPath, path)
	if err != nil {
		return true
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	for _, part := range parts {
		if w.ignoreMap[part] {
			return true
		}
	}
	return false
}

// shouldWatch checks if file should be monitored (based on extension)
func (w *Watcher) shouldWatch(path string) bool {
	// If no extensions configured, monitor all files
	if len(w.extMap) == 0 {
		return true
	}

	ext := strings.ToLower(filepath.Ext(path))
	return w.extMap[ext]
}
