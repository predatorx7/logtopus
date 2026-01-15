package broker

import (
	"context"

	"github.com/predatorx7/logtopus/pkg/model"
)

// Publisher defines the interface for publishing log entries.
type Publisher interface {
	Publish(ctx context.Context, logs []model.LogEntry) error
}

// Subscriber defines the interface for consuming log entries.
type Subscriber interface {
	Subscribe(ctx context.Context) (<-chan []model.LogEntry, error)
}

// Broker combines Publisher and Subscriber interfaces.
type Broker interface {
	Publisher
	Subscriber
	Stats() (uint64, uint64)
}
