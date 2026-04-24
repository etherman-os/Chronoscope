# Chronoscope API Reference

This document provides practical examples for interacting with the Chronoscope REST API.

---

## Base URLs

| Environment         | URL                              |
|---------------------|----------------------------------|
| Local (Ingestion)   | `http://localhost:8080/v1`       |
| Local (Analytics)   | `http://localhost:8081/v1`       |

## Authentication

All endpoints require an API key passed in the `X-API-Key` header.

```bash
export CHRONOSCOPE_API_KEY="your-project-api-key"
```

---

## Rate Limiting

- **Ingestion API**: 100 requests/minute per API key (bursts up to 20)
- **Analytics API**: 300 requests/minute per API key
- Exceeding the limit returns `429 Too Many Requests` with a `Retry-After` header.

---

## Error Codes

| Status | Code                | Description                                      |
|--------|---------------------|--------------------------------------------------|
| 400    | `bad_request`       | Invalid request body or query parameters         |
| 401    | `unauthorized`      | Missing or invalid `X-API-Key`                   |
| 403    | `forbidden`         | API key does not have access to this resource    |
| 404    | `not_found`         | Session, project, or resource not found          |
| 409    | `conflict`          | Resource already exists or state conflict        |
| 429    | `rate_limited`      | Too many requests; see `Retry-After` header      |
| 500    | `internal_error`    | Unexpected server error                          |

**Example error response:**

```json
{
  "error": {
    "code": "bad_request",
    "message": "capture_mode must be one of: video, events, hybrid"
  }
}
```

---

## Ingestion API

### Initialize a Session

Start a new capture session for a user.

```bash
curl -X POST http://localhost:8080/v1/sessions/init \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY" \
  -d '{
    "user_id": "user-123",
    "capture_mode": "hybrid",
    "metadata": {
      "app_version": "1.2.3",
      "os_version": "macOS 14.0"
    }
  }'
```

**Response:**

```json
{
  "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "upload_url": "http://localhost:8080/v1/sessions/a1b2c3d4-e5f6-7890-abcd-ef1234567890/chunks",
  "token": "temp-upload-token",
  "expires_at": "2024-01-15T12:00:00Z"
}
```

---

### Upload a Video Chunk

Upload raw frame data during an active session.

```bash
curl -X POST "http://localhost:8080/v1/sessions/${SESSION_ID}/chunks" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY" \
  -H "X-Chunk-Index: 0" \
  -F "chunk=@frame_chunk_0.bin" \
  -F "timestamp_start=0" \
  -F "timestamp_end=5000"
```

**Response:**

```json
{
  "received": true,
  "next_chunk": 1
}
```

---

### Upload Event Batch

Send user interaction events (clicks, keystrokes, etc.).

```bash
curl -X POST "http://localhost:8080/v1/sessions/${SESSION_ID}/events" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY" \
  -d '{
    "events": [
      {
        "event_type": "click",
        "timestamp_ms": 1200,
        "x": 450,
        "y": 320,
        "target": "button#submit",
        "payload": "{\"buttonText\":\"Save\"}"
      },
      {
        "event_type": "input",
        "timestamp_ms": 3500,
        "x": 200,
        "y": 150,
        "target": "input#username",
        "payload": "{\"value\":\"alice\"}"
      }
    ]
  }'
```

**Response:**

```json
{
  "received": true
}
```

---

### Complete a Session

Signal that capture has finished.

```bash
curl -X POST "http://localhost:8080/v1/sessions/${SESSION_ID}/complete" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "status": "completed"
}
```

---

### List Sessions

Retrieve sessions for a project.

```bash
curl "http://localhost:8080/v1/sessions?project_id=${PROJECT_ID}&limit=10&offset=0" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "sessions": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "user_id": "user-123",
      "duration_ms": 45200,
      "event_count": 42,
      "status": "completed",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

---

### Get Session Details

Fetch a single session with its events.

```bash
curl "http://localhost:8080/v1/sessions/${SESSION_ID}" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "session": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "user_id": "user-123",
    "duration_ms": 45200,
    "video_path": "sessions/a1b2.../video.mp4",
    "event_count": 42,
    "status": "completed"
  },
  "events": [
    {
      "event_type": "click",
      "timestamp_ms": 1200,
      "x": 450,
      "y": 320,
      "target": "button#submit"
    }
  ]
}
```

---

## Analytics API

### Heatmap Data

Get aggregated click coordinates for heatmap visualization.

```bash
curl "http://localhost:8081/v1/analytics/heatmap?project_id=${PROJECT_ID}&start_date=2024-01-01&end_date=2024-01-31" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "heatmap": [
    { "x": 450, "y": 320, "count": 15 },
    { "x": 200, "y": 150, "count": 8 }
  ]
}
```

---

### Funnel Data

Get conversion rates across funnel stages.

```bash
curl "http://localhost:8081/v1/analytics/funnel?project_id=${PROJECT_ID}&funnel_id=signup" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "funnel": [
    { "stage": "landing", "count": 1000, "conversion": 1.0 },
    { "stage": "signup_start", "count": 600, "conversion": 0.6 },
    { "stage": "signup_complete", "count": 250, "conversion": 0.25 }
  ]
}
```

---

### Session Statistics

Get summary stats for a date range.

```bash
curl "http://localhost:8081/v1/analytics/sessions/stats?project_id=${PROJECT_ID}&days=7" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "total_sessions": 1240,
  "avg_duration_ms": 342000,
  "total_events": 52800,
  "error_count": 12
}
```

---

## GDPR Endpoints

### Export User Data

Export all data associated with a user ID.

```bash
curl -X POST "http://localhost:8080/v1/gdpr/export/${USER_ID}" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "download_url": "https://minio.example.com/exports/user-123.zip?token=...",
  "expires_at": "2024-01-16T10:00:00Z"
}
```

---

### Delete User Data

Right-to-be-forgotten request.

```bash
curl -X DELETE "http://localhost:8080/v1/gdpr/delete/${USER_ID}" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "deleted": true,
  "affected_sessions": 5
}
```

---

### List Audit Logs

View compliance audit trail.

```bash
curl "http://localhost:8080/v1/gdpr/audit-logs?project_id=${PROJECT_ID}&limit=50" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "logs": [
    {
      "action": "gdpr.export",
      "actor": "admin@example.com",
      "details": { "user_id": "user-123" },
      "created_at": "2024-01-15T11:00:00Z"
    }
  ]
}
```

---

## Endpoint Summary

| Method | Endpoint                                | Description             | Auth     |
|--------|-----------------------------------------|-------------------------|----------|
| POST   | `/v1/sessions/init`                     | Initialize session      | API Key  |
| POST   | `/v1/sessions/{id}/chunks`              | Upload video chunk      | API Key  |
| POST   | `/v1/sessions/{id}/events`              | Upload event batch      | API Key  |
| POST   | `/v1/sessions/{id}/complete`            | Finalize session        | API Key  |
| GET    | `/v1/sessions`                          | List sessions           | API Key  |
| GET    | `/v1/sessions/{id}`                     | Get session details     | API Key  |
| GET    | `/v1/analytics/heatmap`                 | Heatmap data            | API Key  |
| GET    | `/v1/analytics/funnel`                  | Funnel data             | API Key  |
| GET    | `/v1/analytics/sessions/stats`          | Session statistics      | API Key  |
| POST   | `/v1/gdpr/export/{user_id}`             | Export user data        | API Key  |
| DELETE | `/v1/gdpr/delete/{user_id}`             | Delete user data        | API Key  |
| GET    | `/v1/gdpr/audit-logs`                   | List audit logs         | API Key  |

---

## OpenAPI Specification

For the full schema, see [`protocols/api-contracts/ingestion.yaml`](../protocols/api-contracts/ingestion.yaml).

You can also view it in a Swagger UI by importing the YAML file into [Swagger Editor](https://editor.swagger.io/).
