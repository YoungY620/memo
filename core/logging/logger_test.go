package logging

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestDebugMultilineRendering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithLevel(LevelDebug),
		WithTimeFormat("15:04:05"),
		WithColored(false),
		WithWriter(LevelDebug, &buf),
	)
	logger.SetTimeNow(func() time.Time { return time.Date(2026, 1, 21, 10, 11, 12, 0, time.UTC) })

	logger.Debugf("first line\nsecond line\nthird line")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d\n%s", len(lines), output)
	}

	wantPrefixes := []string{
		"DEBUG 10:11:12  ┬── first line",
		"DEBUG 10:11:12  ├── second line",
		"DEBUG 10:11:12  └── third line",
	}
	for i, want := range wantPrefixes {
		if !strings.Contains(lines[i], want) {
			t.Fatalf("line %d mismatch\nwant contains: %q\ngot: %q", i, want, lines[i])
		}
	}
}

