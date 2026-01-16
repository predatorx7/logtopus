package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/predatorx7/logtopus/pkg/auth"
	"github.com/predatorx7/logtopus/pkg/broker"
	"github.com/predatorx7/logtopus/pkg/subscriber/clickhouse"
	"github.com/predatorx7/logtopus/pkg/subscriber/file"
)

func main() {
	// 1. Initialize Broker (In-Memory for now)
	logBroker := broker.NewMemoryBroker()

	// 1.5 Start Subscribers
	if os.Getenv("ENABLE_FILE_LOGGING") == "true" {
		outDir := os.Getenv("FILE_LOG_DIR")
		if outDir == "" {
			outDir = "./logs"
		}
		fileSub := file.NewSubscriber(logBroker, outDir)
		go func() {
			if err := fileSub.Start(context.Background()); err != nil {
				log.Printf("File subscriber exited with error: %v", err)
			}
		}()
		log.Printf("File logging enabled (dir: %s)", outDir)
	}

	if os.Getenv("ENABLE_CLICKHOUSE") == "true" {
		dsn := os.Getenv("CLICKHOUSE_DSN")
		if dsn == "" {
			dsn = "clickhouse://default:password@localhost:9000/logtopus?debug=true"
		}
		chSub := clickhouse.NewSubscriber(logBroker, dsn)
		go func() {
			if err := chSub.Start(context.Background()); err != nil {
				log.Printf("ClickHouse subscriber exited with error: %v", err)
			}
		}()
		log.Printf("ClickHouse logging enabled (dsn: %s)", dsn)
	}

	// 2. Setup Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP) // To get correct ClientIP

	// 2.5 Setup Auth Secret
	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		// Only for dev, ideally strict
		log.Println("WARNING: AUTH_SECRET not set, using default 'dev-secret'")
		authSecret = "dev-secret"
	}

	verifier := func(key string) (bool, string, error) {
		return auth.VerifyAPIKey(key, []byte(authSecret))
	}

	// 3. Register Handlers
	handler := NewHandler(logBroker, verifier)
	r.Post("/v1/logs", handler.HandleLogs)
	r.Get("/status", HandleStatus(logBroker))

	// Serve Static Files
	r.Get("/logtopus.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/logtopus.png")
	})

	// Serve Swagger UI
	r.Get("/openapi", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/openapi/openapi.html")
	})

	// Serve OpenAPI Spec
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/openapi/openapi.ingestor.yaml")
	})

	r.Get("/openapi.base.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/openapi/openapi.base.yaml")
	})

	// 4. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("Starting Ingestion Service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 5. Graceful Shutdown
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
