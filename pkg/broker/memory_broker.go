package broker

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/predatorx7/logtopus/pkg/model"
)

// MemoryBroker implements a simple in-memory pub/sub using channels.
type MemoryBroker struct {
	subscribers                 []chan []model.LogEntry
	mu                          sync.RWMutex
	ingestedCount, droppedCount atomic.Uint64
}

// NewMemoryBroker creates a new instance of MemoryBroker.
func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		subscribers: make([]chan []model.LogEntry, 0),
	}
}

// Publish sends logs to all registered subscribers non-blocking.
func (b *MemoryBroker) Publish(ctx context.Context, logs []model.LogEntry) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	b.ingestedCount.Add(uint64(len(logs)))

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subscribers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sub <- logs:
			// stats could be tracked per subscriber, but globally we consider it "delivered"
		default:
			// Buffer full, drop message for this subscriber to prevent backpressure
			b.droppedCount.Add(uint64(len(logs)))
		}
	}
	return nil
}

// Subscribe returns a channel that receives log batches.
func (b *MemoryBroker) Subscribe(ctx context.Context) (<-chan []model.LogEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Increase buffer size to handle bursts
	ch := make(chan []model.LogEntry, 2000)
	b.subscribers = append(b.subscribers, ch)
	return ch, nil
}

// Stats returns the current metrics
func (b *MemoryBroker) Stats() (ingested, dropped uint64) {
	return b.ingestedCount.Load(), b.droppedCount.Load()
}
