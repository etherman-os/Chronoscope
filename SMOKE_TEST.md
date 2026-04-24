# Smoke Test Guide

This document provides step-by-step commands to verify the security fixes locally.

## Prerequisites

- Go 1.22+
- Rust 1.75+ with Cargo
- Node.js 20+ with npm
- Docker & Docker Compose
- macOS with Xcode 15+ (for Swift SDK)
- PostgreSQL, Redis, and MinIO (or use Docker Compose)

---

## 1. Go Services

### Ingestion Service

```bash
cd services/ingestion

# Verify compilation
go build ./...

# Run unit tests
go test ./...

# Start server (requires DATABASE_URL, MinIO env vars)
export DATABASE_URL="postgres://chronoscope:chronoscope@localhost:5432/chronoscope?sslmode=disable"
export MINIO_ENDPOINT="localhost:9000"
export MINIO_ACCESS_KEY="chronoscope"
export MINIO_SECRET_KEY="chronoscope123"
export MINIO_BUCKET="chronoscope-sessions"
export CORS_ALLOWED_ORIGIN="http://localhost:5173"
go run cmd/server/main.go
```

**Expected Behavior:**
- Server starts on `:8080`
- `POST /v1/sessions/init` with `X-API-Key` returns 201
- `GET /v1/sessions?project_id=<other>` returns 401/403 (query param ignored)
- `POST /v1/sessions/:id/events` for another project's session returns 403
- Chunk uploads > 2 MiB return 413

### Analytics Service

```bash
cd services/analytics
export DATABASE_URL="postgres://chronoscope:chronoscope@localhost:5432/chronoscope?sslmode=disable"
export CORS_ALLOWED_ORIGIN="http://localhost:5173"
go run cmd/server/main.go
```

**Expected Behavior:**
- Server starts on `:8081`
- CORS header reflects `CORS_ALLOWED_ORIGIN` (not `*`)

---

## 2. Rust Services

### Privacy Engine

```bash
cd services/privacy-engine

# Verify compilation
cargo check

# Run tests
cargo test
```

**Expected Behavior:**
- Compiles without warnings
- Tests pass

### Processor

```bash
cd services/processor

# Install system dependencies (Ubuntu/Debian)
sudo apt-get update && sudo apt-get install -y \
  ffmpeg libavcodec-dev libavformat-dev libavutil-dev libswscale-dev pkg-config

# Verify compilation
cargo check

# Run tests
cargo test
```

**Expected Behavior:**
- Compiles after FFmpeg libraries are installed
- Graceful shutdown works: send `SIGTERM` or `Ctrl+C` and observe clean exit logs

---

## 3. Swift SDK (macOS only)

```bash
cd packages/sdk-macos

# Build
swift build

# Run tests
swift test
```

**Expected Behavior:**
- `CircularBuffer(capacity: 0)` triggers `preconditionFailure`
- `PrivacyEngine()` with invalid config returns `nil`
- `Chronoscope.shared.start(config:)` is actor-isolated and safe from races

**Note:** Cannot be built on Linux. Verify on macOS.

---

## 4. Web Dashboard

```bash
cd services/web

# Install dependencies
npm install

# Run linter (requires eslint config setup)
npm run lint

# Build for production
npm run build

# Start dev server
npm run dev
```

**Expected Behavior:**
- Build succeeds without TypeScript errors
- `VITE_API_KEY` must be set or the app throws at runtime
- `VITE_PROJECT_ID` must be set or the app throws at runtime
- DevTools → Network shows no hardcoded API key in JS bundle

### Verify CSP

1. Open dev server in browser.
2. Open DevTools → Elements → `<head>`.
3. Confirm `<meta http-equiv="Content-Security-Policy" ...>` is present.

---

## 5. Landing Page

```bash
cd services/landing

# Install dependencies
npm install

# Build
npm run build
```

**Expected Behavior:**
- Build succeeds (note: pre-existing client/server component issue may cause failures unrelated to security fixes)

---

## 6. Docker

```bash
# Build images
docker build -t chronoscope-ingestion services/ingestion
docker build -t chronoscope-analytics services/analytics
docker build -t chronoscope-processor services/processor

# Verify .dockerignore works
docker build -t chronoscope-ingestion services/ingestion --no-cache 2>&1 | grep -i ".env" || echo "No .env leaked"
```

**Expected Behavior:**
- Images build successfully
- No `.env` or `.git` directories are copied into layers

### Docker Compose

```bash
cd docker
docker compose up -d
```

**Expected Behavior:**
- PostgreSQL, Redis, and MinIO start
- MinIO image is pinned to a specific release tag (not `latest`)

---

## 7. End-to-End Security Verification

### API Key Hashing

```bash
# Create a project with SHA-256 hashed API key in DB
# Try accessing with raw key — should succeed
# Try accessing with wrong key — should return 401
```

### Cross-Project Access

```bash
# Using Project A's API key:
curl -H "X-API-Key: PROJECT_A_KEY" http://localhost:8080/v1/sessions?project_id=PROJECT_B_ID
# Expected: 200 but returns only Project A's sessions (query param ignored)

curl -H "X-API-Key: PROJECT_A_KEY" -X POST http://localhost:8080/v1/sessions/PROJECT_B_SESSION_ID/events -d '{"events":[]}'
# Expected: 403 Forbidden
```

### Rate Limiting

```bash
# Send >100 requests in 1 minute with the same API key
for i in {1..110}; do
  curl -s -o /dev/null -w "%{http_code}\n" -H "X-API-Key: SAME_KEY" http://localhost:8080/v1/sessions
done
# Expected: last ~10 requests return 429 Too Many Requests
```

### Chunk Upload Size

```bash
# Create a 3 MiB file
dd if=/dev/zero of=/tmp/big_chunk.jpg bs=1M count=3

# Upload
curl -H "X-API-Key: KEY" -H "X-Chunk-Index: 0" \
  -F "chunk=@/tmp/big_chunk.jpg" \
  http://localhost:8080/v1/sessions/SESSION_ID/chunks
# Expected: 413 Request Entity Too Large
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `cargo check` fails in processor | Install FFmpeg dev libraries (`libavcodec-dev`, etc.) |
| `swift build` fails on Linux | Swift SDK is macOS-only; test on macOS |
| `npm run build` fails in landing | Pre-existing Next.js architecture issue; not caused by security fixes |
| `go test` fails with DB connection | Start PostgreSQL/MinIO via Docker Compose first |
| ESLint config missing | Run `npm init @eslint/config` in `services/web` |
