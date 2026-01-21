// Package trigger provides trigger management functionality
package trigger

import (
	"sync"
	"time"

	"github.com/user/kimi-sdk-agent-indexer/core/internal/buffer"
	"github.com/user/kimi-sdk-agent-indexer/core/internal/config"
)

// TriggerFunc 触发时调用的函数
type TriggerFunc func(changes []buffer.Change)

// Manager 触发管理器
type Manager struct {
	cfg       *config.TriggerConfig
	buf       *buffer.Buffer
	triggerFn TriggerFunc
	idleTimer *time.Timer
	mu        sync.Mutex
	done      chan struct{}
	running   bool
}

// New 创建新的触发管理器
func New(cfg *config.TriggerConfig, buf *buffer.Buffer, triggerFn TriggerFunc) *Manager {
	return &Manager{
		cfg:       cfg,
		buf:       buf,
		triggerFn: triggerFn,
		done:      make(chan struct{}),
	}
}

// Start 启动触发管理器
func (m *Manager) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	// 初始化空闲计时器
	idleTimeout := time.Duration(m.cfg.IdleMs) * time.Millisecond
	m.idleTimer = time.NewTimer(idleTimeout)

	go m.loop()
}

// Stop 停止触发管理器
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

// NotifyChange 通知有新的变更
func (m *Manager) NotifyChange() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	// 重置空闲计时器
	if m.idleTimer != nil {
		m.idleTimer.Stop()
		idleTimeout := time.Duration(m.cfg.IdleMs) * time.Millisecond
		m.idleTimer.Reset(idleTimeout)
	}

	// 检查是否达到文件数阈值
	if m.buf.Count() >= m.cfg.MinFiles {
		go m.trigger()
	}
}

// loop 主循环
func (m *Manager) loop() {
	for {
		select {
		case <-m.done:
			// 退出前触发一次（如果有变更）
			if !m.buf.IsEmpty() {
				m.trigger()
			}
			return
		case <-m.idleTimer.C:
			// 空闲超时，如果缓冲区非空则触发
			if !m.buf.IsEmpty() {
				m.trigger()
			}
			// 重置计时器
			idleTimeout := time.Duration(m.cfg.IdleMs) * time.Millisecond
			m.idleTimer.Reset(idleTimeout)
		}
	}
}

// trigger 执行触发
func (m *Manager) trigger() {
	changes := m.buf.Flush()
	if len(changes) == 0 {
		return
	}
	if m.triggerFn != nil {
		m.triggerFn(changes)
	}
}

// ForceTrigger 强制触发（用于手动触发或测试）
func (m *Manager) ForceTrigger() {
	m.trigger()
}
