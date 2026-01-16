package clickhouse

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// Config holds configuration for setting up ClickHouse
type Config struct {
	DSN             string
	LogsTableTTL    time.Duration
	LogsTableEngine string
}

// Setup initializes the ClickHouse database and logs table.
// It connects to the default database to create the target database if strictly necessary,
// or relies on the driver/server handling if possible, but the standard way is connecting to default/system first.
func Setup(ctx context.Context, dsn string) error {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %w", err)
	}

	targetDB := opts.Auth.Database
	if targetDB == "" {
		targetDB = "logtopus"
	}

	// Connect to 'default' database first to ensure we can create the target database
	// We clone options to avoid modifying the original if we were returning it,
	// but here we just need a connection for setup.
	setupOpts := *opts
	setupOpts.Auth.Database = "default"

	conn, err := clickhouse.Open(&setupOpts)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	log.Printf("Connected to ClickHouse at %v for setup\n", setupOpts.Addr)

	// Create Database
	log.Printf("Creating database '%s' if not exists...", targetDB)
	err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", targetDB))
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Create Table
	// Using DateTime64(3) for millisecond precision
	schema := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.logs (
			timestamp DateTime64(3),
			level LowCardinality(String),
			message String,
			object String,
			extra String,
			logger_name String,
			sequence UInt64,
			error String,
			stacktrace String,
			session_id String,
			client_id String,
			source String,
			client_ip String
		) ENGINE = MergeTree()
		ORDER BY timestamp
		TTL timestamp + INTERVAL 3 DAY
	`, targetDB)

	log.Printf("Creating table '%s.logs' if not exists...", targetDB)
	err = conn.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}
