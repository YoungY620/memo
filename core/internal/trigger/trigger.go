// Package trigger provides trigger management functionality
package trigger

import (
	"sync"
	"time"

	"github.com/user/kimi-sdk-agent-indexer/core/internal/buffer"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/config"
)

// TriggerFunc function called when triggered
type TriggerFunc func(changes []buffer.Change)

// Manager trigger manager
type Manager struct {
	cfg       *config.TriggerConfig
	buf       *buffer.Buffer
	triggerFn TriggerFunc
	idleTimer *time.Timer
	mu        sync.Mutex
	done      chan struct{}
	running   bool
}

// New creates a new trigger manager
func New(cfg *config.TriggerConfig, buf *buffer.Buffer, triggerFn TriggerFunc) *Manager {
	return &Manager{
		cfg:       cfg,
		buf:       buf,
		triggerFn: triggerFn,
		done:      make(chan struct{}),
	}
}

// Start starts the trigger manager
func (m *Manager) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	// Initialize idle timer
	idleTimeout := time.Duration(m.cfg.IdleMs) * time.Millisecond
	m.idleTimer = time.NewTimer(idleTimeout)

	go m.loop()
}

// Stop stops the trigger manager
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	close(m.done)
	if m.idleTimer != nil {
		m.idleTimer.Stop()
	}
}

// NotifyChange notifies that there's a new change
func (m *Manager) NotifyChange() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	// Reset idle timer
	if m.idleTimer != nil {
		m.idleTimer.Stop()
		idleTimeout := time.Duration(m.cfg.IdleMs) * time.Millisecond
		m.idleTimer.Reset(idleTimeout)
	}

	// Check if file count threshold reached
	if m.buf.Count() >= m.cfg.MinFiles {
		go m.trigger()
	}
}

// loop main loop
func (m *Manager) loop() {
	for {
		select {
		case <-m.done:
			// Trigger once before exit (if there are changes)
			if !m.buf.IsEmpty() {
				m.trigger()
			}
			return
		case <-m.idleTimer.C:
			// Idle timeout, trigger if buffer not empty
			if !m.buf.IsEmpty() {
				m.trigger()
			}
			// Reset timer
			idleTimeout := time.Duration(m.cfg.IdleMs) * time.Millisecond
			m.idleTimer.Reset(idleTimeout)
		}
	}
}

// trigger executes the trigger
func (m *Manager) trigger() {
	changes := m.buf.Flush()
	if len(changes) == 0 {
		return
	}
	if m.triggerFn != nil {
		m.triggerFn(changes)
	}
}

// ForceTrigger forces a trigger (for manual trigger or testing)
func (m *Manager) ForceTrigger() {
	m.trigger()
}
