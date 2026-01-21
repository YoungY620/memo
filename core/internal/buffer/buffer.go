// Package buffer provides change buffering functionality
package buffer

import (
	"sync"

	"github.com/user/kimi-sdk-agent-indexer/core/internal/watcher"
)

// ChangeType change type
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

// Change single file change
type Change struct {
	Path string
	Type ChangeType
}

// Buffer change buffer
type Buffer struct {
	changes map[string]ChangeType // keyed by path
	mu      sync.RWMutex
}

// New creates a new change buffer
func New() *Buffer {
	return &Buffer{
		changes: make(map[string]ChangeType),
	}
}

// Add adds a change event
func (b *Buffer) Add(event watcher.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	path := event.Path
	newType := eventToChangeType(event.Type)

	// Check if there's existing change record for this file
	if oldType, exists := b.changes[path]; exists {
		// Merge changes
		merged := mergeChanges(oldType, newType)
		if merged == -1 {
			// create + delete = remove
			delete(b.changes, path)
		} else {
			b.changes[path] = merged
		}
	} else {
		b.changes[path] = newType
	}
}

// Count returns the number of changed files in buffer
func (b *Buffer) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.changes)
}

// Flush retrieves and clears all changes
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

	// Clear buffer
	b.changes = make(map[string]ChangeType)
	return changes
}

// IsEmpty checks if buffer is empty
func (b *Buffer) IsEmpty() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.changes) == 0
}

// eventToChangeType converts watcher event type to change type
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

// mergeChanges merges two change types
// Returns -1 if the record should be removed
func mergeChanges(old, new ChangeType) ChangeType {
	/*
		Merge rules:
		| Old    | New    | Result |
		|--------|--------|--------|
		| create | modify | create |
		| create | delete | remove |
		| modify | modify | modify |
		| modify | delete | delete |
	*/
	switch {
	case old == ChangeCreate && new == ChangeModify:
		return ChangeCreate
	case old == ChangeCreate && new == ChangeDelete:
		return -1 // remove
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
