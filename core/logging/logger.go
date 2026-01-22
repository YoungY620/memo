package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level declares supported logging levels ordered by verbosity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelSilent
)

// Printer is the contract implemented by Logger.
type Printer interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	WithComponent(name string) Printer
}

type levelMeta struct {
	tag        string
	colorCode  string
	isTerminal bool
}

var metas = map[Level]levelMeta{
	LevelDebug: {tag: "DEBUG", colorCode: "36"}, // Cyan
	LevelInfo:  {tag: " INFO", colorCode: "32"}, // Green
	LevelWarn:  {tag: " WARN", colorCode: "33"}, // Yellow
	LevelError: {tag: "ERROR", colorCode: "31"}, // Red
}

// Option customises Logger.
type Option func(*Logger)

// WithLevel configures the minimum emitted level.
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// WithTimeFormat sets the timestamp format (empty disables timestamps).
func WithTimeFormat(layout string) Option {
	return func(l *Logger) {
		l.timeFormat = layout
	}
}

// WithColored toggles ANSI colouring.
func WithColored(colored bool) Option {
	return func(l *Logger) {
		l.colored = colored
	}
}

// WithWriter registers a dedicated writer for a level.
func WithWriter(level Level, w io.Writer) Option {
	return func(l *Logger) {
		if w != nil {
			l.writers[level] = w
		}
	}
}

type Logger struct {
	mu          sync.Mutex
	level       Level
	timeFormat  string
	colored     bool
	component   string
	writers     map[Level]io.Writer
	timeNowFunc func() time.Time
}

// New instantiates a structured logger.
func New(opts ...Option) *Logger {
	l := &Logger{
		level:      LevelInfo,
		timeFormat: "15:04:05.000",
		colored:    true,
		writers: map[Level]io.Writer{
			LevelDebug: os.Stdout,
			LevelInfo:  os.Stdout,
			LevelWarn:  os.Stdout,
			LevelError: os.Stderr,
		},
		timeNowFunc: time.Now,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// WithComponent clones the logger, appending component metadata.
func (l *Logger) WithComponent(name string) Printer {
	if l == nil {
		return NewNop()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	clone := l.cloneLocked()
	clone.component = name
	return clone
}

// SetTimeNow overrides the clock (primarily for tests).
func (l *Logger) SetTimeNow(fn func() time.Time) {
	if fn == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeNowFunc = fn
}

func (l *Logger) Debugf(format string, args ...any) {
	l.logf(LevelDebug, format, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	l.logf(LevelInfo, format, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.logf(LevelWarn, format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.logf(LevelError, format, args...)
}

func (l *Logger) logf(level Level, format string, args ...any) {
	if l == nil || level < l.level || l.level == LevelSilent {
		return
	}

	message := fmt.Sprintf(format, args...)
	message = strings.TrimRight(message, "\n")
	if message == "" {
		return
	}

	lines := splitLines(message)

	l.mu.Lock()
	defer l.mu.Unlock()

	ts := ""
	if l.timeFormat != "" {
		ts = l.timeNowFunc().Format(l.timeFormat)
	}

	meta := metas[level]
	writer := l.levelWriter(level)
	prefix := l.renderPrefix(meta, ts)
	connectors := renderConnectors(len(lines))

	for i, line := range lines {
		fmt.Fprintf(writer, "%s%s%s\n", prefix, connectors[i], line)
	}
}

func (l *Logger) levelWriter(level Level) io.Writer {
	if w, ok := l.writers[level]; ok && w != nil {
		return w
	}
	if level >= LevelError {
		if w, ok := l.writers[LevelError]; ok && w != nil {
			return w
		}
		return os.Stderr
	}
	if w, ok := l.writers[LevelInfo]; ok && w != nil {
		return w
	}
	return os.Stdout
}

func (l *Logger) renderPrefix(meta levelMeta, timestamp string) string {
	builder := strings.Builder{}

	tag := meta.tag
	if l.colored && meta.colorCode != "" {
		tag = fmt.Sprintf("\033[%sm%s\033[0m", meta.colorCode, meta.tag)
	}
	builder.WriteString(tag)
	builder.WriteByte(' ')

	if timestamp != "" {
		builder.WriteString(timestamp)
		builder.WriteByte(' ')
	}
	if l.component != "" {
		builder.WriteByte('[')
		builder.WriteString(l.component)
		builder.WriteString("] ")
	} else {
		builder.WriteString(" ")
	}
	return builder.String()
}

func splitLines(msg string) []string {
	msg = strings.ReplaceAll(msg, "\r\n", "\n")
	lines := strings.Split(msg, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, "\r")
	}
	return lines
}

func renderConnectors(total int) []string {
	if total <= 1 {
		return []string{"   "}
	}
	connectors := make([]string, total)
	for i := 0; i < total; i++ {
		switch {
		case i == 0:
			connectors[i] = "┬── "
		case i == total-1:
			connectors[i] = "└── "
		default:
			connectors[i] = "├── "
		}
	}
	return connectors
}

// NopLogger discards every message.
type NopLogger struct{}

func (NopLogger) Debugf(string, ...any) {}
func (NopLogger) Infof(string, ...any)  {}
func (NopLogger) Warnf(string, ...any)  {}
func (NopLogger) Errorf(string, ...any) {}
func (NopLogger) WithComponent(string) Printer {
	return NopLogger{}
}

// NewNop returns a logger that suppresses output.
func NewNop() Printer {
	return NopLogger{}
}

func (l *Logger) cloneLocked() *Logger {
	return &Logger{
		level:       l.level,
		timeFormat:  l.timeFormat,
		colored:     l.colored,
		component:   l.component,
		writers:     l.writers,
		timeNowFunc: l.timeNowFunc,
	}
}
