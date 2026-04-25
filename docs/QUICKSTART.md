# Quick Start

Get Chronoscope running locally in under 5 minutes.

---

## Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for ingestion and analytics APIs)
- Node.js 20+ (for web dashboard)
- Rust 1.75+ (only if building the Linux SDK or running the video processor)
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
export $(grep -v '^#' .env | xargs)
go run cmd/server/main.go
```

The Ingestion API will be available at `http://localhost:8080`.

---

## 3. Start Analytics API

In a new terminal:

```bash
cd services/analytics
cp .env.example .env
export $(grep -v '^#' .env | xargs)
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

> **Note:** The web dashboard reads `VITE_API_KEY` from `.env` to authenticate with the Ingestion API. The default value matches the seeded demo key.

---

## 5. Create a Project and Generate an API Key

There are no seeded demo keys — you must create your own:

```bash
# Create an organization and project directly in the database
docker exec -it chronoscope-postgres psql -U chronoscope -d chronoscope -c "
INSERT INTO organizations (id, name) VALUES
  ('00000000-0000-0000-0000-000000000001', 'My Organization');

INSERT INTO projects (id, org_id, name, api_key_hash) VALUES
  ('00000000-0000-0000-0000-000000000002',
   '00000000-0000-0000-0000-000000000001',
   'My Project',
   -- SHA-256 hex of 'my-secret-api-key'
   '$(echo -n my-secret-api-key | sha256sum | cut -d\" \" -f1)');
"
```

Generate your API key hash:
```bash
echo -n "my-secret-api-key" | sha256sum | cut -d' ' -f1
```

## 6. Verify with cURL

```bash
# Replace SHA256_HASH with the hash from step 5
curl -X POST http://localhost:8080/v1/sessions/init \
  -H "X-API-Key: my-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123","capture_mode":"hybrid"}'
```

You should receive a JSON response with `session_id` and `upload_url`.

---

## Next Steps

- Integrate a [Capture SDK](SDK_INTEGRATION.md) into your desktop app
- Explore the [API Reference](API.md)
- Read the [Deployment Guide](DEPLOYMENT.md) for production setup
