# Chronoscope Architecture

This document describes the high-level design, data flow, component interactions, and security boundaries of the Chronoscope platform.

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

```mermaid
flowchart LR
    subgraph Capture
        SDK[Desktop SDK]
    end

    subgraph Ingestion
        API[Ingestion API]
        DB[(PostgreSQL)]
        S3[MinIO Storage]
    end

    subgraph Processing
        VP[Video Processor]
        PE[Privacy Engine]
    end

    subgraph Consumption
        WD[Web Dashboard]
        AA[Analytics API]
    end

    SDK -->|HTTP/REST| API
    API --> DB
    API --> S3
    S3 --> VP
    VP --> PE
    PE --> S3
    VP --> DB
    WD --> API
    WD --> AA
    AA --> DB
```

### Capture Flow

1. The **SDK** (Swift / C++ / Rust) captures frames and events locally.
2. It buffers data and uploads it to the **Ingestion API** via HTTP/REST.
3. The API stores session metadata in **PostgreSQL** and raw video chunks in **MinIO**.
4. A message is pushed to **Redis** to notify the **Processor**.

### Processing Flow

1. The **Processor** (Rust) polls the Redis queue for new jobs.
2. It downloads chunks from MinIO, transcodes them with **FFmpeg**, and deduplicates frames using **perceptual hashing**.
3. The **Privacy Engine** detects and redacts PII in frames (blur, blackout, replace).
4. Processed videos and event indexes are uploaded back to MinIO.
5. PostgreSQL is updated with processed paths and durations.

### Replay Flow

1. The **Web Dashboard** queries the Ingestion API for session lists and details.
2. It fetches processed video segments from MinIO (via presigned URLs).
3. The Canvas-based player renders video with an overlaid event timeline.

### Analytics Flow

1. The **Analytics API** reads from PostgreSQL.
2. It pre-computes heatmaps, funnel stages, and session statistics.
3. The Web Dashboard visualizes these aggregates.

---

## Component Diagram

```mermaid
flowchart TB
    subgraph Client
        A[macOS App]
        B[Windows App]
        C[Linux App]
    end

    subgraph Edge
        N[Nginx / Load Balancer]
    end

    subgraph Services
        I[Ingestion API<br/>Go + Gin]
        An[Analytics API<br/>Go + Gin]
        W[Web Dashboard<br/>React + Vite]
        L[Landing Page<br/>Next.js]
    end

    subgraph Data
        P[(PostgreSQL)]
        R[Redis]
        M[MinIO]
    end

    subgraph Workers
        Pr[Video Processor<br/>Rust + FFmpeg]
        Pe[Privacy Engine<br/>Rust C ABI]
    end

    A --> N
    B --> N
    C --> N
    N --> I
    N --> An
    N --> W
    N --> L
    I --> P
    I --> M
    An --> P
    W --> I
    W --> An
    M --> Pr
    Pr --> Pe
    Pe --> M
    Pr --> P
    I --> R
    Pr --> R
```

---

## Database Schema

```mermaid
erDiagram
    ORGANIZATIONS ||--o{ PROJECTS : has
    PROJECTS ||--o{ SESSIONS : has
    SESSIONS ||--o{ EVENTS : contains
    PROJECTS ||--o{ AUDIT_LOGS : generates

    ORGANIZATIONS {
        uuid id PK
        string name
        string plan
        timestamp created_at
    }

    PROJECTS {
        uuid id PK
        uuid org_id FK
        string name
        string api_key_hash
        jsonb privacy_config
        int retention_days
        timestamp created_at
    }

    SESSIONS {
        uuid id PK
        uuid project_id FK
        string user_id
        int duration_ms
        string video_path
        int event_count
        int error_count
        jsonb metadata
        string status
        timestamp created_at
        timestamp completed_at
        timestamp processed_at
    }

    EVENTS {
        bigint id PK
        uuid session_id FK
        string event_type
        int timestamp_ms
        int x
        int y
        string target
        jsonb payload
        timestamp created_at
    }

    AUDIT_LOGS {
        bigint id PK
        uuid project_id FK
        string action
        string actor
        jsonb details
        timestamp created_at
    }
```

### Indexes

- `idx_events_session` — fast event lookup per session
- `idx_sessions_project` — session list per project
- `idx_sessions_created` — time-range queries
- `idx_audit_logs_project` — audit queries per project
- `idx_audit_logs_created` — audit time-range queries

---

## API Contract Overview

### REST

The ingestion contract is defined in the Go handler source code under `services/ingestion/internal/handlers/` and `services/analytics/internal/handlers/`.

**Authentication**: All endpoints require the `X-API-Key` header. The key is hashed with SHA-256 before comparison against the `projects.api_key_hash` column.

### Protobuf

The capture schema is defined in [`protocols/capture-schema/session.proto`](../protocols/capture-schema/session.proto).

Messages:
- `FrameChunk` — video frame batch
- `Event` / `EventBatch` — user interaction events
- `SessionMetadata` — device and session context

---

## Deployment Topology (Production)

```mermaid
flowchart TB
    CDN[CDN<br/>Landing Page]
    LB[Load Balancer<br/>Nginx + TLS]

    subgraph API_Tier
        I1[Ingestion API]
        I2[Ingestion API]
        I3[Ingestion API]
        A1[Analytics API]
    end

    subgraph Data_Tier
        PG[(PostgreSQL<br/>Primary)]
        PG_R[(PostgreSQL<br/>Read Replica)]
        RD[Redis<br/>Sentinel]
        MN[MinIO<br/>Distributed]
    end

    subgraph Worker_Tier
        P1[Processor]
        P2[Processor]
    end

    CDN --> LB
    LB --> I1
    LB --> I2
    LB --> I3
    LB --> A1
    I1 --> PG
    I2 --> PG
    I3 --> PG
    I1 --> MN
    I2 --> MN
    I3 --> MN
    A1 --> PG_R
    PG --> RD
    RD --> P1
    RD --> P2
    MN --> P1
    MN --> P2
    P1 --> MN
    P2 --> MN
    P1 --> PG
    P2 --> PG
```

- **Ingestion API**: Horizontally scalable stateless Go containers.
- **Analytics API**: Read-only queries; scales independently.
- **Processor**: Long-running Rust workers consuming the Redis queue.
- **PostgreSQL**: Primary + read replicas recommended for analytics.
- **MinIO**: Distributed mode for high availability.

---

## Security Boundaries

```
+------------------+
| Public Internet  |
| (TLS terminated) |
+------------------+
         |
+------------------+
| DMZ / Edge       |
| - Nginx          |
| - CDN            |
+------------------+
         |
+------------------+
| Application      |
| - Ingestion API  |
| - Analytics API  |
| - Web Dashboard  |
+------------------+
         |
+------------------+
| Data (Private)   |
| - PostgreSQL     |
| - Redis          |
| - MinIO          |
+------------------+
         |
+------------------+
| Processing       |
| - Video Processor|
| - Privacy Engine |
+------------------+
```

### Trust Boundaries

1. **SDK to Ingestion API**: Authenticated via `X-API-Key`. TLS required in production.
2. **API to Database**: PostgreSQL and MinIO credentials stored as environment variables.
3. **Processor to Storage**: Processor uses IAM-style credentials with least-privilege bucket access.
4. **Dashboard to Video**: Video URLs are served via MinIO presigned URLs; no direct bucket access.

---

For deployment specifics, see [DEPLOYMENT.md](DEPLOYMENT.md).
For API usage examples, see [API.md](API.md).
For security details, see [SECURITY.md](SECURITY.md).
