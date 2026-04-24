# Production Deployment Guide

Deploy Chronoscope on your own infrastructure with Docker Compose, SSL/TLS, backups, and monitoring.

---

## Table of Contents

- [Docker Compose Production Setup](#docker-compose-production-setup)
- [Environment Variables](#environment-variables)
- [SSL/TLS with Nginx](#ssltls-with-nginx)
- [Backup Strategy](#backup-strategy)
- [Monitoring Basics](#monitoring-basics)

---

## Docker Compose Production Setup

Use the provided `docker-compose.yml` as a base. For production, create a `docker-compose.prod.yml` override:

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
      - "127.0.0.1:5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U chronoscope -d chronoscope"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redisdata:/data
    ports:
      - "127.0.0.1:6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_PASSWORD}
    volumes:
      - miniodata:/data
    ports:
      - "127.0.0.1:9000:9000"
      - "127.0.0.1:9001:9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5

  ingestion:
    build:
      context: .
      dockerfile: services/ingestion/Dockerfile
    environment:
      SERVER_ADDR: :8080
      DATABASE_URL: postgres://chronoscope:${DB_PASSWORD}@postgres:5432/chronoscope?sslmode=disable
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379
      S3_ENDPOINT: http://minio:9000
      S3_ACCESS_KEY: ${MINIO_USER}
      S3_SECRET_KEY: ${MINIO_PASSWORD}
      S3_BUCKET: chronoscope
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      minio:
        condition: service_healthy
    restart: unless-stopped

  analytics:
    build:
      context: .
      dockerfile: services/analytics/Dockerfile
    environment:
      SERVER_ADDR: :8081
      DATABASE_URL: postgres://chronoscope:${DB_PASSWORD}@postgres:5432/chronoscope?sslmode=disable
    ports:
      - "127.0.0.1:8081:8081"
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  processor:
    build:
      context: .
      dockerfile: services/processor/Dockerfile
    environment:
      DATABASE_URL: postgres://chronoscope:${DB_PASSWORD}@postgres:5432/chronoscope?sslmode=disable
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379
      S3_ENDPOINT: http://minio:9000
      S3_ACCESS_KEY: ${MINIO_USER}
      S3_SECRET_KEY: ${MINIO_PASSWORD}
      WORKER_COUNT: 4
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      minio:
        condition: service_healthy
    restart: unless-stopped

  web:
    build:
      context: ./services/web
      dockerfile: Dockerfile
    ports:
      - "127.0.0.1:3000:80"
    restart: unless-stopped

volumes:
  pgdata:
  redisdata:
  miniodata:
```

### Start Production Stack

```bash
export DB_PASSWORD=$(openssl rand -base64 32)
export REDIS_PASSWORD=$(openssl rand -base64 32)
export MINIO_USER=chronoscope
export MINIO_PASSWORD=$(openssl rand -base64 32)

docker compose -f docker-compose.prod.yml up -d
```

---

## Environment Variables

### Ingestion API

| Variable      | Default       | Description                       |
|---------------|---------------|-----------------------------------|
| `SERVER_ADDR` | `:8080`       | HTTP listen address               |
| `DATABASE_URL`| --            | PostgreSQL connection string      |
| `REDIS_URL`   | --            | Redis connection string           |
| `S3_ENDPOINT` | --            | MinIO / S3 endpoint               |
| `S3_ACCESS_KEY`| --           | S3 access key                     |
| `S3_SECRET_KEY`| --           | S3 secret key                     |
| `S3_BUCKET`   | `chronoscope` | Default bucket name               |
| `RATE_LIMIT_RPM` | `100`      | Requests per minute per API key   |

### Analytics API

| Variable      | Default       | Description                       |
|---------------|---------------|-----------------------------------|
| `SERVER_ADDR` | `:8081`       | HTTP listen address               |
| `DATABASE_URL`| --            | PostgreSQL connection string      |

### Processor

| Variable      | Default       | Description                       |
|---------------|---------------|-----------------------------------|
| `DATABASE_URL`| --            | PostgreSQL connection string      |
| `REDIS_URL`   | --            | Redis connection string           |
| `S3_ENDPOINT` | --            | MinIO / S3 endpoint               |
| `S3_ACCESS_KEY`| --           | S3 access key                     |
| `S3_SECRET_KEY`| --           | S3 secret key                     |
| `WORKER_COUNT`| `4`           | Concurrent processing workers     |

---

## SSL/TLS with Nginx

Terminate TLS at Nginx and proxy to the Chronoscope services.

### Nginx Configuration

```nginx
server {
    listen 443 ssl http2;
    server_name chronoscope.example.com;

    ssl_certificate /etc/letsencrypt/live/chronoscope.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/chronoscope.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256';
    ssl_prefer_server_ciphers on;

    # Ingestion API
    location /v1/ {
        proxy_pass http://127.0.0.1:8080/v1/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;
        proxy_request_buffering off;
        client_max_body_size 512M;
    }

    # Analytics API
    location /analytics/ {
        proxy_pass http://127.0.0.1:8081/v1/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Web Dashboard
    location / {
        proxy_pass http://127.0.0.1:3000/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name chronoscope.example.com;
    return 301 https://$server_name$request_uri;
}
```

### Obtain Certificates (Let's Encrypt)

```bash
sudo certbot --nginx -d chronoscope.example.com
```

---

## Backup Strategy

### PostgreSQL

Automated daily backups with `pg_dump`:

```bash
#!/bin/bash
# /usr/local/bin/backup-postgres.sh
BACKUP_DIR="/backups/postgres"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p "$BACKUP_DIR"

docker compose exec -T postgres pg_dump -U chronoscope chronoscope \
  | gzip > "$BACKUP_DIR/chronoscope_$DATE.sql.gz"

# Retain last 7 days
find "$BACKUP_DIR" -name "*.sql.gz" -mtime +7 -delete
```

Add to crontab:
```
0 2 * * * /usr/local/bin/backup-postgres.sh >> /var/log/backup-postgres.log 2>&1
```

### MinIO

Use `mc mirror` to sync buckets to offsite storage:

```bash
mc alias set local http://localhost:9000 $MINIO_USER $MINIO_PASSWORD
mc mirror local/chronoscope remote/chronoscope-backup
```

### Restore

```bash
# PostgreSQL
gunzip < /backups/postgres/chronoscope_20260115_020000.sql.gz \
  | docker compose exec -T postgres psql -U chronoscope -d chronoscope

# MinIO
mc mirror remote/chronoscope-backup local/chronoscope
```

---

## Monitoring Basics

### Health Checks

| Service    | Endpoint     | Expected Response          |
|------------|--------------|----------------------------|
| Ingestion  | `GET /healthz` | `200 OK` (DB + S3 healthy) |
| Analytics  | `GET /healthz` | `200 OK` (DB healthy)      |
| Processor  | stdout       | Heartbeat log every 30s    |

### Prometheus Metrics

Both Go services expose `/metrics` when `ENABLE_METRICS=true`:

| Metric                              | Type      | Description                  |
|-------------------------------------|-----------|------------------------------|
| `chronoscope_requests_total`        | Counter   | Total HTTP requests          |
| `chronoscope_request_duration_seconds` | Histogram | Request latency              |
| `chronoscope_active_sessions`       | Gauge     | Currently capturing sessions |

### Logging

All services log structured JSON to stdout. Recommended stack:
- **Collection**: Fluent Bit / Vector
- **Storage**: Loki / Elasticsearch
- **Dashboard**: Grafana

### Alerts (Example Prometheus Rules)

```yaml
groups:
  - name: chronoscope
    rules:
      - alert: HighErrorRate
        expr: rate(chronoscope_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate on {{ $labels.service }}"

      - alert: ProcessorDown
        expr: absent(chronoscope_processor_heartbeat)
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Video processor heartbeat missing"
```

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
For security best practices, see [SECURITY.md](SECURITY.md).
