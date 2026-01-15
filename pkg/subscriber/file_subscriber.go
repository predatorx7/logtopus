package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/predatorx7/logtopus/pkg/broker"
	"github.com/predatorx7/logtopus/pkg/model"
)

type FileSubscriber struct {
	Broker    broker.Subscriber
	OutputDir string
	mu        sync.Mutex
}

func NewFileSubscriber(b broker.Subscriber, outDir string) *FileSubscriber {
	return &FileSubscriber{
		Broker:    b,
		OutputDir: outDir,
	}
}

func (s *FileSubscriber) Start(ctx context.Context) error {
	log.Println("Starting File Subscriber...")
	ch, err := s.Broker.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	if err := os.MkdirAll(s.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batch := <-ch:
			s.processBatch(batch)
		}
	}
}

func (s *FileSubscriber) processBatch(batch []model.LogEntry) {
	grouped := make(map[string][]model.LogEntry)
	for _, entry := range batch {
		sid := entry.SessionID
		if sid == "" {
			sid = "unknown_session"
		}
		grouped[sid] = append(grouped[sid], entry)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for sessionID, entries := range grouped {
		filename := filepath.Join(s.OutputDir, fmt.Sprintf("session_%s.log", sessionID))
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Error opening file %s: %v", filename, err)
			continue
		}

		for _, entry := range entries {
			data, _ := json.Marshal(entry)
			f.WriteString(string(data) + "\n")
		}
		f.Close()
	}
}
