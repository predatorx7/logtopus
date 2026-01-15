#!/bin/bash

# Generated via: ./bin/apikey-gen -client=my-mobile-app -secret=dev-secret
API_KEY="my-mobile-app.BOkT86kkaDfz0BPcOJNohRccxYsDaQyqG7cQfuPAszo"

curl -X POST http://localhost:8080/v1/logs \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '[
    {
      "level": "INFO",
      "message": "Authenticated User Log",
      "logger_name": "auth_test",
      "session_id": "sess_auth_001",
      "client_id": "my-mobile-app",
      "source": "mobile",
      "sequence": 1
    }
  ]'
