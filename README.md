<div align="center">

# 🔭 Chronoscope

**Session replay for desktop apps. Free. Open source. Self-hosted.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/etherman-os/chronoscope)](https://goreportcard.com/report/github.com/etherman-os/chronoscope)
[![CI Status](https://github.com/etherman-os/chronoscope/actions/workflows/ci.yml/badge.svg)](https://github.com/etherman-os/chronoscope/actions)
[![Version](https://img.shields.io/badge/version-0.2.0-blue.svg)](https://github.com/etherman-os/chronoscope/blob/main/VERSION)

[Quick Start](#quick-start) • [Features](#features) • [Architecture](#architecture) • [Docs](docs/)

</div>

---

## What is Chronoscope?

Chronoscope is a **session replay infrastructure** for native desktop applications. It records your users' screens, clicks, and interactions — then lets you replay them like a video in your browser.

Built for teams who:
- Ship **macOS, Windows, or Linux** desktop apps
- Need to debug "how did the user get here?" support tickets
- Want session replay insights without sending screen recordings to a third-party cloud
- Care about **data privacy** (GDPR, HIPAA, enterprise security)

> Think of it as a DVR for your desktop app. You press "record" via SDK, and your support team watches the playback later.

---

## Features

- **Screen & Event Capture** — Frame-by-frame video + click/scroll/keyboard event tracking via native SDKs
- **Cross-Platform SDKs** — Swift (macOS/ScreenCaptureKit), C++20 (Windows/WinRT Graphics Capture) *experimental*, Rust (Linux/PipeWire & X11) *experimental*
- **Privacy-First** — Text-level PII detection (credit cards, emails, passwords, SSN) via on-device Rust privacy engine; frame-level redaction is on the roadmap
- **Self-Hosted** — Runs entirely on your infrastructure. PostgreSQL + Redis + MinIO (S3-compatible object storage). No external SaaS dependency.
- **Real-Time Processing** — FFmpeg-powered video processor transcodes and deduplicates frames asynchronously
- **Replay Dashboard** — React-based player with timeline scrubbing and event overlay
- **Analytics API** — Pre-computed heatmaps, funnel stages, and session statistics
- **GDPR Ready** — User data export, right-to-be-forgotten deletion, and audit logging endpoints
- **Production Hardened** — bcrypt/SHA-256 API key hashing (migration path), rate limiting, CORS restrictions, input validation, and CSP headers

---

## Architecture

```mermaid
graph TD
    A[Desktop App] -->|SDK| B[Ingestion API]
    B --> C[(PostgreSQL)]
    B --> D[MinIO Storage]
    D --> E[Video Processor]
    E --> D
    E --> C
    F[Web Dashboard] --> B
    F --> G[Analytics API]
    G --> C
    H[Privacy Engine] --> E
```

**Flow:**
1. **Capture** — SDK records frames + events locally
2. **Ingest** — API receives chunks, stores metadata in PostgreSQL and video in MinIO
3. **Process** — Rust worker transcodes video, deduplicates frames, redacts PII
4. **Replay** — Web dashboard fetches processed video and events for playback
5. **Analyze** — Analytics API serves heatmaps, funnels, and session stats

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full deep dive.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| **Ingestion API** | Go 1.22 + Gin |
| **Analytics API** | Go 1.22 + Gin |
| **Video Processor** | Rust + FFmpeg + libav* |
| **Privacy Engine** | Rust (C ABI for cross-language FFI) |
| **Dashboard** | React 18 + Vite + TypeScript |
| **Landing Page** | Next.js 14 |
| **macOS SDK** | Swift + ScreenCaptureKit |
| **Windows SDK** | C++20 + WinRT Graphics Capture |
| **Linux SDK** | Rust + PipeWire / X11 |
| **Database** | PostgreSQL 16 |
| **Cache/Queue** | Redis 7 |
| **Object Storage** | MinIO (S3-compatible) |
| **Infra** | Docker Compose |

---

## Quick Start

Get a local instance running in **5 minutes**:

### Prerequisites

- Docker & Docker Compose
- Go 1.22+
- Node.js 20+
- Git
- Rust 1.75+ (only if building the Linux SDK)

### 1. Clone & Start Infrastructure

```bash
git clone https://github.com/etherman-os/chronoscope.git
cd chronoscope
make up
```

This starts PostgreSQL, Redis, and MinIO in the background.

### 2. Start the APIs

```bash
# Terminal 1 — Ingestion API
cd services/ingestion
cp .env.example .env
export $(grep -v '^#' .env | xargs)
go run cmd/server/main.go
# Server binds to SERVER_ADDR (default :8080 from .env)

# Terminal 2 — Analytics API
cd services/analytics
cp .env.example .env
export $(grep -v '^#' .env | xargs)
go run cmd/server/main.go
# Server binds to SERVER_ADDR (default :8081 from .env)
```

> **Note:** Use `SERVER_ADDR` env var to change the port (e.g., `SERVER_ADDR=:9000`). The legacy `PORT` env var is not supported.

### 3. Start the Dashboard

```bash
# Terminal 3 — Web UI
cd services/web
cp .env.example .env
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173) in your browser.

### 4. Create a Project & API Key

The easiest way is via `psql`:

```bash
docker exec -it chronoscope-postgres psql -U chronoscope -d chronoscope
```

```sql
INSERT INTO projects (id, name, api_key_hash)
VALUES (
  gen_random_uuid(),
  'My Project',
  -- Option A: bcrypt (recommended for new projects)
  '$2a$10$your-bcrypt-hash-here'
  -- Option B: SHA-256 hex (legacy, for migration compatibility)
  -- '$(echo -n "your-api-key" | sha256sum | cut -d' ' -f1)'
);
```

Use any bcrypt hasher to generate the hash from your desired API key.
SHA-256 hash can be generated with: `echo -n "your-api-key" | sha256sum | cut -d' ' -f1`

### 5. Verify with cURL

```bash
# 1. Initialize a session
curl -X POST http://localhost:8080/v1/sessions/init \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123","capture_mode":"hybrid"}'
# Returns: {"session_id":"...","upload_url":"/v1/sessions/.../chunks","expires_at":"..."}

# 2. Upload video chunks (multipart/form-data, JPEG required)
curl -X POST http://localhost:8080/v1/sessions/SESSION_ID/chunks \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "X-Chunk-Index: 0" \
  -F "chunk=@frame.jpg;type=image/jpeg"
# Returns: {"received":true,"next_chunk":1}

# 3. Upload click/scroll/keyboard events (JSON)
curl -X POST http://localhost:8080/v1/sessions/SESSION_ID/events \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"events":[{"event_type":"click","timestamp_ms":1000,"x":100,"y":200,"target":"button","payload":{"id":"btn-submit"}}]}'
# Returns: {"count":1}

# 4. Complete the session
curl -X POST http://localhost:8080/v1/sessions/SESSION_ID/complete \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"duration_ms":5000}'
# Returns: {"status":"completed"}
```

See [docs/QUICKSTART.md](docs/QUICKSTART.md) for the complete guide, including SDK embedding.

---

## SDK Quick Integration

Drop the SDK into your desktop app and start capturing in minutes:

**macOS (Swift):**
```swift
import Chronoscope

let endpoint = URL(string: "https://api.yourapp.com")!
let config = CaptureConfig(apiKey: "your-key", endpoint: endpoint)
await Chronoscope.shared.start(config: config)
// ... later ...
await Chronoscope.shared.stop()
```

**Windows (C++20):**
```cpp
#include <chronoscope/sdk.h>

chronoscope::CaptureConfig config{"your-key", "https://api.yourapp.com"};
auto session = chronoscope::Chronoscope::Instance()
    .StartSession(config, hwnd, nullptr);
// ... later ...
session->Stop();
```

**Linux (Rust):**
```rust
use chronoscope_sdk::{CaptureConfig, LinuxCapture};

let config = CaptureConfig::new("your-key", "https://api.yourapp.com");
let mut capture = LinuxCapture::new(config)?;
capture.start().await?;
// ... later ...
capture.stop().await?;
```

See [docs/SDK_INTEGRATION.md](docs/SDK_INTEGRATION.md) for full integration guides.

---

## Security

Security is not an afterthought. See [docs/SECURITY.md](docs/SECURITY.md) for the full policy.

Highlights:
- **API Key Hashing** — bcrypt (recommended, cost factor 10) and SHA-256 hex (legacy migration path)
- **Project Isolation** — Cross-project session access is impossible
- **Rate Limiting** — Redis-backed distributed rate limiting with in-memory fallback
- **Input Validation** — Chunk size (2 MiB), chunk index (10,000), event batch (1,000) limits
- **PII Redaction** — Automatic credit card, email, password, and SSN detection in frames
- **CSP & CORS** — Strict headers, configurable origin allowlist
- **Audit Logging** — Every GDPR export/delete is logged

---

## Documentation

- [Quick Start](docs/QUICKSTART.md) — 5-minute local setup
- [Architecture](docs/ARCHITECTURE.md) — Data flow, DB schema, deployment topology
- [API Reference](docs/API.md) — REST endpoints with cURL examples
- [SDK Integration](docs/SDK_INTEGRATION.md) — Embed capture SDKs into your app
- [Deployment](docs/DEPLOYMENT.md) — Production Docker Compose, SSL, backups, monitoring
- [Security](docs/SECURITY.md) — Security policy and hardening checklist
- [Contributing](docs/CONTRIBUTING.md) — Development setup and PR process

---

## Roadmap

- [ ] Windows SDK CI build on `windows-latest` runner
- [ ] Real-time WebSocket streaming for live session preview
- [ ] Session search by user action ("show me users who clicked X")
- [ ] SAML/SSO support for dashboard authentication
- [x] Prometheus metrics exporter for all services
- [ ] Electron SDK wrapper
- [ ] Mobile SDK (iOS/Android) experimental support

---

## Contributing

We welcome contributions! Please read [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) before opening a PR.

All commits use [Conventional Commits](https://www.conventionalcommits.org/) format.

---

## License

[MIT](LICENSE) © Chronoscope Contributors
