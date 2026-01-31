package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/YoungY620/memo/internal"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	debounceMs, maxWaitMs int
	ignorePatterns        []string
	onChange              func([]string)
	watcher               *fsnotify.Watcher
	rootPath              string

	mu                sync.Mutex
	pending           map[string]struct{}
	debounce, maxWait *time.Timer
	sem               chan struct{} // capacity 1 semaphore for analysis guard
}

func NewWatcher(root string, ignore []string, debounceMs, maxWaitMs int, onChange func([]string)) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		rootPath:       root,
		ignorePatterns: ignore,
		debounceMs:     debounceMs,
		maxWaitMs:      maxWaitMs,
		onChange:       onChange,
		watcher:        fsw,
		pending:        make(map[string]struct{}),
		sem:            make(chan struct{}, 1),
	}
	if err := w.watchAll(root); err != nil {
		fsw.Close()
		return nil, err
	}
	return w, nil
}

func (w *Watcher) watchAll(dir string) error {
	return filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if w.ignored(p) {
			return filepath.SkipDir
		}
		return w.watcher.Add(p)
	})
}

// ScanAll traverses all files and adds them to pending, triggering initial analysis
func (w *Watcher) ScanAll() {
	count := 0
	filepath.WalkDir(w.rootPath, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if w.ignored(p) {
			return nil
		}
		w.add(p)
		count++
		return nil
	})
	internal.LogDebug("ScanAll: added %d files to pending", count)
}

func (w *Watcher) ignored(path string) bool {
	rel, _ := filepath.Rel(w.rootPath, path)
	base := filepath.Base(path)
	for _, p := range w.ignorePatterns {
		if strings.HasPrefix(p, "*.") && strings.HasSuffix(path, p[1:]) {
			return true
		}
		if strings.Contains(rel, p) || base == p {
			return true
		}
	}
	return false
}

func (w *Watcher) Run() error {
	for {
		select {
		case e, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			if w.ignored(e.Name) {
				continue
			}
			internal.LogDebug("Event: %s %s", e.Op, e.Name)
			if e.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(e.Name); err == nil && info.IsDir() {
					internal.LogDebug("Watching new directory: %s", e.Name)
					w.watcher.Add(e.Name)
				}
			}
			if e.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				w.add(e.Name)
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			if err != nil {
				internal.LogError("Watcher error: %v", err)
			}
		}
	}
}

func (w *Watcher) add(file string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	first := len(w.pending) == 0
	w.pending[file] = struct{}{}

	// Reset debounce timer
	if w.debounce != nil {
		w.debounce.Stop()
	}
	w.debounce = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, w.Flush)

	// Start max wait timer on first change
	if first {
		w.maxWait = time.AfterFunc(time.Duration(w.maxWaitMs)*time.Millisecond, w.Flush)
	}
}

func (w *Watcher) Flush() {
	// Non-blocking acquire: skip if analysis already running
	select {
	case w.sem <- struct{}{}:
		// acquired
	default:
		internal.LogDebug("Analysis in progress, skipping flush (files remain in pending)")
		return
	}
	defer func() { <-w.sem }()

	w.mu.Lock()
	if w.debounce != nil {
		w.debounce.Stop()
		w.debounce = nil
	}
	if w.maxWait != nil {
		w.maxWait.Stop()
		w.maxWait = nil
	}
	files := make([]string, 0, len(w.pending))
	for f := range w.pending {
		files = append(files, f)
	}
	w.pending = make(map[string]struct{})
	w.mu.Unlock()

	if len(files) > 0 && w.onChange != nil {
		w.onChange(files)
	}
}

func (w *Watcher) Close() error {
	w.mu.Lock()
	if w.debounce != nil {
		w.debounce.Stop()
	}
	if w.maxWait != nil {
		w.maxWait.Stop()
	}
	w.mu.Unlock()
	return w.watcher.Close()
}
