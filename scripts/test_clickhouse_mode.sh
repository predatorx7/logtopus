#!/bin/bash
set -e

echo "=== Testing ClickHouse Mode ==="

# 1. Start Services
echo "Starting Docker services..."
make docker-up MODE=clickhouse

# Wait for services
echo "Waiting for services to be ready..."
sleep 10

# 2. Setup DB
echo "Initializing Database..."
make setup-db

# 3. Generate API Key
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi
AUTH_SECRET=${AUTH_SECRET:-dev-secret}
echo "Using Secret: $AUTH_SECRET"

API_KEY=$(./scripts/create_api_key.sh -client test-ch -secret "$AUTH_SECRET" | grep -oE 'test-ch\.[a-zA-Z0-9_-]+')
echo "API Key: $API_KEY"

# 4. Ingest Log
echo "Ingesting log..."
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/v1/logs \
  -H "X-API-Key: $API_KEY" \
  -d '[{"message":"Automated CLICKHOUSE test", "level":"INFO"}]')

if [ "$RESPONSE" -ne 202 ]; then
    echo "Ingestion failed with status $RESPONSE"
    exit 1
fi
echo "Ingestion accepted."

# 5. Query Log
echo "Querying log..."
sleep 5 # async insert delay
QUERY_RESULT=$(curl -s "http://localhost:8081/v1/logs?subscriber_type=clickhouse&search=Automated")
echo "Query Result: $QUERY_RESULT"

if echo "$QUERY_RESULT" | grep -q "Automated CLICKHOUSE test"; then
    echo "SUCCESS: Log found in ClickHouse."
else
    echo "FAILURE: Log not found in ClickHouse."
    exit 1
fi

echo "=== ClickHouse Mode Test Passed ==="
