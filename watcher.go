package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	root           string
	ignorePatterns []string
	debounceMs     int
	onChange       func(files []string)
	watcher        *fsnotify.Watcher

	mu           sync.Mutex
	pendingFiles map[string]struct{}
	timer        *time.Timer
}

func NewWatcher(root string, ignorePatterns []string, debounceMs int, onChange func(files []string)) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		root:           root,
		ignorePatterns: ignorePatterns,
		debounceMs:     debounceMs,
		onChange:       onChange,
		watcher:        fsw,
		pendingFiles:   make(map[string]struct{}),
	}

	if err := w.addRecursive(root); err != nil {
		fsw.Close()
		return nil, err
	}

	return w, nil
}

func (w *Watcher) addRecursive(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if w.shouldIgnore(path) {
				return filepath.SkipDir
			}
			return w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) shouldIgnore(path string) bool {
	rel, _ := filepath.Rel(w.root, path)
	for _, pattern := range w.ignorePatterns {
		if strings.HasPrefix(pattern, "*.") {
			// extension match
			if strings.HasSuffix(path, pattern[1:]) {
				return true
			}
		} else {
			// directory/file name match
			if strings.Contains(rel, pattern) || filepath.Base(path) == pattern {
				return true
			}
		}
	}
	return false
}

func (w *Watcher) Run() error {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			if w.shouldIgnore(event.Name) {
				continue
			}
			// handle new directories
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.watcher.Add(event.Name)
				}
			}
			// collect changed files
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				w.addPending(event.Name)
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			if err != nil {
				// log but continue
				continue
			}
		}
	}
}

func (w *Watcher) addPending(file string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.pendingFiles[file] = struct{}{}

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, w.flush)
}

func (w *Watcher) flush() {
	w.mu.Lock()
	files := make([]string, 0, len(w.pendingFiles))
	for f := range w.pendingFiles {
		files = append(files, f)
	}
	w.pendingFiles = make(map[string]struct{})
	w.mu.Unlock()

	if len(files) > 0 && w.onChange != nil {
		w.onChange(files)
	}
}

func (w *Watcher) Close() error {
	return w.watcher.Close()
}
