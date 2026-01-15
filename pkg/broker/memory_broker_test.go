package broker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/predatorx7/logtopus/pkg/model"
)

func TestMemoryBroker(t *testing.T) {
	b := NewMemoryBroker()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. Subscribe
	subCh, err := b.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// 2. Publish in background
	logs := []model.LogEntry{
		{Message: "test log 1", Level: model.LogLevelInfo},
		{Message: "test log 2", Level: model.LogLevelSevere},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := b.Publish(ctx, logs); err != nil {
			t.Errorf("Publish failed: %v", err)
		}
	}()

	// 3. Receive
	select {
	case received := <-subCh:
		if len(received) != 2 {
			t.Errorf("Expected 2 logs, got %d", len(received))
		}
		if received[0].Message != "test log 1" {
			t.Errorf("Unexpected log message: %s", received[0].Message)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for logs")
	}

	wg.Wait()
}

func TestMemoryBroker_ContextCancel(t *testing.T) {
	b := NewMemoryBroker()
	ctx, cancel := context.WithCancel(context.Background())

	// Fill the buffer of a subscriber
	subCh, _ := b.Subscribe(ctx)

	// Create a huge batch to fill buffer if needed, or simply test cancel
	// The buffer size is 100. Let's cancel context immediately.
	cancel()

	logs := []model.LogEntry{{Message: "msg"}}

	// Publish should fail or return ctx error
	err := b.Publish(ctx, logs)
	if err == nil {
		t.Error("Expected error on cancelled context, got nil")
	}

	// Drain just in case
	select {
	case <-subCh:
	default:
	}
}
