# Chronoscope API Reference

This document provides practical examples for interacting with the Chronoscope REST API.

---

## Base URLs

| Environment         | URL                              |
|---------------------|----------------------------------|
| Local (Ingestion)   | `http://localhost:8080/v1`       |
| Local (Analytics)   | `http://localhost:8081/v1`       |

## Authentication

All endpoints require an API key passed in the `X-API-Key` header. The server hashes the provided key with SHA-256 and compares it against `projects.api_key_hash`.

```bash
export CHRONOSCOPE_API_KEY="your-project-api-key"
```

---

## Rate Limiting

- **Ingestion API**: 100 requests/minute per API key
- **Analytics API**: 100 requests/minute per API key
- Exceeding the limit returns `429 Too Many Requests`.

---

## Error Codes

| Status | Description                                      |
|--------|--------------------------------------------------|
| 400    | Invalid request body or query parameters         |
| 401    | Missing or invalid `X-API-Key`                   |
| 403    | Session does not belong to project               |
| 404    | Session or resource not found                    |
| 429    | Rate limit exceeded                              |
| 500    | Unexpected server error                          |

**Example error response:**

```json
{
  "error": "session not found"
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
  "upload_url": "/v1/sessions/a1b2c3d4-e5f6-7890-abcd-ef1234567890/chunks",
  "expires_at": "2026-04-25T12:00:00Z"
}
```

**Request schema:**

| Field         | Type   | Required | Description                           |
|---------------|--------|----------|---------------------------------------|
| `user_id`     | string | yes      | User identifier                       |
| `capture_mode`| string | yes      | `video`, `events`, or `hybrid`        |
| `metadata`    | object | no       | Arbitrary key-value session metadata  |

---

### Upload a Video Chunk

Upload raw frame data during an active session. Maximum chunk size is 2 MiB. Maximum chunk index is 10000.

```bash
curl -X POST "http://localhost:8080/v1/sessions/${SESSION_ID}/chunks" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY" \
  -H "X-Chunk-Index: 0" \
  -F "chunk=@frame_chunk_0.jpg"
```

**Response:**

```json
{
  "received": true,
  "next_chunk": 1
}
```

**Headers:**

| Header          | Required | Description               |
|-----------------|----------|---------------------------|
| `X-Chunk-Index` | yes      | Zero-based chunk index    |

---

### Upload Event Batch

Send user interaction events (clicks, keystrokes, etc.). Maximum batch size is 1000 events.

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
        "payload": {"buttonText": "Save"}
      },
      {
        "event_type": "input",
        "timestamp_ms": 3500,
        "x": 200,
        "y": 150,
        "target": "input#username",
        "payload": {"value": "alice"}
      }
    ]
  }'
```

**Response:**

```json
{
  "count": 2
}
```

**Event schema:**

| Field         | Type   | Required | Description                           |
|---------------|--------|----------|---------------------------------------|
| `event_type`  | string | yes      | Category of event                     |
| `timestamp_ms`| int    | yes      | Milliseconds since session start      |
| `x`           | int    | no       | Screen X coordinate                   |
| `y`           | int    | no       | Screen Y coordinate                   |
| `target`      | string | no       | CSS-like selector or element ID       |
| `payload`     | object | no       | Arbitrary JSON extra data             |

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

Retrieve sessions for the authenticated project.

```bash
curl "http://localhost:8080/v1/sessions?limit=20&offset=0" \
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
      "created_at": "2026-04-25T10:30:00Z"
    }
  ]
}
```

**Query parameters:**

| Parameter | Type | Default | Description          |
|-----------|------|---------|----------------------|
| `limit`   | int  | 20      | Page size            |
| `offset`  | int  | 0       | Pagination offset    |

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
      "id": 1,
      "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "event_type": "click",
      "timestamp_ms": 1200,
      "x": 450,
      "y": 320,
      "target": "button#submit",
      "payload": null,
      "created_at": "2026-04-25T10:30:01Z"
    }
  ]
}
```

---

## Analytics API

### Heatmap Data

Get aggregated click coordinates for heatmap visualization.

```bash
curl "http://localhost:8081/v1/analytics/heatmap" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "project_id": "22222222-2222-2222-2222-222222222222",
  "points": [
    { "x": 450, "y": 320, "count": 15 },
    { "x": 200, "y": 150, "count": 8 }
  ]
}
```

---

### Funnel Data

Get conversion rates across funnel stages.

```bash
curl "http://localhost:8081/v1/analytics/funnel" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "project_id": "22222222-2222-2222-2222-222222222222",
  "funnel": [
    { "stage": "total_sessions", "count": 1000 },
    { "stage": "sessions_with_events", "count": 600 },
    { "stage": "sessions_with_chunks", "count": 250 },
    { "stage": "completed_sessions", "count": 180 }
  ]
}
```

---

### Session Statistics

Get summary stats for the authenticated project.

```bash
curl "http://localhost:8081/v1/analytics/sessions/stats" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "project_id": "22222222-2222-2222-2222-222222222222",
  "stats": {
    "avg_duration_ms": 342000,
    "total_sessions": 1240,
    "total_events": 52800,
    "avg_events_per_session": 42.58
  }
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
  "user_id": "user-123",
  "sessions": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "user_id": "user-123",
      "events": [...]
    }
  ],
  "total_events": 42
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
  "deleted_sessions": 5,
  "deleted_events": 120
}
```

---

### List Audit Logs

View compliance audit trail.

```bash
curl "http://localhost:8080/v1/gdpr/audit-logs?limit=50&offset=0" \
  -H "X-API-Key: $CHRONOSCOPE_API_KEY"
```

**Response:**

```json
{
  "logs": [
    {
      "id": 1,
      "project_id": "22222222-2222-2222-2222-222222222222",
      "action": "session_initiated",
      "actor": "user-123",
      "details": {"session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"},
      "created_at": "2026-04-25T11:00:00Z"
    }
  ],
  "total": 1
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
