package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/predatorx7/logtopus/pkg/broker"
)

type StatusResponse struct {
	Status       string `json:"status"`
	Uptime       string `json:"uptime"`
	IngestedLogs uint64 `json:"ingested_logs"`
	DroppedLogs  uint64 `json:"dropped_logs"`
}

var startTime = time.Now()

func HandleStatus(b *broker.MemoryBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ingested, dropped := b.Stats()

		resp := StatusResponse{
			Status:       "ok",
			Uptime:       time.Since(startTime).String(),
			IngestedLogs: ingested,
			DroppedLogs:  dropped,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
