# Quick Start

Get Chronoscope running locally in under 5 minutes.

---

## Prerequisites

- Docker & Docker Compose
- Go 1.22+ (optional, for local dev)
- Node.js 18+ (optional, for frontend dev)

---

## 1. Clone & Start

```bash
git clone https://github.com/etherman-os/chronoscope.git
cd chronoscope
make up
```

This starts PostgreSQL, Redis, and MinIO in the background.

---

## 2. Verify Services

```bash
docker ps
# Should show: postgres, redis, minio, analytics
```

Wait until all containers report `healthy`:

```bash
docker compose -f docker/docker-compose.yml ps
```

---

## 3. Start Backend

```bash
cd services/ingestion
cp .env.example .env
go run cmd/server/main.go
```

The Ingestion API will be available at `http://localhost:8080`.

---

## 4. Start Dashboard

In a new terminal:

```bash
cd services/web
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

---

## 5. Test API

```bash
curl -X POST http://localhost:8080/v1/sessions/init \
  -H "X-API-Key: acad389951a6aa7659c8315a796f91e9" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123","capture_mode":"video"}'
```

You should receive a JSON response with a `session_id` and `upload_url`.

---

## Next Steps

- Integrate a [Capture SDK](docs/SDK_INTEGRATION.md) into your desktop app
- Explore the [API Reference](docs/API.md)
- Read the [Deployment Guide](docs/DEPLOYMENT.md) for production setup
