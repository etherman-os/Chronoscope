# Chronoscope Ingestion Service

Go + Gin API for ingesting session replay capture data (video chunks and events).

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | *(required)* |
| `MINIO_ENDPOINT` | MinIO S3-compatible endpoint | *(required)* |
| `MINIO_ACCESS_KEY` | MinIO access key | *(required)* |
| `MINIO_SECRET_KEY` | MinIO secret key | *(required)* |
| `SERVER_ADDR` | HTTP server listen address | `:8080` |

## Running Locally

1. Copy `.env.example` to `.env` and adjust values.
2. Ensure PostgreSQL and MinIO are running.
3. Run the service:

```bash
go run cmd/server/main.go
```

## API Endpoints

All endpoints are prefixed with `/v1` and require the `X-API-Key` header.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/sessions/init` | Initialize a new session |
| POST | `/v1/sessions/:id/chunks` | Upload a video/image chunk |
| POST | `/v1/sessions/:id/events` | Upload a batch of events |
| POST | `/v1/sessions/:id/complete` | Finalize a session |
| GET | `/v1/sessions` | List sessions |
| GET | `/v1/sessions/:id` | Get session details with events |

## Docker

Build and run with Docker:

```bash
docker build -t chronoscope-ingestion .
docker run -p 8080:8080 --env-file .env chronoscope-ingestion
```
