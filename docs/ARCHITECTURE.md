# Chronoscope Architecture

This document describes the high-level design, data flow, and component interactions of the Chronoscope platform.

---

## System Overview

Chronoscope is a multi-service platform composed of:

1. **Capture SDKs** — embedded in desktop applications to record screen frames and user events.
2. **Ingestion API** — receives raw capture data, validates API keys, and stores metadata.
3. **Video Processor** — asynchronously processes video chunks (transcode, deduplicate, index).
4. **Analytics API** — serves aggregated metrics (heatmaps, funnels, session stats).
5. **Web Dashboard** — React-based UI for replaying sessions and exploring analytics.
6. **Privacy Engine** — Rust library for PII detection and frame redaction.
7. **Landing Page** — static Next.js site for marketing.

---

## Data Flow

```
┌──────────────┐     ┌──────────────┐     ┌─────────────────┐
│  Desktop App │────▶│  Ingestion   │────▶│   PostgreSQL    │
│  (SDK)       │     │  API (Go)    │     │   (metadata)    │
└──────────────┘     └──────────────┘     └─────────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │    MinIO     │
                     │  (S3 store)  │
                     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  Processor   │
                     │  (Rust)      │
                     └──────────────┘
                            │
           ┌────────────────┼────────────────┐
           ▼                ▼                ▼
    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
    │  Web Dash   │  │ Analytics   │  │  Privacy    │
    │  (React)    │  │  API (Go)   │  │  Engine     │
    └─────────────┘  └─────────────┘  └─────────────┘
```

### Capture Flow

1. The **SDK** (Swift / C++ / Rust) captures frames and events locally.
2. It buffers data and uploads it to the **Ingestion API** via HTTP/REST or Protobuf.
3. The API stores session metadata in **PostgreSQL** and raw video chunks in **MinIO**.
4. A message is pushed to **Redis** to notify the **Processor**.

### Processing Flow

1. The **Processor** (Rust) polls the Redis queue for new jobs.
2. It downloads chunks from MinIO, transcodes them with **FFmpeg**, and deduplicates frames using **perceptual hashing** (`img_hash`).
3. Processed videos and event indexes are uploaded back to MinIO.
4. PostgreSQL is updated with processed paths and durations.

### Replay Flow

1. The **Web Dashboard** queries the Ingestion API for session lists and details.
2. It fetches processed video segments from MinIO (via presigned URLs).
3. The Canvas-based player renders video with an overlaid event timeline.

### Analytics Flow

1. The **Analytics API** reads from PostgreSQL.
2. It pre-computes heatmaps, funnel stages, and session statistics.
3. The Web Dashboard visualizes these aggregates.

---

## Component Interactions

### Ingestion API (`services/ingestion/`)

- **Role**: Entry point for all capture data.
- **Tech**: Go 1.22, Gin, PostgreSQL, MinIO (via S3 SDK).
- **Key Endpoints**:
  - `POST /v1/sessions/init` — start a session
  - `POST /v1/sessions/{id}/chunks` — upload video chunk
  - `POST /v1/sessions/{id}/events` — upload event batch
  - `POST /v1/sessions/{id}/complete` — finalize session
  - `GET /v1/sessions` — list sessions
- **Middleware**: CORS, API key authentication.

### Analytics API (`services/analytics/`)

- **Role**: Aggregated metrics and reporting.
- **Tech**: Go 1.22, Gin, PostgreSQL.
- **Key Endpoints**:
  - `GET /v1/analytics/heatmap` — click heatmap data
  - `GET /v1/analytics/funnel` — funnel conversion data
  - `GET /v1/analytics/sessions/stats` — session summary stats

### Video Processor (`services/processor/`)

- **Role**: Background worker for video pipeline.
- **Tech**: Rust, Tokio, FFmpeg, AWS S3 SDK, Redis.
- **Modules**:
  - `downloader.rs` — fetch chunks from MinIO
  - `encoder.rs` — transcode with FFmpeg
  - `deduplicator.rs` — perceptual-hash frame dedup
  - `uploader.rs` — store processed output
  - `indexer.rs` — build event-time indexes
  - `queue.rs` — Redis job queue consumer

### Privacy Engine (`services/privacy-engine/`)

- **Role**: PII detection and frame redaction.
- **Tech**: Rust, `image`, `regex`.
- **Capabilities**:
  - Detect credit cards, emails, passwords in OCR/text
  - Redact frames (blur, blackout, replace)
  - FFI bindings for SDK integration
- **Modules**:
  - `detector.rs` — PII regex detection
  - `redaction.rs` — image manipulation
  - `consent.rs` — user consent tracking
  - `audit.rs` — audit log helpers
  - `ffi.rs` — C-compatible exports

### Web Dashboard (`services/web/`)

- **Role**: Session replay and management UI.
- **Tech**: React 18, Vite, TypeScript, Canvas API.
- **Features**:
  - Session list with filters
  - Canvas-based video player
  - Event timeline overlay
  - GDPR export/delete actions

### Landing Page (`services/landing/`)

- **Role**: Marketing site.
- **Tech**: Next.js 14, Tailwind CSS, static export.

---

## Database Schema

### Core Tables

```sql
organizations
├── id UUID PK
├── name VARCHAR
├── plan VARCHAR (free/enterprise)
└── created_at TIMESTAMPTZ

projects
├── id UUID PK
├── org_id UUID FK
├── name VARCHAR
├── api_key_hash VARCHAR UNIQUE
├── privacy_config JSONB
├── retention_days INT
└── created_at TIMESTAMPTZ

sessions
├── id UUID PK
├── project_id UUID FK
├── user_id VARCHAR
├── duration_ms INT
├── video_path VARCHAR
├── event_count INT
├── error_count INT
├── metadata JSONB
├── status VARCHAR (capturing/processing/completed)
├── created_at TIMESTAMPTZ
├── completed_at TIMESTAMPTZ
└── processed_at TIMESTAMPTZ

events
├── id BIGSERIAL PK
├── session_id UUID FK
├── event_type VARCHAR
├── timestamp_ms INT
├── x INT
├── y INT
├── target VARCHAR
├── payload JSONB
└── created_at TIMESTAMPTZ

audit_logs
├── id BIGSERIAL PK
├── project_id UUID FK
├── action VARCHAR
├── actor VARCHAR
├── details JSONB
└── created_at TIMESTAMPTZ
```

### Indexes

- `idx_events_session` — fast event lookup per session
- `idx_sessions_project` — session list per project
- `idx_sessions_created` — time-range queries
- `idx_audit_logs_project` — audit queries per project

---

## API Contract Overview

### REST (OpenAPI 3.0)

The ingestion contract is defined in [`protocols/api-contracts/ingestion.yaml`](../protocols/api-contracts/ingestion.yaml).

**Authentication**: All endpoints require `X-API-Key` header.

### Protobuf

The capture schema is defined in [`protocols/capture-schema/session.proto`](../protocols/capture-schema/session.proto).

Messages:
- `FrameChunk` — video frame batch
- `Event` / `EventBatch` — user interaction events
- `SessionMetadata` — device and session context

---

## Deployment Topology (Production)

```
┌─────────────┐
│   CDN       │
│ (Landing)   │
└─────────────┘
       │
┌──────▼──────┐      ┌──────────────┐
│  Load Bal.  │─────▶│  Ingestion   │
└─────────────┘      │  API (x3)    │
       │             └──────────────┘
┌──────▼──────┐             │
│  Web Dash   │             ▼
│  (static)   │      ┌──────────────┐
└─────────────┘      │  PostgreSQL  │
                     │  (primary)   │
                     └──────────────┘
                            │
                     ┌──────────────┐
                     │    MinIO     │
                     │   (cluster)  │
                     └──────────────┘
                            │
                     ┌──────────────┐
                     │   Redis      │
                     │  (sentinel)  │
                     └──────────────┘
                            │
                     ┌──────────────┐
                     │  Processor   │
                     │  workers     │
                     └──────────────┘
```

- **Ingestion API**: Horizontally scalable stateless Go containers.
- **Processor**: Long-running Rust workers consuming Redis queue.
- **PostgreSQL**: Primary + read replicas recommended for analytics.
- **MinIO**: Distributed mode for high availability.

---

For deployment specifics, see [DEPLOYMENT.md](DEPLOYMENT.md).
For API usage examples, see [API.md](API.md).
