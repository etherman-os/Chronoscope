# Production Deployment Guide

Deploy Chronoscope on your own infrastructure with Docker Compose, SSL/TLS, backups, and monitoring.

---

## Table of Contents

- [Docker Compose Production Setup](#docker-compose-production-setup)
- [Environment Variables](#environment-variables)
- [SSL/TLS with Nginx](#ssltls-with-nginx)
- [Backup Strategy](#backup-strategy)
- [Monitoring Basics](#monitoring-basics)
- [Zero-Downtime Updates](#zero-downtime-updates)

---

## Docker Compose Production Setup

Use the provided `docker/docker-compose.yml` as a base. For production, create a `docker-compose.prod.yml` override:

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
    image: minio/minio:RELEASE.2024-10-13T13-34-11Z
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
      context: ./services/ingestion
      dockerfile: Dockerfile
    environment:
      SERVER_ADDR: :8080
      DATABASE_URL: postgres://chronoscope:${DB_PASSWORD}@postgres:5432/chronoscope?sslmode=disable
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: ${MINIO_USER}
      MINIO_SECRET_KEY: ${MINIO_PASSWORD}
      MINIO_SECURE: "false"
      CORS_ALLOWED_ORIGIN: "https://app.yourdomain.com"
      DB_MAX_OPEN_CONNS: "25"
      DB_MAX_IDLE_CONNS: "5"
      DB_CONN_MAX_LIFETIME_MINUTES: "30"
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
      context: ./services/analytics
      dockerfile: Dockerfile
    environment:
      SERVER_ADDR: :8081
      DATABASE_URL: postgres://chronoscope:${DB_PASSWORD}@postgres:5432/chronoscope?sslmode=disable
      DB_MAX_OPEN_CONNS: "25"
      DB_MAX_IDLE_CONNS: "5"
      DB_CONN_MAX_LIFETIME_MINUTES: "30"
    ports:
      - "127.0.0.1:8081:8081"
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  processor:
    build:
      context: ./services/processor
      dockerfile: Dockerfile
    environment:
      DATABASE_URL: postgres://chronoscope:${DB_PASSWORD}@postgres:5432/chronoscope?sslmode=disable
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379
      AWS_ENDPOINT_URL: http://minio:9000
      AWS_ACCESS_KEY_ID: ${MINIO_USER}
      AWS_SECRET_ACCESS_KEY: ${MINIO_PASSWORD}
      S3_BUCKET: chronoscope-sessions
      S3_PROCESSED_BUCKET: chronoscope-processed
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
    environment:
      VITE_API_URL: "https://api.yourdomain.com/v1"
      VITE_PROJECT_ID: "your-project-id"
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

| Variable                      | Default       | Description                       |
|-------------------------------|---------------|-----------------------------------|
| `SERVER_ADDR`                 | `:8080`       | HTTP listen address               |
| `DATABASE_URL`                | —             | PostgreSQL connection string      |
| `MINIO_ENDPOINT`              | —             | MinIO host:port                   |
| `MINIO_ACCESS_KEY`            | —             | MinIO access key                  |
| `MINIO_SECRET_KEY`            | —             | MinIO secret key                  |
| `MINIO_SECURE`                | `false`       | Use TLS for MinIO                 |
| `CORS_ALLOWED_ORIGIN`         | `https://app.chronoscope.io` | Allowed CORS origin |
| `DB_MAX_OPEN_CONNS`           | `25`          | Max open DB connections           |
| `DB_MAX_IDLE_CONNS`           | `5`           | Max idle DB connections           |
| `DB_CONN_MAX_LIFETIME_MINUTES`| `30`          | Max DB connection lifetime        |

### Analytics API

| Variable                      | Default       | Description                       |
|-------------------------------|---------------|-----------------------------------|
| `SERVER_ADDR`                 | `:8081`       | HTTP listen address               |
| `DATABASE_URL`                | —             | PostgreSQL connection string      |
| `DB_MAX_OPEN_CONNS`           | `25`          | Max open DB connections           |
| `DB_MAX_IDLE_CONNS`           | `5`           | Max idle DB connections           |
| `DB_CONN_MAX_LIFETIME_MINUTES`| `30`          | Max DB connection lifetime        |

### Processor

| Variable              | Default                  | Description                   |
|-----------------------|--------------------------|-------------------------------|
| `DATABASE_URL`        | —                        | PostgreSQL connection string  |
| `REDIS_URL`           | `redis://localhost:6379` | Redis connection string       |
| `AWS_ENDPOINT_URL`    | `http://localhost:9000`  | S3-compatible endpoint        |
| `AWS_ACCESS_KEY_ID`   | —                        | S3 access key                 |
| `AWS_SECRET_ACCESS_KEY`| —                       | S3 secret key                 |
| `S3_BUCKET`           | —                        | Raw sessions bucket           |
| `S3_PROCESSED_BUCKET` | —                        | Processed videos bucket       |

### Web Dashboard

| Variable          | Default                      | Description              |
|-------------------|------------------------------|--------------------------|
| `VITE_API_URL`    | `http://localhost:8080/v1`   | Ingestion API base URL   |
| `VITE_PROJECT_ID` | —                            | Default project UUID     |

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
mc mirror local/chronoscope-sessions remote/chronoscope-sessions-backup
mc mirror local/chronoscope-processed remote/chronoscope-processed-backup
```

### Restore

```bash
# PostgreSQL
gunzip < /backups/postgres/chronoscope_20260115_020000.sql.gz \
  | docker compose exec -T postgres psql -U chronoscope -d chronoscope

# MinIO
mc mirror remote/chronoscope-sessions-backup local/chronoscope-sessions
mc mirror remote/chronoscope-processed-backup local/chronoscope-processed
```

---

## Monitoring Basics

### Prometheus Metrics

The Go services expose Prometheus-compatible metrics on `/metrics` when `ENABLE_METRICS=true`.

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

## Zero-Downtime Updates

1. **Blue/Green** or **Rolling** deployment for stateless APIs.
2. **Processor**: scale new workers, drain old queue, then terminate old workers.
3. **Database**: run migrations before deploying new code.

```bash
# Rolling restart
docker compose -f docker-compose.prod.yml up -d --no-deps --build ingestion
docker compose -f docker-compose.prod.yml up -d --no-deps --build analytics
docker compose -f docker-compose.prod.yml up -d --no-deps --build processor
```

---

For local development setup, see [CONTRIBUTING.md](CONTRIBUTING.md).
For security best practices, see [SECURITY.md](SECURITY.md).
