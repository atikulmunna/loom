package hub

import (
	"context"
	"testing"
	"time"

	"github.com/atikulmunna/loom/internal/model"
	"github.com/atikulmunna/loom/internal/parser"
)

func TestHubBroadcast(t *testing.T) {
	input := make(chan model.RawLine, 10)
	h := New(input, parser.NewAutoParser())

	sub1 := h.Subscribe()
	sub2 := h.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.Start(ctx)

	// Send a line.
	input <- model.RawLine{Text: "ERROR disk full", Source: "test.log"}

	// Both subscribers should receive it.
	select {
	case e := <-sub1:
		if e.Level != "ERROR" {
			t.Errorf("sub1: expected ERROR, got %s", e.Level)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("sub1: timed out")
	}

	select {
	case e := <-sub2:
		if e.Level != "ERROR" {
			t.Errorf("sub2: expected ERROR, got %s", e.Level)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("sub2: timed out")
	}

	cancel()
}

func TestHubSlowConsumer(t *testing.T) {
	input := make(chan model.RawLine, 10)
	h := New(input, parser.NewAutoParser())

	// Subscribe but never read — simulates a slow consumer.
	_ = h.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.Start(ctx)

	// Fill beyond the subscriber buffer (1024).
	for i := 0; i < subscriberBuffer+100; i++ {
		input <- model.RawLine{Text: "line", Source: "test.log"}
	}

	// Give hub time to process.
	time.Sleep(500 * time.Millisecond)

	if h.Dropped() == 0 {
		t.Error("expected dropped entries for slow consumer, got 0")
	}

	cancel()
}
