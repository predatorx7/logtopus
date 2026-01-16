package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/predatorx7/logtopus/pkg/model"
	"github.com/predatorx7/logtopus/pkg/storage"
)

type ClickHouseStore struct {
	conn driver.Conn
	db   string
}

func NewClickHouseStore(dsn string) (*ClickHouseStore, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	dbName := opts.Auth.Database
	if dbName == "" {
		dbName = "logtopus"
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return &ClickHouseStore{
		conn: conn,
		db:   dbName,
	}, nil
}

func (s *ClickHouseStore) Query(ctx context.Context, params storage.QueryParams) ([]model.LogEntry, error) {
	query := fmt.Sprintf("SELECT timestamp, level, message, object, extra, logger_name, sequence, error, stacktrace, session_id, client_id, source, client_ip FROM %s.logs WHERE 1=1", s.db)
	args := []interface{}{}

	if !params.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, params.StartTime)
	}
	if !params.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, params.EndTime)
	}
	if params.Level != "" {
		query += " AND level = ?"
		args = append(args, params.Level)
	}
	if params.Search != "" {
		query += " AND message ILIKE ?" // ClickHouse ILIKE for case-insensitive
		args = append(args, "%"+params.Search+"%")
	}

	query += " ORDER BY timestamp DESC"

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var logs []model.LogEntry
	for rows.Next() {
		var entry model.LogEntry
		var objStr, extraStr string
		var levelStr string

		if err := rows.Scan(
			&entry.Time,
			&levelStr,
			&entry.Message,
			&objStr,
			&extraStr,
			&entry.LoggerName,
			&entry.Sequence,
			&entry.Error,
			&entry.Stacktrace,
			&entry.SessionID,
			&entry.ClientID,
			&entry.Source,
			&entry.ClientIP,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		entry.Level = model.LogLevel(levelStr)

		if objStr != "" {
			_ = json.Unmarshal([]byte(objStr), &entry.Object)
		}
		if extraStr != "" {
			_ = json.Unmarshal([]byte(extraStr), &entry.Extra)
		}

		logs = append(logs, entry)
	}

	return logs, nil
}
