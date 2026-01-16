#!/bin/bash
set -e

echo "=== Testing File Mode ==="

# 1. Start Services
echo "Starting Docker services..."
make docker-up MODE=file

# Wait for services
echo "Waiting for services to be ready..."
sleep 5

# 2. Generate API Key
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi
AUTH_SECRET=${AUTH_SECRET:-dev-secret}
echo "Using Secret: $AUTH_SECRET"

API_KEY=$(./scripts/create_api_key.sh -client test-file -secret "$AUTH_SECRET" | grep -oE 'test-file\.[a-zA-Z0-9_-]+')
echo "API Key: $API_KEY"

# 3. Ingest Log
echo "Ingesting log..."
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/v1/logs \
  -H "X-API-Key: $API_KEY" \
  -d '[{"message":"Automated FILE test", "level":"INFO"}]')

if [ "$RESPONSE" -ne 202 ]; then
    echo "Ingestion failed with status $RESPONSE"
    exit 1
fi
echo "Ingestion accepted."

# 4. Query Log
echo "Querying log..."
sleep 2 # buffer flush
QUERY_RESULT=$(curl -s "http://localhost:8081/v1/logs?subscriber_type=file&search=Automated")
echo "Query Result: $QUERY_RESULT"

if echo "$QUERY_RESULT" | grep -q "Automated FILE test"; then
    echo "SUCCESS: Log found in File Store."
else
    echo "FAILURE: Log not found in File Store."
    exit 1
fi

echo "=== File Mode Test Passed ==="
