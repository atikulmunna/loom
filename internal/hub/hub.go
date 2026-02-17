package hub

import (
	"context"
	"log"
	"sync"

	"github.com/atikulmunna/loom/internal/model"
	"github.com/atikulmunna/loom/internal/parser"
)

const subscriberBuffer = 1024

// Hub receives raw lines, parses them, and broadcasts LogEntry values to all subscribers.
type Hub struct {
	parser      parser.Parser
	input       <-chan model.RawLine
	mu          sync.RWMutex
	subscribers []chan model.LogEntry
	dropped     int64
}

// New creates a Hub that reads from the input channel and parses with the given parser.
func New(input <-chan model.RawLine, p parser.Parser) *Hub {
	return &Hub{
		parser: p,
		input:  input,
	}
}

// Subscribe returns a buffered channel that will receive parsed log entries.
// Multiple consumers can subscribe; each gets a copy of every entry.
func (h *Hub) Subscribe() <-chan model.LogEntry {
	ch := make(chan model.LogEntry, subscriberBuffer)
	h.mu.Lock()
	h.subscribers = append(h.subscribers, ch)
	h.mu.Unlock()
	return ch
}

// Dropped returns the total number of entries dropped due to slow consumers.
func (h *Hub) Dropped() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.dropped
}

// Start begins reading from the input channel, parsing, and broadcasting.
// Blocks until the context is cancelled or the input channel is closed.
func (h *Hub) Start(ctx context.Context) {
	defer h.closeAll()

	for {
		select {
		case <-ctx.Done():
			return
		case raw, ok := <-h.input:
			if !ok {
				return
			}
			entry := h.parser.Parse(raw.Text, raw.Source)
			h.broadcast(entry)
		}
	}
}

// broadcast sends an entry to all subscribers.
// If a subscriber's channel is full, the entry is dropped for that subscriber.
func (h *Hub) broadcast(entry model.LogEntry) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.subscribers {
		select {
		case ch <- entry:
		default:
			h.dropped++
			log.Printf("hub: dropped entry for slow consumer (total dropped: %d)", h.dropped)
		}
	}
}

// closeAll closes all subscriber channels.
func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.subscribers {
		close(ch)
	}
	h.subscribers = nil
}
