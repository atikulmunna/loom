package tailer

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/atikulmunna/loom/internal/model"
	"github.com/atikulmunna/loom/internal/watcher"
)

// Tailer reads newly appended lines from watched files and emits RawLine values.
type Tailer struct {
	mu     sync.Mutex
	files  map[string]*trackedFile
	out    chan model.RawLine
	ckpt   *Checkpoint
	events <-chan watcher.Event
	watch  *watcher.Watcher
}

type trackedFile struct {
	path   string
	file   *os.File
	offset int64
	buf    string // partial line buffer
}

// New creates a Tailer that reads events from the given Watcher.
func New(w *watcher.Watcher, ckpt *Checkpoint) *Tailer {
	return &Tailer{
		files:  make(map[string]*trackedFile),
		out:    make(chan model.RawLine, 512),
		ckpt:   ckpt,
		events: w.Events,
		watch:  w,
	}
}

// Lines returns the channel where raw log lines are sent.
func (t *Tailer) Lines() <-chan model.RawLine {
	return t.out
}

// Start begins processing watcher events. Blocks until context is cancelled.
func (t *Tailer) Start(ctx context.Context) {
	defer close(t.out)

	// Open all initially watched files.
	for _, p := range t.watch.Paths() {
		t.openFile(p)
	}

	// Periodic checkpoint save.
	saveTicker := time.NewTicker(5 * time.Second)
	defer saveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.saveCheckpoint()
			t.closeAll()
			return

		case ev, ok := <-t.events:
			if !ok {
				return
			}
			t.handleEvent(ev)

		case <-saveTicker.C:
			t.saveCheckpoint()
		}
	}
}

// handleEvent dispatches watcher events to the appropriate handler.
func (t *Tailer) handleEvent(ev watcher.Event) {
	switch {
	case ev.Op&fsnotify.Write != 0:
		t.readNewLines(ev.Path)

	case ev.Op&fsnotify.Create != 0:
		// New file appeared (possibly after rotation).
		t.openFile(ev.Path)
		t.readNewLines(ev.Path)

	case ev.Op&fsnotify.Remove != 0, ev.Op&fsnotify.Rename != 0:
		// File rotated or deleted — close and schedule reconnect.
		t.closeFile(ev.Path)
		go t.reconnect(ev.Path)
	}
}

// openFile opens a file for tailing, resuming from the checkpointed offset.
func (t *Tailer) openFile(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.files[path]; exists {
		return
	}

	f, err := os.Open(path)
	if err != nil {
		log.Printf("cannot open %s: %v", path, err)
		return
	}

	// Resume from checkpoint or start at end of file.
	var offset int64
	if saved, ok := t.ckpt.Get(path); ok {
		offset = saved
	} else {
		offset, _ = f.Seek(0, io.SeekEnd)
	}
	f.Seek(offset, io.SeekStart)

	t.files[path] = &trackedFile{
		path:   path,
		file:   f,
		offset: offset,
	}
}

// readNewLines reads from the last offset to EOF and emits complete lines.
func (t *Tailer) readNewLines(path string) {
	t.mu.Lock()
	tf, ok := t.files[path]
	if !ok {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	scanner := bufio.NewScanner(tf.file)
	for scanner.Scan() {
		line := tf.buf + scanner.Text()
		tf.buf = ""

		t.out <- model.RawLine{Text: line, Source: path}
	}

	// If the last chunk didn't end with a newline, buffer it.
	if err := scanner.Err(); err != nil {
		log.Printf("read error on %s: %v", path, err)
	}

	// Update offset.
	pos, _ := tf.file.Seek(0, io.SeekCurrent)
	tf.offset = pos
	t.ckpt.Set(path, pos)
}

// closeFile releases a tracked file.
func (t *Tailer) closeFile(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if tf, ok := t.files[path]; ok {
		tf.file.Close()
		delete(t.files, path)
	}
}

// reconnect polls for a file to reappear after rotation (up to 5 retries).
func (t *Tailer) reconnect(path string) {
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		if _, err := os.Stat(path); err == nil {
			log.Printf("reconnected to rotated file: %s", path)
			_ = t.watch.ReWatch(path)
			t.openFile(path)
			return
		}
	}
	log.Printf("gave up reconnecting to %s after 5 retries", path)
}

// saveCheckpoint persists the current offsets to disk.
func (t *Tailer) saveCheckpoint() {
	if err := t.ckpt.Save(); err != nil {
		log.Printf("checkpoint save failed: %v", err)
	}
}

// closeAll closes all tracked file handles.
func (t *Tailer) closeAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for path, tf := range t.files {
		tf.file.Close()
		delete(t.files, path)
	}
}
