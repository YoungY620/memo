package buffer

import (
	"strings"
	"sync"
)

// ChangeKind represents the normalized type of a file-system change.
type ChangeKind int

const (
	// ChangeUnknown means no decision has been made yet.
	ChangeUnknown ChangeKind = iota
	// ChangeCreate indicates a new file appeared.
	ChangeCreate
	// ChangeModify indicates an existing file was modified.
	ChangeModify
	// ChangeDelete indicates a file was removed.
	ChangeDelete
)

func (k ChangeKind) String() string {
	switch k {
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

// SourceEvent captures information coming from upstream watchers or other producers.
// Providers may leave Kind as ChangeUnknown when they rely on the heuristic classifier.
type SourceEvent struct {
	Path string
	Op   string
	Kind ChangeKind
}

// Change represents the aggregated state for a single path.
type Change struct {
	Path string
	Kind ChangeKind
}

// Classifier decides how to interpret a SourceEvent when Kind is ChangeUnknown.
type Classifier interface {
	Classify(SourceEvent) ChangeKind
}

// defaultClassifier implements a simple suffix/operation heuristic.
type defaultClassifier struct {
	suffix map[string]ChangeKind
}

func newDefaultClassifier() *defaultClassifier {
	return &defaultClassifier{
		suffix: map[string]ChangeKind{
			".create": ChangeCreate,
			".add":    ChangeCreate,
			".new":    ChangeCreate,
			".update": ChangeModify,
			".change": ChangeModify,
			".modify": ChangeModify,
			".delete": ChangeDelete,
			".remove": ChangeDelete,
			".drop":   ChangeDelete,
		},
	}
}

func (c *defaultClassifier) Classify(ev SourceEvent) ChangeKind {
	if ev.Kind != ChangeUnknown {
		return ev.Kind
	}

	// Try suffix-based classification.
	lower := strings.ToLower(ev.Path)
	for suffix, kind := range c.suffix {
		if strings.HasSuffix(lower, suffix) {
			return kind
		}
	}

	// Fallback to operation hints.
	switch strings.ToLower(ev.Op) {
	case "create", "created", "add", "added":
		return ChangeCreate
	case "write", "update", "modify", "changed":
		return ChangeModify
	case "remove", "delete", "deleted", "rename":
		return ChangeDelete
	default:
		return ChangeModify
	}
}

// Buffer accumulates deduplicated changes and notifies subscribers when new data arrives.
type Buffer struct {
	mu         sync.RWMutex
	changes    map[string]ChangeKind
	classifier Classifier

	notifyOnce sync.Once
	notifyCh   chan struct{}
}

// Option allows customizing the buffer.
type Option func(*Buffer)

// WithClassifier injects a custom classifier.
func WithClassifier(classifier Classifier) Option {
	return func(b *Buffer) {
		if classifier != nil {
			b.classifier = classifier
		}
	}
}

// New returns a Buffer instance ready for concurrent use.
func New(opts ...Option) *Buffer {
	buf := &Buffer{
		changes:    make(map[string]ChangeKind),
		classifier: newDefaultClassifier(),
		notifyCh:   make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(buf)
	}
	return buf
}

// Ingest adds a new event to the buffer, merging with any existing state.
func (b *Buffer) Ingest(ev SourceEvent) {
	if ev.Path == "" {
		return
	}

	kind := b.classifier.Classify(ev)
	if kind == ChangeUnknown {
		kind = ChangeModify
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	prev, exists := b.changes[ev.Path]
	next := merge(prev, kind, exists)

	if next == ChangeUnknown {
		delete(b.changes, ev.Path)
	} else {
		b.changes[ev.Path] = next
	}

	select {
	case b.notifyCh <- struct{}{}:
	default:
	}
}

// Pending returns the number of tracked paths.
func (b *Buffer) Pending() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.changes)
}

// Flush returns the aggregated changes in FIFO order (based on map iteration) and clears the buffer.
func (b *Buffer) Flush() []Change {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.changes) == 0 {
		return nil
	}

	result := make([]Change, 0, len(b.changes))
	for path, kind := range b.changes {
		result = append(result, Change{Path: path, Kind: kind})
	}
	b.changes = make(map[string]ChangeKind)
	return result
}

// NotifyChan returns a channel that receives a signal whenever new data is available.
func (b *Buffer) NotifyChan() <-chan struct{} {
	b.notifyOnce.Do(func() {
		// make sure the channel is non-nil before returning.
		if b.notifyCh == nil {
			b.notifyCh = make(chan struct{}, 1)
		}
	})
	return b.notifyCh
}

// merge applies coalescing rules for sequential events.
func merge(old ChangeKind, new ChangeKind, existed bool) ChangeKind {
	if !existed {
		return new
	}

	switch {
	case old == ChangeCreate && new == ChangeDelete:
		return ChangeUnknown
	case old == ChangeCreate && new == ChangeModify:
		return ChangeCreate
	case old == ChangeDelete && new == ChangeCreate:
		return ChangeModify
	case new == ChangeDelete:
		return ChangeDelete
	default:
		return new
	}
}
