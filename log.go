package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/YoungY620/memo/mcp"
)

// Log levels: error=0, notice=1, info=2, debug=3
var logLevel = 2 // default: info

// Global history logger for watcher logs
var historyLog *mcp.HistoryLogger

func SetLogLevel(level string) {
	switch strings.ToLower(level) {
	case "error":
		logLevel = 0
	case "notice":
		logLevel = 1
	case "info":
		logLevel = 2
	case "debug":
		logLevel = 3
	default:
		logLevel = 2
	}
}

// InitHistoryLogger initializes the history logger for watcher
func InitHistoryLogger(memoDir string) {
	h, err := mcp.NewHistoryLogger(memoDir, "watcher")
	if err == nil {
		historyLog = h
	}
}

// CloseHistoryLogger closes the history logger
func CloseHistoryLogger() {
	if historyLog != nil {
		historyLog.Close()
		historyLog = nil
	}
}

func logError(format string, v ...any) {
	if logLevel >= 0 {
		log.Printf("[ERROR] "+format, v...)
	}
	if historyLog != nil {
		historyLog.LogError(fmt.Sprintf(format, v...), nil)
	}
}

func logNotice(format string, v ...any) {
	if logLevel >= 1 {
		log.Printf("[NOTICE] "+format, v...)
	}
	if historyLog != nil {
		historyLog.LogInfo(format, v...)
	}
}

func logInfo(format string, v ...any) {
	if logLevel >= 2 {
		log.Printf("[INFO] "+format, v...)
	}
	if historyLog != nil {
		historyLog.LogInfo(format, v...)
	}
}

func logDebug(format string, v ...any) {
	if logLevel >= 3 {
		log.Printf("[DEBUG] "+format, v...)
	}
	if historyLog != nil {
		historyLog.LogDebug(format, v...)
	}
}

// ============== Line Buffer ==============

// LineBuffer buffers text output and flushes on newlines or timeout
type LineBuffer struct {
	buffer    strings.Builder
	lastFlush time.Time
	timeout   time.Duration
}

// NewLineBuffer creates a new LineBuffer with the specified timeout
func NewLineBuffer(timeout time.Duration) *LineBuffer {
	return &LineBuffer{
		timeout:   timeout,
		lastFlush: time.Now(),
	}
}

// Write appends text to the buffer
func (lb *LineBuffer) Write(s string) {
	lb.buffer.WriteString(s)
}

// Flush returns content that should be output
// force=true: flush all buffered content
// force=false: only flush complete lines or on timeout
func (lb *LineBuffer) Flush(force bool) string {
	content := lb.buffer.String()
	if content == "" {
		return ""
	}

	// Force flush
	if force {
		lb.buffer.Reset()
		lb.lastFlush = time.Now()
		return strings.TrimRight(content, "\n")
	}

	// Check for complete lines
	if idx := strings.LastIndex(content, "\n"); idx != -1 {
		lines := content[:idx]
		lb.buffer.Reset()
		lb.buffer.WriteString(content[idx+1:])
		lb.lastFlush = time.Now()
		return lines
	}

	// Check timeout
	if time.Since(lb.lastFlush) >= lb.timeout {
		lb.buffer.Reset()
		lb.lastFlush = time.Now()
		return content
	}

	return ""
}
