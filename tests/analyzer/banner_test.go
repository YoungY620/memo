package analyzer_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/YoungY620/memo/analyzer"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintBanner(t *testing.T) {
	opts := analyzer.BannerOptions{
		WorkDir: "/test/path",
		Version: "1.0.0",
	}

	// Just make sure it doesn't panic
	output := captureOutput(func() {
		analyzer.PrintBanner(opts)
	})

	if output == "" {
		t.Error("Expected non-empty banner output")
	}

	// Should contain version
	if !bytes.Contains([]byte(output), []byte("1.0.0")) {
		t.Error("Banner should contain version")
	}
}

func TestPrintBanner_LongPath(t *testing.T) {
	opts := analyzer.BannerOptions{
		WorkDir: "/very/long/path/that/might/need/truncation/to/fit/in/the/banner/display/properly/test",
		Version: "dev",
	}

	// Should not panic with long path
	output := captureOutput(func() {
		analyzer.PrintBanner(opts)
	})

	if output == "" {
		t.Error("Expected non-empty banner output")
	}
}
