package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/predatorx7/logtopus/pkg/broker"
	"github.com/predatorx7/logtopus/pkg/model"
)

type Subscriber struct {
	Broker broker.Subscriber
	DSN    string
	conn   driver.Conn
}

func NewSubscriber(b broker.Subscriber, dsn string) *Subscriber {
	return &Subscriber{
		Broker: b,
		DSN:    dsn,
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	log.Println("Starting ClickHouse Subscriber...")

	// Connection retry loop
	for {
		opts, err := clickhouse.ParseDSN(s.DSN)
		if err != nil {
			return fmt.Errorf("failed to parse DSN: %w", err)
		}

		conn, err := clickhouse.Open(opts)
		if err == nil {
			if err = conn.Ping(ctx); err == nil {
				s.conn = conn
				log.Println("ClickHouse Subscriber connected successfully")
				break
			}
		}

		log.Printf("ClickHouse Connection failed: %v. Retrying in 5s...", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			continue
		}
	}

	ch, err := s.Broker.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batch := <-ch:
			s.insertBatch(ctx, batch)
		}
	}
}

func (s *Subscriber) insertBatch(ctx context.Context, batch []model.LogEntry) {
	if len(batch) == 0 {
		return
	}

	start := time.Now()

	// Prepare batch
	batchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	batchConn, err := s.conn.PrepareBatch(batchCtx, "INSERT INTO logs")
	if err != nil {
		log.Printf("Failed to prepare batch: %v", err)
		return
	}

	for _, entry := range batch {
		objJSON, _ := json.Marshal(entry.Object)
		extraJSON, _ := json.Marshal(entry.Extra)

		err := batchConn.Append(
			entry.Time,
			string(entry.Level),
			entry.Message,
			string(objJSON),
			string(extraJSON),
			entry.LoggerName,
			entry.Sequence,
			entry.Error,
			entry.Stacktrace,
			entry.SessionID,
			entry.ClientID,
			entry.Source,
			entry.ClientIP,
		)
		if err != nil {
			log.Printf("Failed to append to batch: %v", err)
			return // abort batch
		}
	}

	if err := batchConn.Send(); err != nil {
		log.Printf("Failed to send batch to ClickHouse: %v", err)
	} else {
		log.Printf("[ClickHouse] Inserted %d rows in %v", len(batch), time.Since(start))
	}
}
