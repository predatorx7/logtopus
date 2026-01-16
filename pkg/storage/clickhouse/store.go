package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
		query += " AND lower(level) = lower(?)"
		args = append(args, params.Level)
	}
	if params.Search != "" {
		query += " AND message ILIKE ?" // ClickHouse ILIKE for case-insensitive
		args = append(args, "%"+params.Search+"%")
	}
	if params.SessionID != "" {
		query += " AND lower(session_id) = lower(?)"
		args = append(args, params.SessionID)
	}
	if params.ClientID != "" {
		query += " AND lower(client_id) = lower(?)"
		args = append(args, params.ClientID)
	}
	if params.Source != "" {
		query += " AND source ILIKE ?"
		args = append(args, "%"+params.Source+"%")
	}
	if params.Error != "" {
		query += " AND error ILIKE ?"
		args = append(args, "%"+params.Error+"%")
	}

	query += " ORDER BY timestamp DESC"

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	} else {
		query += " LIMIT 100"
	}

	// 1. Initial Query
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var initialEntries []model.LogEntry
	for rows.Next() {
		entry, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		initialEntries = append(initialEntries, entry)
	}
	rows.Close()

	// 2. Fetch Context if needed
	if (params.Before > 0 || params.After > 0) && len(initialEntries) > 0 {
		finalResults := make([]model.LogEntry, 0, len(initialEntries)*(params.Before+params.After+1))

		for _, match := range initialEntries {
			// Construct base WHERE clause for context anchors
			whereClauses := []string{}
			args := []interface{}{}

			if params.SessionID != "" {
				whereClauses = append(whereClauses, "session_id = ?")
				args = append(args, match.SessionID)
			}
			if params.ClientID != "" {
				whereClauses = append(whereClauses, "client_id = ?")
				args = append(args, match.ClientID)
			}

			baseWhere := strings.Join(whereClauses, " AND ")
			if baseWhere == "" {
				// Should be caught by validation, but safe fallback
				continue
			}

			// Fetch Before
			if params.Before > 0 {
				beforeArgs := append(append([]interface{}{}, args...), match.Time, params.Before)
				beforeQuery := fmt.Sprintf(`SELECT timestamp, level, message, object, extra, logger_name, sequence, error, stacktrace, session_id, client_id, source, client_ip FROM %s.logs WHERE %s AND timestamp < ? ORDER BY timestamp DESC LIMIT ?`, s.db, baseWhere)

				beforeRows, err := s.conn.Query(ctx, beforeQuery, beforeArgs...)
				if err == nil {
					var beforeCtx []model.LogEntry
					for beforeRows.Next() {
						e, _ := scanRow(beforeRows)
						beforeCtx = append(beforeCtx, e)
					}
					beforeRows.Close()
					// Reverse to restore chronological order (older -> newer)
					for i := len(beforeCtx) - 1; i >= 0; i-- {
						finalResults = append(finalResults, beforeCtx[i])
					}
				}
			}

			// Add Match
			finalResults = append(finalResults, match)

			// Fetch After
			if params.After > 0 {
				afterArgs := append(append([]interface{}{}, args...), match.Time, params.After)
				afterQuery := fmt.Sprintf(`SELECT timestamp, level, message, object, extra, logger_name, sequence, error, stacktrace, session_id, client_id, source, client_ip FROM %s.logs WHERE %s AND timestamp > ? ORDER BY timestamp ASC LIMIT ?`, s.db, baseWhere)

				afterRows, err := s.conn.Query(ctx, afterQuery, afterArgs...)
				if err == nil {
					for afterRows.Next() {
						e, _ := scanRow(afterRows)
						finalResults = append(finalResults, e)
					}
					afterRows.Close()
				}
			}
		}
		return finalResults, nil
	}

	return initialEntries, nil
}

func scanRow(rows driver.Rows) (model.LogEntry, error) {
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
		return entry, fmt.Errorf("failed to scan row: %w", err)
	}

	entry.Level = model.LogLevel(levelStr)

	if objStr != "" {
		_ = json.Unmarshal([]byte(objStr), &entry.Object)
	}
	if extraStr != "" {
		_ = json.Unmarshal([]byte(extraStr), &entry.Extra)
	}
	return entry, nil
}
