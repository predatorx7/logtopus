package subscriber

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/predatorx7/logtopus/pkg/broker"
	"github.com/predatorx7/logtopus/pkg/model"
)

type ClickHouseSubscriber struct {
	Broker broker.Subscriber
	DSN    string
}

func NewClickHouseSubscriber(b broker.Subscriber, dsn string) *ClickHouseSubscriber {
	return &ClickHouseSubscriber{
		Broker: b,
		DSN:    dsn,
	}
}

func (s *ClickHouseSubscriber) Start(ctx context.Context) error {
	log.Println("Starting ClickHouse Subscriber...")
	ch, err := s.Broker.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batch := <-ch:
			s.insertBatch(batch)
		}
	}
}

func (s *ClickHouseSubscriber) insertBatch(batch []model.LogEntry) {
	log.Printf("[ClickHouse] Inserting %d rows into %s (Mock)...", len(batch), s.DSN)
	time.Sleep(10 * time.Millisecond)
}
