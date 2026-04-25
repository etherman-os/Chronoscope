# Quick Start

Get Chronoscope running locally in under 5 minutes.

---

## Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for ingestion and analytics APIs)
- Node.js 20+ (for web dashboard)
- Git

---

## 1. Clone & Start Infrastructure

```bash
git clone https://github.com/etherman-os/chronoscope.git
cd chronoscope
make up
```

This starts PostgreSQL, Redis, and MinIO in the background.

Verify all containers are healthy:

```bash
docker compose -f docker/docker-compose.yml ps
```

---

## 2. Start Ingestion API

```bash
cd services/ingestion
cp .env.example .env
go run cmd/server/main.go
```

The Ingestion API will be available at `http://localhost:8080`.

---

## 3. Start Analytics API

In a new terminal:

```bash
cd services/analytics
cp .env.example .env
go run cmd/server/main.go
```

The Analytics API will be available at `http://localhost:8081`.

---

## 4. Start Web Dashboard

In a new terminal:

```bash
cd services/web
cp .env.example .env
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173) in your browser.

---

## 5. Verify with cURL

Initialize a session using the seeded demo API key:

```bash
curl -X POST http://localhost:8080/v1/sessions/init \
  -H "X-API-Key: acad389951a6aa7659c8315a796f91e9" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123","capture_mode":"hybrid"}'
```

You should receive a JSON response with `session_id` and `upload_url`.

---

## Next Steps

- Integrate a [Capture SDK](SDK_INTEGRATION.md) into your desktop app
- Explore the [API Reference](API.md)
- Read the [Deployment Guide](DEPLOYMENT.md) for production setup
