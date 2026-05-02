# Audit Validation Report

**Date**: 2026-05-02

This file reflects the current README-scope validation pass. Older audit files in `docs/audits/` remain useful historical input, but several of their findings have since been fixed and should not be read as the current state without revalidation.

## Current Validation Summary

| Area | Current status |
|------|----------------|
| Ingestion API | Tests pass; session, chunk, event, complete, video, GDPR, auth, rate-limit, CORS, and readiness paths are covered. |
| Analytics API | Tests pass for stats, heatmap, funnel, middleware, and router wiring. |
| Shared middleware | Tests pass for auth and CORS behavior. |
| Processor | Tests pass for processing helpers, deduplication, encoding, indexing, S3 client setup, and a mocked full pipeline. |
| Privacy engine | Tests pass for text PII detection, FFI null safety, custom patterns, redaction bounds, and consent state. |
| Linux SDK | Tests pass for buffer behavior, X11 no-display handling, cancellable Wayland stub, uploader error paths, and lifecycle setup. |
| Web dashboard | Tests, lint, and TypeScript checks pass with API calls mocked in component tests. |
| Synthetic replay E2E | Verified locally through ingestion, MinIO chunk storage, Redis queueing, processor MP4 publishing, session `ready` status, and authenticated video GET. |

## Remaining README-Scope Gaps

- Linux X11 is the only verified capture path. Wayland/PipeWire is still a cancellable stub and should remain documented as planned.
- macOS and Windows SDKs are beta/experimental. They should not be marketed as production-complete until their capture, upload, privacy, and lifecycle paths are verified on native CI.
- Frame-level OCR/redaction is not implemented in the processor. The privacy engine currently supports text-level PII detection and redaction primitives.
- Production Compose is a self-hosted starting point, not a full enterprise deployment. Operators still need TLS termination, secret management, backups, and monitoring.
- Full `make up -> seed-local -> record-linux -> replay` validation with real X11 capture should be run on a Linux X11 desktop before a release.

## Commands Used In This Pass

- `go test ./...` in `services/ingestion`
- `go test ./...` in `services/analytics`
- `go test ./...` in `pkg/middleware`
- `npm test`, `npm run lint`, and `npx tsc --noEmit` in `services/web`
- `cargo test --locked` in `services/processor`
- `cargo test --locked` in `services/privacy-engine`
- `cargo test --locked` in `packages/sdk-linux`
- `make up`, `make seed-local`, `make demo-session ENDPOINT=http://localhost:18080 DEMO_FRAMES=12 FPS=4`
- Authenticated `GET /v1/sessions/:id` returned `status: ready`
- Authenticated `GET /v1/sessions/:id/video` returned `200 OK` with `Content-Type: video/mp4`
