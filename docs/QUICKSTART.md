# Quick Start

Run Chronoscope locally as a self-hosted desktop session replay stack, record a real Linux X11 session, and replay it in the browser.

---

## Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for ingestion and analytics APIs)
- Node.js 20+ (for web dashboard)
- Rust 1.75+ and FFmpeg/libav development libraries
- Linux X11 session for local recording
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

## 2. Seed a Local Project

```bash
make seed-local
```

This creates:

- API key: `local-dev-key`
- Project ID: `22222222-2222-2222-2222-222222222222`
- Project name: `Local Desktop App`

---

## 3. Start Services

Open one terminal per process:

```bash
make run-ingestion
make run-processor
make run-analytics
make run-web
```

The dashboard will be available at `http://localhost:5173`.

---

## 4. Record Linux Desktop

```bash
make record-linux DURATION=30 FPS=5
```

The recorder initializes a session through the ingestion API, captures X11 frames as JPEG chunks, records click events, completes the session, and lets the processor encode the replay video.

If you are on a server, CI machine, Wayland-only desktop, or any environment without an X11 display, create a synthetic replay instead:

```bash
make demo-session
```

This uploads generated JPEG frames and demo events through the same public ingestion API, then queues the processor exactly like a real captured session.

---

## 5. Replay the Session

Open `http://localhost:5173` and login with:

- API key: `local-dev-key`
- Project ID: `22222222-2222-2222-2222-222222222222`

Select the recorded session. If it is still `completed`, wait for the processor to publish the MP4 and refresh the session.

---

## 6. Verify API Manually

```bash
curl -X POST http://localhost:8080/v1/sessions/init \
  -H "X-API-Key: local-dev-key" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123","capture_mode":"hybrid"}'
```

You should receive a JSON response with `session_id` and `upload_url`.

---

## Next Steps

- Integrate a [Capture SDK](SDK_INTEGRATION.md) into your desktop app
- Explore the [API Reference](API.md)
- Read the [Deployment Guide](DEPLOYMENT.md) for production setup
