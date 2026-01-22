package watcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Op represents a simplified fsnotify operation.
type Op string

const (
	OpCreate Op = "create"
	OpWrite  Op = "write"
	OpRemove Op = "remove"
	OpRename Op = "rename"
	OpChmod  Op = "chmod"
)

// Event describes a change detected by the watcher.
type Event struct {
	Path string
	Op   Op
}

// Config contains watcher configuration.
type Config struct {
	Root        string
	IgnoreGlobs []string
	Extensions  []string
}

// Watcher monitors a directory tree and forwards events to subscribers.
type Watcher struct {
	cfg       Config
	fs        *fsnotify.Watcher
	events    chan Event
	done      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once

	ignoreMatchers []globMatcher
	extFilter      map[string]struct{}
}

// New returns a prepared Watcher.
func New(cfg Config) (*Watcher, error) {
	if cfg.Root == "" {
		return nil, errors.New("watcher: root path is required")
	}
	rootAbs, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, fmt.Errorf("watcher: resolve root: %w", err)
	}
	cfg.Root = rootAbs

	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("watcher: create fsnotify: %w", err)
	}

	w := &Watcher{
		cfg:    cfg,
		fs:     fs,
		events: make(chan Event, 128),
		done:   make(chan struct{}),
	}
	for _, glob := range cfg.IgnoreGlobs {
		if glob == "" {
			continue
		}
		w.ignoreMatchers = append(w.ignoreMatchers, newGlobMatcher(glob))
	}
	if len(cfg.Extensions) > 0 {
		w.extFilter = make(map[string]struct{}, len(cfg.Extensions))
		for _, ext := range cfg.Extensions {
			if ext == "" {
				continue
			}
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			w.extFilter[strings.ToLower(ext)] = struct{}{}
		}
	}
	return w, nil
}

// Start begins monitoring in the background.
func (w *Watcher) Start() error {
	var startErr error
	w.startOnce.Do(func() {
		if err := w.walkAndWatch(w.cfg.Root); err != nil {
			startErr = err
			return
		}
		go w.loop()
	})
	return startErr
}

// Events returns the event channel.
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Stop releases resources.
func (w *Watcher) Stop() error {
	var stopErr error
	w.stopOnce.Do(func() {
		close(w.done)
		stopErr = w.fs.Close()
	})
	return stopErr
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			close(w.events)
			return
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			w.process(ev)
		case <-w.fs.Errors:
			// Ignore fsnotify internal errors; loop continues.
		}
	}
}

func (w *Watcher) process(ev fsnotify.Event) {
	path := ev.Name
	if path == "" {
		return
	}

	if w.shouldIgnore(path) {
		return
	}
	if !w.shouldProcess(path) {
		return
	}

	op := normalizeOp(ev.Op)
	if op == "" {
		return
	}

	// Add newly created directories to watcher.
	if op == OpCreate {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			_ = w.walkAndWatch(path)
			return
		}
	}

	select {
	case w.events <- Event{Path: path, Op: op}:
	default:
	}
}

func (w *Watcher) walkAndWatch(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if d.IsDir() {
			if w.shouldIgnore(path) {
				return filepath.SkipDir
			}
			if watchErr := w.fs.Add(path); watchErr != nil {
				return nil
			}
		}
		return nil
	})
}

func (w *Watcher) shouldIgnore(path string) bool {
	rel, err := filepath.Rel(w.cfg.Root, path)
	if err != nil {
		return true
	}
	rel = filepath.ToSlash(rel)

	for _, matcher := range w.ignoreMatchers {
		if matcher.Match(rel) {
			return true
		}
	}
	return false
}

func (w *Watcher) shouldProcess(path string) bool {
	if w.extFilter == nil {
		return true
	}
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	if info.IsDir() {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := w.extFilter[ext]
	return ok
}

func normalizeOp(op fsnotify.Op) Op {
	switch {
	case op&fsnotify.Create == fsnotify.Create:
		return OpCreate
	case op&fsnotify.Write == fsnotify.Write:
		return OpWrite
	case op&fsnotify.Remove == fsnotify.Remove:
		return OpRemove
	case op&fsnotify.Rename == fsnotify.Rename:
		return OpRename
	case op&fsnotify.Chmod == fsnotify.Chmod:
		return OpChmod
	default:
		return ""
	}
}

// globMatcher provides minimal glob support (`*` and `**`).
type globMatcher struct {
	raw      string
	segments []string
}

func newGlobMatcher(raw string) globMatcher {
	raw = filepath.ToSlash(strings.TrimSpace(raw))
	return globMatcher{
		raw:      raw,
		segments: splitGlob(raw),
	}
}

func splitGlob(glob string) []string {
	if glob == "" {
		return nil
	}
	return strings.Split(glob, "/")
}

func (m globMatcher) Match(path string) bool {
	if m.raw == "" {
		return false
	}
	path = filepath.ToSlash(path)
	pathSegs := strings.Split(path, "/")
	return matchSegments(pathSegs, m.segments)
}

func matchSegments(pathSegs, pattern []string) bool {
	if len(pattern) == 0 {
		return len(pathSegs) == 0
	}
	head := pattern[0]
	switch head {
	case "**":
		if len(pattern) == 1 {
			return true
		}
		for i := 0; i <= len(pathSegs); i++ {
			if matchSegments(pathSegs[i:], pattern[1:]) {
				return true
			}
		}
		return false
	case "*":
		if len(pathSegs) == 0 {
			return false
		}
		return matchSegments(pathSegs[1:], pattern[1:])
	default:
		if len(pathSegs) == 0 {
			return false
		}
		if !strings.EqualFold(head, pathSegs[0]) {
			return false
		}
		return matchSegments(pathSegs[1:], pattern[1:])
	}
}
