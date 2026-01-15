#!/bin/bash

curl -X POST http://localhost:8080/v1/logs \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key-123" \
  -d '[
    {
      "level": "INFO",
      "message": "User logged in",
      "logger_name": "auth_service",
      "session_id": "sess_001",
      "client_id": "android_v1",
      "source": "mobile",
      "sequence": 1,
      "object": {"user_id": 101}
    },
    {
      "level": "ERROR",
      "message": "Failed to fetch profile",
      "logger_name": "profile_service",
      "session_id": "sess_001",
      "client_id": "android_v1",
      "source": "mobile",
      "sequence": 2,
      "error": "connection timeout"
    },
    {
      "level": "DEBUG",
      "message": "Cache miss",
      "logger_name": "cache_layer",
      "session_id": "sess_002",
      "client_id": "web_v2",
      "source": "frontend",
      "sequence": 1
    }
  ]'
