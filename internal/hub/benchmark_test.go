package hub

import (
	"context"
	"fmt"
	"testing"

	"github.com/atikulmunna/loom/internal/model"
	"github.com/atikulmunna/loom/internal/parser"
)

// BenchmarkHubBroadcast measures the cost of broadcasting to N subscribers.
func BenchmarkHubBroadcast1(b *testing.B)  { benchHubBroadcast(b, 1) }
func BenchmarkHubBroadcast5(b *testing.B)  { benchHubBroadcast(b, 5) }
func BenchmarkHubBroadcast10(b *testing.B) { benchHubBroadcast(b, 10) }

func benchHubBroadcast(b *testing.B, numSubs int) {
	input := make(chan model.RawLine, b.N+1)
	h := New(input, parser.NewAutoParser())

	// Create subscribers and drain them.
	for i := 0; i < numSubs; i++ {
		ch := h.Subscribe()
		go func() {
			for range ch {
			}
		}()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Start(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		input <- model.RawLine{
			Text:   fmt.Sprintf("2026-02-17 INFO benchmark event %d", i),
			Source: "bench.log",
		}
	}

	cancel()
}
