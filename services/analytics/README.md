# Analytics Service

The Analytics Service provides aggregated analytics endpoints for Chronoscope projects.

## Endpoints

All endpoints are prefixed with `/v1` and require the `X-API-Key` header.

### GET /v1/analytics/heatmap
Returns event density by (x, y) coordinates.

### GET /v1/analytics/funnel
Returns the session completion funnel:
- total_sessions
- sessions_with_events
- sessions_with_chunks
- completed_sessions

### GET /v1/analytics/sessions/stats
Returns aggregate session statistics:
- avg_duration_ms
- total_sessions
- total_events
- avg_events_per_session

## Environment Variables

| Variable      | Description                    | Default |
|---------------|--------------------------------|---------|
| DATABASE_URL  | PostgreSQL connection string   | required |
| SERVER_ADDR   | Server listen address          | :8081   |

## Running Locally

```bash
cp .env.example .env
# adjust DATABASE_URL if needed
go run cmd/server/main.go
```

## Docker

```bash
docker build -t chronoscope-analytics .
docker run -p 8081:8081 -e DATABASE_URL=... chronoscope-analytics
```
