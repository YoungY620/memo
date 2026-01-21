// Package watcher provides file watching functionality
package watcher

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/config"
)

// EventType file changed事件类型
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

// Event file changed事件
type Event struct {
	Path string
	Type EventType
}

// Watcher 文件监控器
type Watcher struct {
	fsWatcher  *fsnotify.Watcher
	cfg        *config.WatcherConfig
	events     chan Event
	done       chan struct{}
	rootPath   string
	ignoreMap  map[string]bool
	extMap     map[string]bool
}

// New 创建新的文件监控器
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

	// 构建忽略映射
	for _, pattern := range cfg.Ignore {
		w.ignoreMap[pattern] = true
	}

	// 构建扩展名映射
	for _, ext := range cfg.Extensions {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		w.extMap[ext] = true
	}

	return w, nil
}

// Start 启动监控
func (w *Watcher) Start() error {
	// 递归添加目录
	err := filepath.Walk(w.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略无法访问的路径
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

	// 启动事件处理
	go w.loop()
	return nil
}

// Events 返回事件通道
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Stop 停止监控
func (w *Watcher) Stop() error {
	close(w.done)
	return w.fsWatcher.Close()
}

// loop 事件处理循环
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
			// 忽略错误，继续监控
		}
	}
}

// handleEvent 处理单个 fsnotify 事件
func (w *Watcher) handleEvent(e fsnotify.Event) {
	path := e.Name

	// 检查是否应该忽略
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
		// 如果是目录，添加监控
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			_ = w.fsWatcher.Add(path)
			return // 不发送目录创建事件
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

	// 发送事件
	select {
	case w.events <- Event{Path: path, Type: eventType}:
	default:
		// 通道满了，丢弃事件
	}
}

// shouldIgnore 检查路径是否应该被忽略
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

// shouldWatch 检查文件是否应该被监控（基于扩展名）
func (w *Watcher) shouldWatch(path string) bool {
	// 如果没有配置扩展名，监控所有文件
	if len(w.extMap) == 0 {
		return true
	}

	ext := strings.ToLower(filepath.Ext(path))
	return w.extMap[ext]
}
