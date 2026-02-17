package tailer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atikulmunna/loom/internal/watcher"
)

func TestTailNewLines(t *testing.T) {
	// Create a temp log file with some pre-existing content.
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	if err := os.WriteFile(logPath, []byte("existing line\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Set up watcher, checkpoint, and tailer.
	w, err := watcher.New([]string{logPath})
	if err != nil {
		t.Fatal(err)
	}

	ckptPath := filepath.Join(dir, ".loom-state.json")
	ckpt, err := NewCheckpoint(ckptPath)
	if err != nil {
		t.Fatal(err)
	}

	tail := New(w, ckpt)

	ctx, cancel := context.WithCancel(context.Background())

	go w.Start(ctx)
	go tail.Start(ctx)

	// Give the tailer a moment to initialize and seek to end.
	time.Sleep(300 * time.Millisecond)

	// Append a new line — this should be picked up.
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("hello from test\n")
	f.Close()

	// Wait for the entry.
	select {
	case raw := <-tail.Lines():
		if raw.Text != "hello from test" {
			t.Errorf("expected 'hello from test', got %q", raw.Text)
		}
		if raw.Source != logPath {
			t.Errorf("expected source %q, got %q", logPath, raw.Source)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for log entry")
	}

	// Cancel and allow goroutines to stop before TempDir cleanup.
	cancel()
	time.Sleep(200 * time.Millisecond)
}

func TestCheckpointSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ckpt.json")

	// Create and save checkpoint.
	c1, err := NewCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	c1.Set("/var/log/app.log", 42)
	c1.Set("/var/log/err.log", 1024)
	if err := c1.Save(); err != nil {
		t.Fatal(err)
	}

	// Load checkpoint in a new instance.
	c2, err := NewCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}

	v1, ok := c2.Get("/var/log/app.log")
	if !ok || v1 != 42 {
		t.Errorf("expected 42, got %d (found=%v)", v1, ok)
	}

	v2, ok := c2.Get("/var/log/err.log")
	if !ok || v2 != 1024 {
		t.Errorf("expected 1024, got %d (found=%v)", v2, ok)
	}

	_, ok = c2.Get("/nonexistent")
	if ok {
		t.Error("expected missing key to return false")
	}
}
