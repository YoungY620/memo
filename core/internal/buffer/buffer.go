// Package buffer provides change buffering functionality
package buffer

import (
	"sync"

	"github.com/user/kimi-sdk-agent-indexer/core/internal/watcher"
)

// ChangeType 变更类型
type ChangeType int

const (
	ChangeCreate ChangeType = iota
	ChangeModify
	ChangeDelete
)

func (c ChangeType) String() string {
	switch c {
	case ChangeCreate:
		return "create"
	case ChangeModify:
		return "modify"
	case ChangeDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Change 单个file changed
type Change struct {
	Path string
	Type ChangeType
}

// Buffer 变更缓冲区
type Buffer struct {
	changes map[string]ChangeType // 以路径为 key
	mu      sync.RWMutex
}

// New 创建新的变更缓冲区
func New() *Buffer {
	return &Buffer{
		changes: make(map[string]ChangeType),
	}
}

// Add 添加变更事件
func (b *Buffer) Add(event watcher.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	path := event.Path
	newType := eventToChangeType(event.Type)

	// 检查是否已有该文件的变更记录
	if oldType, exists := b.changes[path]; exists {
		// 合并变更
		merged := mergeChanges(oldType, newType)
		if merged == -1 {
			// create + delete = 移除
			delete(b.changes, path)
		} else {
			b.changes[path] = merged
		}
	} else {
		b.changes[path] = newType
	}
}

// Count 返回缓冲区中的变更文件数
func (b *Buffer) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.changes)
}

// Flush 取出并清空所有变更
func (b *Buffer) Flush() []Change {
	b.mu.Lock()
	defer b.mu.Unlock()

	changes := make([]Change, 0, len(b.changes))
	for path, changeType := range b.changes {
		changes = append(changes, Change{
			Path: path,
			Type: changeType,
		})
	}

	// 清空缓冲区
	b.changes = make(map[string]ChangeType)
	return changes
}

// IsEmpty 检查缓冲区是否为空
func (b *Buffer) IsEmpty() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.changes) == 0
}

// eventToChangeType 将 watcher 事件类型转换为变更类型
func eventToChangeType(e watcher.EventType) ChangeType {
	switch e {
	case watcher.EventCreate:
		return ChangeCreate
	case watcher.EventModify:
		return ChangeModify
	case watcher.EventDelete, watcher.EventRename:
		return ChangeDelete
	default:
		return ChangeModify
	}
}

// mergeChanges 合并两个变更类型
// 返回 -1 表示应该移除这条记录
func mergeChanges(old, new ChangeType) ChangeType {
	/*
		合并规则：
		| 旧事件 | 新事件 | 结果 |
		|--------|--------|------|
		| create | modify | create |
		| create | delete | 移除 |
		| modify | modify | modify |
		| modify | delete | delete |
	*/
	switch {
	case old == ChangeCreate && new == ChangeModify:
		return ChangeCreate
	case old == ChangeCreate && new == ChangeDelete:
		return -1 // 移除
	case old == ChangeModify && new == ChangeModify:
		return ChangeModify
	case old == ChangeModify && new == ChangeDelete:
		return ChangeDelete
	case old == ChangeDelete && new == ChangeCreate:
		return ChangeModify // delete + create = modify
	default:
		return new
	}
}
