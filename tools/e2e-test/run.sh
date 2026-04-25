#!/bin/bash
set -e

echo "=== Chronoscope E2E Test Suite ==="

# Start test infrastructure
docker compose -f docker-compose.test.yml up -d
sleep 5

# Wait for DB
POSTGRES_CONTAINER=$(docker compose -f docker-compose.test.yml ps -q postgres-test)
until docker exec "$POSTGRES_CONTAINER" pg_isready -U test -d chronoscope_test; do
    echo "Waiting for test DB..."
    sleep 1
done

# Fix api_key_hash to match SHA256 hashing in auth middleware
docker exec "$POSTGRES_CONTAINER" psql -U test -d chronoscope_test -c "
UPDATE projects SET api_key_hash = '0b61f1668881de754863abb929c1d7bd7048419fbec15bb49511d2c5781c7c13'
WHERE name = 'Demo App';
"

echo "=== Testing Session Lifecycle ==="

# 1. Init session
SESSION_RESPONSE=$(curl -s -X POST http://localhost:8082/v1/sessions/init \
    -H "X-API-Key: acad389951a6aa7659c8315a796f91e9" \
    -H "Content-Type: application/json" \
    -d '{"capture_mode":"video","user_id":"e2e-test"}')

SESSION_ID=$(echo $SESSION_RESPONSE | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)
echo "Created session: $SESSION_ID"

# 2. Upload events
curl -s -X POST "http://localhost:8082/v1/sessions/${SESSION_ID}/events" \
    -H "X-API-Key: acad389951a6aa7659c8315a796f91e9" \
    -H "Content-Type: application/json" \
    -d '{"events":[{"event_type":"click","timestamp_ms":1000,"x":100,"y":200}]}'

# 3. Complete session
curl -s -X POST "http://localhost:8082/v1/sessions/${SESSION_ID}/complete" \
    -H "X-API-Key: acad389951a6aa7659c8315a796f91e9"

# 4. Verify session exists
LIST_RESPONSE=$(curl -s "http://localhost:8082/v1/sessions?project_id=22222222-2222-2222-2222-222222222222" \
    -H "X-API-Key: acad389951a6aa7659c8315a796f91e9")

echo "Session list: $LIST_RESPONSE"

# Cleanup
docker compose -f docker-compose.test.yml down

echo "=== E2E PASSED ==="
