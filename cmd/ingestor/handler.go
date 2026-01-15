package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/predatorx7/logtopus/pkg/broker"
	"github.com/predatorx7/logtopus/pkg/model"
)

type Handler struct {
	Broker   broker.Broker
	Verifier func(string) (bool, string, error)
}

func NewHandler(b broker.Broker, verifier func(string) (bool, string, error)) *Handler {
	return &Handler{
		Broker:   b,
		Verifier: verifier,
	}
}

func (h *Handler) HandleLogs(w http.ResponseWriter, r *http.Request) {
	// Authentication Placeholder
	// Authentication
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "Missing API Key", http.StatusUnauthorized)
		return
	}

	valid, _, err := h.Verifier(apiKey)
	if !valid || err != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	// Decode Batch
	var logs []model.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&logs); err != nil {
		http.Error(w, "Invalid Payload", http.StatusBadRequest)
		return
	}

	// Enrichment
	clientIP := r.RemoteAddr
	// If behind proxy, RealIP middleware handles it, but here we take what's available
	// or trust the middleware to have set it in RemoteAddr or standard headers if we used a helper.
	// Since we used middleware.RealIP, r.RemoteAddr is updated.

	for i := range logs {
		logs[i].ClientIP = clientIP

		// Default Level
		if logs[i].Level == "" {
			logs[i].Level = model.LogLevelInfo
		}

		// Default Time
		if logs[i].Time.IsZero() {
			logs[i].Time = time.Now()
		}
	}

	// Publish to Broker
	if err := h.Broker.Publish(r.Context(), logs); err != nil {
		http.Error(w, "Failed to ingest logs", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted"}`))
}
