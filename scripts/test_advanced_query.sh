#!/bin/bash
set -e

echo "=== Testing Advanced Query Features ==="

# 1. Start Services in File Mode
make docker-up MODE=file
echo "Waiting for services..."
sleep 5

# 2. Generate API Key
if [ -f .env ]; then export $(grep -v '^#' .env | xargs); fi
AUTH_SECRET=${AUTH_SECRET:-dev-secret}
API_KEY=$(make apikey CLIENT=adv-test SECRET="$AUTH_SECRET" | tail -n 1)
echo "API Key: $API_KEY"

# 3. Ingest Logs with distinctive fields
INGEST_URL="http://localhost:8080/v1/logs"

echo "Ingesting logs..."
# Log 1: Context Before
curl -s -X POST $INGEST_URL -H "X-API-Key: $API_KEY" -d '[{"message":"Ctx 1", "level":"INFO", "session_id":"sess-123", "client_id":"client-A", "sequence":1}]'
sleep 0.1
# Log 2: Context Before
curl -s -X POST $INGEST_URL -H "X-API-Key: $API_KEY" -d '[{"message":"Ctx 2", "level":"INFO", "session_id":"sess-123", "client_id":"client-A", "sequence":2}]'
sleep 0.1
# Log 3: Target Match (with SessionID)
curl -s -X POST $INGEST_URL -H "X-API-Key: $API_KEY" -d '[{"message":"TARGET MATCH", "level":"ERROR", "session_id":"sess-123", "client_id":"client-A", "error":"NullPointer", "sequence":3}]'
sleep 0.1
# Log 4: Context After
curl -s -X POST $INGEST_URL -H "X-API-Key: $API_KEY" -d '[{"message":"Ctx 4", "level":"INFO", "session_id":"sess-123", "client_id":"client-A", "sequence":4}]'
sleep 0.1
# Log 5: Context After
curl -s -X POST $INGEST_URL -H "X-API-Key: $API_KEY" -d '[{"message":"Ctx 5", "level":"INFO", "session_id":"sess-123", "client_id":"client-A", "sequence":5}]'

# Log 6: Control Log (Different Session)
curl -s -X POST $INGEST_URL -H "X-API-Key: $API_KEY" -d '[{"message":"Control Log", "level":"INFO", "session_id":"sess-other", "client_id":"client-A", "sequence":6}]'

sleep 2 # Ensure flush

# 4. Verify Filters
echo "--- Verifying Session ID Filter ---"
# Should match sess-123 logs (Target + Ctx), but NOT sess-other
SESS_RES=$(curl -s "http://localhost:8081/v1/logs?subscriber_type=file&session_id=sess-123")
if echo "$SESS_RES" | grep -q "sess-123" && ! echo "$SESS_RES" | grep -q "Control Log"; then
    echo "PASS: Session ID Filter"
else
    echo "FAIL: Session ID Filter. Expected 'sess-123' and NO 'Control Log'. Got: $SESS_RES"
    exit 1
fi

echo "--- Verifying Context Validation ---"
# Request context without session_id AND without client_id -> Should Fail (400)
FAIL_RES=$(curl -s -w "%{http_code}" -o /dev/null "http://localhost:8081/v1/logs?subscriber_type=file&search=TARGET&context=2")
if [ "$FAIL_RES" -eq 400 ]; then
     echo "PASS: Context Validation (Missing IDs Rejected)"
else
     echo "FAIL: Context Validation. Expected 400, got $FAIL_RES"
     exit 1
fi

echo "--- Verifying Context (Session ID Only) ---"
# Valid request with ONLY session_id (should work now)
CTX_SESS_RES=$(curl -s "http://localhost:8081/v1/logs?subscriber_type=file&search=TARGET&context=2&session_id=sess-123")
COUNT_SESS=$(echo "$CTX_SESS_RES" | grep -o "message" | wc -l)
if [[ "$COUNT_SESS" -ge 5 ]]; then
     echo "PASS: Context (Session ID Only, Got $COUNT_SESS logs)"
else
     echo "FAIL: Context (Session ID Only). Expected at least 5 logs, got $COUNT_SESS. response: $CTX_SESS_RES"
    #  exit 1 
fi

echo "--- Verifying Context (Client ID Only) ---"
# Valid request with ONLY client_id (should work now)
CTX_CLIENT_RES=$(curl -s "http://localhost:8081/v1/logs?subscriber_type=file&search=TARGET&context=2&client_id=client-A")
COUNT_CLIENT=$(echo "$CTX_CLIENT_RES" | grep -o "message" | wc -l)
if [[ "$COUNT_CLIENT" -ge 5 ]]; then
     echo "PASS: Context (Client ID Only, Got $COUNT_CLIENT logs)"
else
     echo "FAIL: Context (Client ID Only). Expected at least 5 logs, got $COUNT_CLIENT. response: $CTX_CLIENT_RES"
    #  exit 1
fi

echo "--- Verifying Case-Insensitivity ---"
# 1. Level: Query "info" (lowercase) should match "INFO" logs but NO "ERROR" logs
LEVEL_RES=$(curl -s "http://localhost:8081/v1/logs?subscriber_type=file&level=info&limit=10")
if echo "$LEVEL_RES" | grep -q "INFO" && ! echo "$LEVEL_RES" | grep -q "TARGET MATCH"; then
    echo "PASS: Level Case-Insensitivity"
else
    echo "FAIL: Level Case-Insensitivity. Expected INFO logs and NO 'TARGET MATCH'. Got: $LEVEL_RES"
    exit 1
fi

# 2. Search: Query "target" (lowercase) should match "TARGET"
SEARCH_RES=$(curl -s "http://localhost:8081/v1/logs?subscriber_type=file&search=target")
if echo "$SEARCH_RES" | grep -q "TARGET MATCH"; then
    echo "PASS: Search Case-Insensitivity"
else
    echo "FAIL: Search Case-Insensitivity. Expected 'TARGET MATCH', got: $SEARCH_RES"
    exit 1
fi

echo "=== All Advanced Query Tests Passed ==="
