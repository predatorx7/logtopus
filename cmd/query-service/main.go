package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/predatorx7/logtopus/pkg/storage"
	"github.com/predatorx7/logtopus/pkg/storage/clickhouse"
	"github.com/predatorx7/logtopus/pkg/storage/file"
)

func main() {
	// 1. Initialize Stores
	var clickHouseStore storage.LogStore
	var fileStore storage.LogStore
	var err error

	// Initialize ClickHouse Store
	dsn := os.Getenv("CLICKHOUSE_DSN")
	if dsn != "" {
		// Retry connection loop
		for i := 0; i < 5; i++ {
			chStore, err := clickhouse.NewClickHouseStore(dsn)
			if err == nil {
				clickHouseStore = chStore
				break
			}
			log.Printf("Failed to connect to ClickHouse (attempt %d/5): %v", i+1, err)
			time.Sleep(1 * time.Second)
		}
		if clickHouseStore == nil {
			log.Println("Warning: ClickHouse store initialization failed, proceeding without it.")
		} else {
			log.Println("ClickHouse store initialized.")
		}
	}

	// Initialize File Store
	searchDir := os.Getenv("SEARCH_DIR")
	if searchDir == "" {
		searchDir = "./logs" // Default
	}
	fileStore, err = file.NewFileStore(searchDir)
	if err != nil {
		log.Printf("Warning: File store initialization failed: %v", err)
	} else {
		log.Printf("File store initialized (dir: %s)", searchDir)
	}

	// 2. Setup Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Serve OpenAPI Spec
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "openapi.query.yaml")
	})

	r.Get("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		// Parse Query Params
		params := storage.QueryParams{}
		subscriberType := r.URL.Query().Get("subscriber_type")

		var targetStore storage.LogStore
		switch subscriberType {
		case "file":
			targetStore = fileStore
		case "clickhouse", "":
			targetStore = clickHouseStore
		}

		if targetStore == nil {
			http.Error(w, fmt.Sprintf("Store '%s' is not available", subscriberType), http.StatusServiceUnavailable)
			return
		}

		if startStr := r.URL.Query().Get("start_time"); startStr != "" {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				params.StartTime = t
			}
		}
		if endStr := r.URL.Query().Get("end_time"); endStr != "" {
			if t, err := time.Parse(time.RFC3339, endStr); err == nil {
				params.EndTime = t
			}
		}
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil {
				params.Limit = l
			}
		}
		params.Level = r.URL.Query().Get("level")
		params.Search = r.URL.Query().Get("search")

		logs, err := targetStore.Query(r.Context(), params)
		if err != nil {
			http.Error(w, "Failed to query logs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logs)
	})

	// 3. Start Server
	port := os.Getenv("QUERY_PORT")
	if port == "" {
		port = "8081"
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("Starting Query Service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 4. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}
