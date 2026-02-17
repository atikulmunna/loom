package watcher

import (
	"context"
	"log"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fsnotify/fsnotify"
)

// Event represents a file change detected by the watcher.
type Event struct {
	Path string
	Op   fsnotify.Op
}

// Watcher monitors files and directories for changes using OS-level notifications.
type Watcher struct {
	fsw    *fsnotify.Watcher
	Events chan Event
	paths  []string
}

// New creates a Watcher for the given glob patterns.
// Patterns are expanded at startup and the resulting files are watched.
func New(patterns []string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsw:    fsw,
		Events: make(chan Event, 256),
	}

	for _, pattern := range patterns {
		matches, err := expandGlob(pattern)
		if err != nil {
			log.Printf("warning: failed to expand pattern %q: %v", pattern, err)
			continue
		}
		for _, m := range matches {
			abs, _ := filepath.Abs(m)
			if err := fsw.Add(abs); err != nil {
				log.Printf("warning: cannot watch %s: %v", abs, err)
				continue
			}
			w.paths = append(w.paths, abs)
		}
	}

	return w, nil
}

// Start begins listening for file events. It blocks until the context is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	defer w.fsw.Close()
	defer close(w.Events)

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			// Forward relevant events (write, create, remove, rename).
			switch {
			case ev.Op&fsnotify.Write != 0,
				ev.Op&fsnotify.Create != 0,
				ev.Op&fsnotify.Remove != 0,
				ev.Op&fsnotify.Rename != 0:
				w.Events <- Event{Path: ev.Name, Op: ev.Op}
			}
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

// Paths returns the list of files currently being watched.
func (w *Watcher) Paths() []string {
	return w.paths
}

// ReWatch adds a path back to the watcher (used after rotation).
func (w *Watcher) ReWatch(path string) error {
	return w.fsw.Add(path)
}

// expandGlob resolves a glob pattern to matching file paths.
// Supports recursive patterns like /var/log/**/*.log via doublestar.
func expandGlob(pattern string) ([]string, error) {
	return doublestar.FilepathGlob(pattern, doublestar.WithFilesOnly(), doublestar.WithFailOnIOErrors())
}
