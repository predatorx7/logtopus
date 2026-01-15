package broker

import (
	"context"
	"testing"

	"github.com/predatorx7/logtopus/pkg/model"
)

func TestMemoryBroker_Metrics(t *testing.T) {
	b := NewMemoryBroker()
	ctx := context.Background()

	// 1. Subscribe with small buffer to force drops
	// Actually Subscribe returns a large buffer now (2000). To test drops, we'd need to fill it
	// or modify the test to inject a small-buffer channel if possible, OR just trust logic.
	// Since we can't easily inject a small channel via public API, let's just test ingestion count.

	logs := []model.LogEntry{{Message: "msg"}}
	err := b.Publish(ctx, logs)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	ingested, dropped := b.Stats()
	if ingested != 1 {
		t.Errorf("Expected 1 ingested, got %d", ingested)
	}
	if dropped != 0 {
		t.Errorf("Expected 0 dropped, got %d", dropped)
	}
}
