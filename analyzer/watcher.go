package analyzer

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"sync"
	"time"

	"github.com/YoungY620/memo/internal"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	debounceMs, maxWaitMs int
	matcher               *GitignoreMatcher
	onChange              func([]string)
	watcher               *fsnotify.Watcher
	rootPath              string

	mu                sync.Mutex
	pending           map[string]struct{}
	debounce, maxWait *time.Timer
	sem               chan struct{} // capacity 1 semaphore for analysis guard
}

// describePathError extracts detailed information from filesystem errors (cross-platform)
func describePathError(err error) (op, path, reason string) {
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		return "", "", err.Error()
	}

	op = pathErr.Op
	path = pathErr.Path

	// Use cross-platform error checks from io/fs package
	switch {
	case errors.Is(err, fs.ErrNotExist):
		reason = "no such file or directory (broken symlink?)"
	case errors.Is(err, fs.ErrPermission):
		reason = "permission denied"
	case errors.Is(err, fs.ErrInvalid):
		reason = "invalid argument"
	case errors.Is(err, fs.ErrClosed):
		reason = "file already closed"
	default:
		reason = pathErr.Err.Error()
	}

	return op, path, reason
}

func NewWatcher(root string, globalPatterns []string, debounceMs, maxWaitMs int, onChange func([]string)) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Create gitignore matcher BEFORE watchAll
	matcher, err := NewGitignoreMatcher(root, globalPatterns)
	if err != nil {
		fsw.Close()
		return nil, err
	}

	w := &Watcher{
		rootPath:   root,
		matcher:    matcher,
		debounceMs: debounceMs,
		maxWaitMs:  maxWaitMs,
		onChange:   onChange,
		watcher:    fsw,
		pending:    make(map[string]struct{}),
		sem:        make(chan struct{}, 1),
	}
	if err := w.watchAll(root); err != nil {
		fsw.Close()
		return nil, err
	}
	return w, nil
}

func (w *Watcher) watchAll(dir string) error {
	return filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			op, path, reason := describePathError(err)
			internal.LogWarning("Skipping path: op=%s, path=%s, reason=%s", op, path, reason)
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if w.matcher.Match(p) {
			return filepath.SkipDir
		}
		if err := w.watcher.Add(p); err != nil {
			op, path, reason := describePathError(err)
			internal.LogWarning("Cannot add watch: op=%s, path=%s, reason=%s", op, path, reason)
		}
		return nil
	})
}

// ScanAll traverses all files and adds them to pending, triggering initial analysis
func (w *Watcher) ScanAll() {
	count := 0
	_ = filepath.WalkDir(w.rootPath, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if w.matcher.Match(p) {
			return nil
		}
		w.add(p)
		count++
		return nil
	})
	internal.LogDebug("ScanAll: added %d files to pending", count)
}



func (w *Watcher) Run() error {
	for {
		select {
		case e, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}

			// Handle .gitignore file changes
			if filepath.Base(e.Name) == ".gitignore" {
				if e.Op&fsnotify.Write != 0 || e.Op&fsnotify.Create != 0 {
					internal.LogInfo(".gitignore changed: %s", e.Name)
					if err := w.matcher.AddGitignore(e.Name); err != nil {
						internal.LogWarning("Failed to reload .gitignore: %v", err)
					}
				} else if e.Op&fsnotify.Remove != 0 {
					w.matcher.RemoveGitignore(e.Name)
				}
				continue
			}

			if w.matcher.Match(e.Name) {
				continue
			}
			internal.LogDebug("Event: %s %s", e.Op, e.Name)
			if e.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(e.Name); err == nil && info.IsDir() {
					internal.LogDebug("Watching new directory: %s", e.Name)
					_ = w.watcher.Add(e.Name)
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
