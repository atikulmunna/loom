package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/atikulmunna/loom/internal/model"
)

func TestEPSCalculation(t *testing.T) {
	ch := make(chan model.LogEntry, 100)
	agg := New(ch, func() int64 { return 0 }, func() int { return 2 })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agg.Start(ctx)

	// Send 10 entries quickly.
	for i := 0; i < 10; i++ {
		ch <- model.LogEntry{Level: "INFO", Message: "test"}
	}

	// Wait for processing.
	time.Sleep(200 * time.Millisecond)

	stats := agg.Snapshot()
	if stats.TotalEvents != 10 {
		t.Errorf("expected 10 total events, got %d", stats.TotalEvents)
	}
	if stats.EPS <= 0 {
		t.Errorf("expected positive EPS, got %f", stats.EPS)
	}

	cancel()
}

func TestLevelCounts(t *testing.T) {
	ch := make(chan model.LogEntry, 100)
	agg := New(ch, func() int64 { return 0 }, func() int { return 1 })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agg.Start(ctx)

	// Send entries with different levels.
	ch <- model.LogEntry{Level: "INFO", Message: "a"}
	ch <- model.LogEntry{Level: "INFO", Message: "b"}
	ch <- model.LogEntry{Level: "ERROR", Message: "c"}
	ch <- model.LogEntry{Level: "WARN", Message: "d"}
	ch <- model.LogEntry{Level: "ERROR", Message: "e"}

	time.Sleep(200 * time.Millisecond)

	stats := agg.Snapshot()
	if stats.LevelCounts["INFO"] != 2 {
		t.Errorf("expected 2 INFO, got %d", stats.LevelCounts["INFO"])
	}
	if stats.LevelCounts["ERROR"] != 2 {
		t.Errorf("expected 2 ERROR, got %d", stats.LevelCounts["ERROR"])
	}
	if stats.LevelCounts["WARN"] != 1 {
		t.Errorf("expected 1 WARN, got %d", stats.LevelCounts["WARN"])
	}
	if stats.FilesWatched != 1 {
		t.Errorf("expected 1 file watched, got %d", stats.FilesWatched)
	}

	cancel()
}
