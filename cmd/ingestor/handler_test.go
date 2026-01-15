package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/predatorx7/logtopus/pkg/broker"
	"github.com/predatorx7/logtopus/pkg/model"
)

// MockBroker for testing Handler
type MockBroker struct {
	PublishedLogs []model.LogEntry
	PublishErr    error
}

func (m *MockBroker) Publish(ctx context.Context, logs []model.LogEntry) error {
	if m.PublishErr != nil {
		return m.PublishErr
	}
	m.PublishedLogs = append(m.PublishedLogs, logs...)
	return nil
}

func (m *MockBroker) Subscribe(ctx context.Context) (<-chan []model.LogEntry, error) {
	return nil, nil
}

func (m *MockBroker) Stats() (uint64, uint64) {
	return uint64(len(m.PublishedLogs)), 0
}

// MockVerifier
func mockVerifierValid(key string) (bool, string, error) {
	return true, "test-client", nil
}

func mockVerifierInvalid(key string) (bool, string, error) {
	return false, "", nil
}

func mockVerifierError(key string) (bool, string, error) {
	return false, "", errors.New("verify error")
}

func TestHandler_HandleLogs(t *testing.T) {
	// Setup
	mockBroker := &MockBroker{}
	handler := NewHandler(mockBroker, mockVerifierValid)

	// Case 1: Success
	logs := []model.LogEntry{
		{Message: "msg1", Level: model.LogLevelInfo},
	}
	body, _ := json.Marshal(logs)
	req := httptest.NewRequest("POST", "/v1/logs", bytes.NewReader(body))
	req.Header.Set("X-API-Key", "valid-key")
	w := httptest.NewRecorder()

	handler.HandleLogs(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected 202, got %d", w.Code)
	}
	if len(mockBroker.PublishedLogs) != 1 {
		t.Errorf("Expected 1 log published, got %d", len(mockBroker.PublishedLogs))
	}
	if mockBroker.PublishedLogs[0].ClientID != "" {
		// Verify enrichment if any (we don't strictly set ClientID in handler yet, only remove APIKey)
	}

	// Case 2: Missing API Key
	req = httptest.NewRequest("POST", "/v1/logs", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.HandleLogs(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 on missing key, got %d", w.Code)
	}

	// Case 3: Invalid API Key
	handlerInvalid := NewHandler(mockBroker, mockVerifierInvalid)
	req = httptest.NewRequest("POST", "/v1/logs", bytes.NewReader(body))
	req.Header.Set("X-API-Key", "invalid-key")
	w = httptest.NewRecorder()
	handlerInvalid.HandleLogs(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 on invalid key, got %d", w.Code)
	}

	// Case 4: Invalid JSON
	req = httptest.NewRequest("POST", "/v1/logs", bytes.NewReader([]byte("{bad json")))
	req.Header.Set("X-API-Key", "valid-key")
	w = httptest.NewRecorder()
	handler.HandleLogs(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 on bad json, got %d", w.Code)
	}

	// Case 5: Broker Error
	mockBroker.PublishErr = errors.New("broker fail")
	req = httptest.NewRequest("POST", "/v1/logs", bytes.NewReader(body))
	req.Header.Set("X-API-Key", "valid-key")
	w = httptest.NewRecorder()
	handler.HandleLogs(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 on broker error, got %d", w.Code)
	}
}

// Ensure the MockBroker satisfies the interface
var _ broker.Broker = &MockBroker{}
