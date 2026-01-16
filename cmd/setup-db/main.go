package main

import (
	"context"
	"log"
	"os"

	"github.com/predatorx7/logtopus/pkg/subscriber/clickhouse"
)

func main() {
	dsn := os.Getenv("CLICKHOUSE_DSN")
	if dsn == "" {
		dsn = "clickhouse://default:@localhost:9000/logtopus?debug=true"
	}

	log.Println("Starting ClickHouse setup...")
	ctx := context.Background()
	if err := clickhouse.Setup(ctx, dsn); err != nil {
		log.Fatalf("ClickHouse setup failed: %v", err)
	}

	log.Println("Database setup completed successfully.")
}
