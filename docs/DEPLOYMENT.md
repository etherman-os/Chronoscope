# Self-Hosting Deployment Guide

Deploy Chronoscope on your own infrastructure.

---

## Table of Contents

- [Quick Start (Docker Compose)](#quick-start-docker-compose)
- [Production Deployment](#production-deployment)
- [Environment Variables](#environment-variables)
- [SSL / HTTPS](#ssl--https)
- [Backup & Restore](#backup--restore)
- [Monitoring](#monitoring)
- [Updating](#updating)

---

## Quick Start (Docker Compose)

The fastest way to run Chronoscope locally or on a single server.

### 1. Clone and Configure

```bash
git clone https://github.com/etherman-os/chronoscope.git
cd chronoscope

# Create environment files
cp services/ingestion/.env.example services/ingestion/.env
cp services/analytics/.env.example services/analytics/.env
```

Edit `.env` files with your database and MinIO credentials.

### 2. Start Infrastructure

```bash
make up
```

This starts:
- PostgreSQL 16 (`localhost:5432`)
- Redis 7 (`localhost:6379`)
- MinIO (`localhost:9000`)

### 3. Run Migrations

```bash
psql postgres://chronoscope:password@localhost:5432/chronoscope \
  -f migrations/001_initial_schema.sql

psql postgres://chronoscope:password@localhost:5432/chronoscope \
  -f migrations/002_audit_logs.sql
```

### 4. Start Services

**Terminal 1 — Ingestion API:**
```bash
cd services/ingestion
go run cmd/server/main.go
```

**Terminal 2 — Analytics API:**
```bash
cd services/analytics
go run cmd/server/main.go
```

**Terminal 3 — Processor:**
```bash
cd services/processor
cargo run --release
```

**Terminal 4 — Web Dashboard:**
```bash
cd services/web
npm install
npm run build
npx serve dist
```

---

## Production Deployment

### Recommended Topology

```
                 ┌─────────────┐
                 │   CDN       │
                 │ (Landing)   │
                 └─────────────┘
                        │
┌──────────────┐  ┌─────▼──────┐  ┌──────────────┐
│   Load       │  │  Ingestion │  │   Web Dash   │
│  Balancer    │──│   API      │──│   (static)   │
│   (Nginx)    │  │  (3x)      │  └──────────────┘
└──────────────┘  └────────────┘
                         │
           ┌─────────────┼─────────────┐
           ▼             ▼             ▼
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │ Postgres │  │  MinIO   │  │  Redis   │
    │ Primary  │  │ Cluster  │  │ Sentinel │
    └──────────┘  └──────────┘  └──────────┘
           │             │             │
    ┌──────▼──────┐  ┌───▼────┐  ┌────▼─────┐
    │  Postgres   │  │Processor│  │ Analytics│
    │  Replica    │  │ Workers │  │   API    │
    └─────────────┘  └─────────┘  └──────────┘
```

### Docker Compose (Single Node)

A production-ready `docker-compose.yml` example:

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: chronoscope
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: chronoscope
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    volumes:
      - redisdata:/data
    ports:
      - "6379:6379"

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_PASSWORD}
    volumes:
      - miniodata:/data
    ports:
      - "9000:9000"
      - "9001:9001"

  ingestion:
    build:
      context: .
      dockerfile: services/ingestion/Dockerfile
    environment:
      DATABASE_URL: postgres://chronoscope:${DB_PASSWORD}@postgres:5432/chronoscope
      REDIS_URL: redis://redis:6379
      S3_ENDPOINT: http://minio:9000
      S3_ACCESS_KEY: ${MINIO_USER}
      S3_SECRET_KEY: ${MINIO_PASSWORD}
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
      - minio

  analytics:
    build:
      context: .
      dockerfile: services/analytics/Dockerfile
    environment:
      DATABASE_URL: postgres://chronoscope:${DB_PASSWORD}@postgres:5432/chronoscope
    ports:
      - "8081:8081"
    depends_on:
      - postgres

  processor:
    build:
      context: .
      dockerfile: services/processor/Dockerfile
    environment:
      DATABASE_URL: postgres://chronoscope:${DB_PASSWORD}@postgres:5432/chronoscope
      REDIS_URL: redis://redis:6379
      S3_ENDPOINT: http://minio:9000
      S3_ACCESS_KEY: ${MINIO_USER}
      S3_SECRET_KEY: ${MINIO_PASSWORD}
    depends_on:
      - postgres
      - redis
      - minio

  web:
    build:
      context: ./services/web
      dockerfile: Dockerfile
    ports:
      - "80:80"

volumes:
  pgdata:
  redisdata:
  miniodata:
```

### Kubernetes (Helm)

For multi-node deployments, a Helm chart is available in `infrastructure/helm/`:

```bash
helm upgrade --install chronoscope ./infrastructure/helm/chronoscope \
  --namespace chronoscope \
  --create-namespace \
  --values values-production.yaml
```

---

## Environment Variables

### Ingestion API

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_ADDR` | `:8080` | HTTP listen address |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `REDIS_URL` | — | Redis connection string |
| `S3_ENDPOINT` | — | MinIO / S3 endpoint |
| `S3_ACCESS_KEY` | — | S3 access key |
| `S3_SECRET_KEY` | — | S3 secret key |
| `S3_BUCKET` | `chronoscope` | Default bucket name |

### Analytics API

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_ADDR` | `:8081` | HTTP listen address |
| `DATABASE_URL` | — | PostgreSQL connection string |

### Processor

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string |
| `REDIS_URL` | — | Redis connection string |
| `S3_ENDPOINT` | — | MinIO / S3 endpoint |
| `S3_ACCESS_KEY` | — | S3 access key |
| `S3_SECRET_KEY` | — | S3 secret key |
| `WORKER_COUNT` | `4` | Concurrent processing workers |

---

## SSL / HTTPS

For production, terminate TLS at your load balancer (Nginx, Traefik, AWS ALB, etc.).

### Nginx Example

```nginx
server {
    listen 443 ssl http2;
    server_name chronoscope.example.com;

    ssl_certificate /etc/letsencrypt/live/chronoscope.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/chronoscope.example.com/privkey.pem;

    location /v1/ {
        proxy_pass http://ingestion:8080/v1/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location / {
        proxy_pass http://web:80/;
    }
}
```

---

## Backup & Restore

### PostgreSQL

```bash
# Backup
pg_dump -h localhost -U chronoscope chronoscope > backup.sql

# Restore
psql -h localhost -U chronoscope chronoscope < backup.sql
```

### MinIO

Use `mc mirror` to sync buckets to offsite storage:

```bash
mc mirror local/chronoscope remote/chronoscope-backup
```

---

## Monitoring

### Health Checks

- **Ingestion**: `GET /healthz` (returns 200 if DB and S3 are reachable)
- **Analytics**: `GET /healthz`
- **Processor**: Logs heartbeat to stdout every 30s.

### Metrics (Prometheus)

Both Go services expose `/metrics` on their admin port when `ENABLE_METRICS=true`.

### Logging

All services log structured JSON to stdout. Recommended stack:
- **Collection**: Fluent Bit / Vector
- **Storage**: Loki / Elasticsearch
- **Dashboard**: Grafana

---

## Updating

### Zero-Downtime Update

1. **Blue/Green** or **Rolling** deployment for stateless APIs.
2. **Processor**: scale new workers, drain old queue, then terminate old workers.
3. **Database**: run migrations before deploying new code.

```bash
# Run migrations
docker compose exec postgres psql -U chronoscope -d chronoscope -f /migrations/003_new_feature.sql

# Rolling restart
docker compose up -d --no-deps --build ingestion
docker compose up -d --no-deps --build analytics
```

---

For local development setup, see [CONTRIBUTING.md](CONTRIBUTING.md).
