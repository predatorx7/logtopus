package storage

import (
	"context"
	"time"

	"github.com/predatorx7/logtopus/pkg/model"
)

// QueryParams defines criteria for filtering logs
type QueryParams struct {
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Level     string
	Search    string
}

// LogStore defines the interface for querying logs from a storage backend
type LogStore interface {
	Query(ctx context.Context, params QueryParams) ([]model.LogEntry, error)
}
