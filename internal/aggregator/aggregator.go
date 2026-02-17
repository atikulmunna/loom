package aggregator

import (
	"context"
	"sync"
	"time"

	"github.com/atikulmunna/loom/internal/model"
)

// Stats holds a point-in-time snapshot of aggregated metrics.
type Stats struct {
	Uptime       string         `json:"uptime"`
	TotalEvents  int64          `json:"total_events"`
	EPS          float64        `json:"eps"`
	LevelCounts  map[string]int64 `json:"level_counts"`
	DroppedLogs  int64          `json:"dropped_logs"`
	FilesWatched int            `json:"files_watched"`
}

// Aggregator subscribes to the Hub and computes time-windowed metrics.
type Aggregator struct {
	mu          sync.RWMutex
	startTime   time.Time
	totalEvents int64
	levelCounts map[string]int64
	window      []time.Time // timestamps for EPS calculation (last 5 seconds)
	dropped     func() int64
	fileCount   func() int
	entries     <-chan model.LogEntry
}

// New creates an Aggregator that reads from the given Hub subscriber channel.
// droppedFn and fileCountFn provide live values from Hub and Watcher respectively.
func New(entries <-chan model.LogEntry, droppedFn func() int64, fileCountFn func() int) *Aggregator {
	return &Aggregator{
		startTime:   time.Now(),
		levelCounts: make(map[string]int64),
		dropped:     droppedFn,
		fileCount:   fileCountFn,
		entries:     entries,
	}
}

// Snapshot returns the current metrics.
func (a *Aggregator) Snapshot() Stats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Copy level counts.
	counts := make(map[string]int64)
	for k, v := range a.levelCounts {
		counts[k] = v
	}

	// Calculate EPS from the sliding window.
	now := time.Now()
	cutoff := now.Add(-5 * time.Second)
	var recent int
	for _, t := range a.window {
		if t.After(cutoff) {
			recent++
		}
	}
	eps := float64(recent) / 5.0

	return Stats{
		Uptime:       time.Since(a.startTime).Truncate(time.Second).String(),
		TotalEvents:  a.totalEvents,
		EPS:          eps,
		LevelCounts:  counts,
		DroppedLogs:  a.dropped(),
		FilesWatched: a.fileCount(),
	}
}

// Start begins consuming entries and updating metrics. Blocks until context is cancelled.
func (a *Aggregator) Start(ctx context.Context) {
	// Periodically prune the sliding window.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-a.entries:
			if !ok {
				return
			}
			a.record(entry)
		case <-ticker.C:
			a.prune()
		}
	}
}

// record adds an entry to the metrics.
func (a *Aggregator) record(entry model.LogEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.totalEvents++
	a.levelCounts[entry.Level]++
	a.window = append(a.window, time.Now())
}

// prune removes timestamps older than 5 seconds from the sliding window.
func (a *Aggregator) prune() {
	a.mu.Lock()
	defer a.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Second)
	i := 0
	for _, t := range a.window {
		if t.After(cutoff) {
			a.window[i] = t
			i++
		}
	}
	a.window = a.window[:i]
}
