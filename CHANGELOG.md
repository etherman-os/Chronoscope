# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-04-25

### Security
- Hash incoming API keys with SHA-256 before database comparison
- Enforce project ownership checks on all session mutation endpoints
- Add per-API-key rate limiting (100 req/min) to ingestion and analytics APIs
- Restrict CORS origins via `CORS_ALLOWED_ORIGIN` environment variable
- Enforce maximum chunk size (2 MiB), chunk index (10000), and event batch size (1000)
- Add request body size limits (8 MiB ingestion, 1 MiB analytics)
- Sanitize session IDs to prevent path traversal in processor output filenames
- Remove hardcoded database and AWS credential fallbacks from processor config
- Rotate default development API key to a random 32-hex value
- Add Content-Security-Policy headers to web dashboard and landing page
- Remove hardcoded API key fallback from web dashboard
- Regenerate `go.sum` via `go mod tidy`

### Quality
- Convert Chronoscope Swift singleton to `actor` to prevent race conditions
- Add preconditions to prevent zero or negative buffer capacity crashes
- Replace force unwrap with failable initializer in `PrivacyEngine`
- Apply privacy engine to raw frames before JPEG encoding and add `SCStreamDelegate`
- Add graceful shutdown signal handling and Redis reconnect loop to processor
- Clone source buffer before blur to prevent reading already-blurred pixels
- Limit custom regex size and complexity to mitigate ReDoS
- Merge overlapping text detections to prevent `replace_range` panic
- Skip frames with mismatched dimensions instead of panicking in encoder
- Prevent temp directory leak by returning `TempDir` to caller in downloader
- Fix FFI null checks, prevent integer overflow, validate dimensions, and add `SAFETY` comments
- Log audit errors instead of silently discarding them
- Remove `project_id` query parameter and enforce auth context in `ListSessions`
- Add `.dockerignore` files and pin base image tags
- Add missing ESLint and TypeScript-ESLint dependencies to web dashboard
- Source project ID from environment variable instead of hardcoding
- Add explicit `URLSession` timeouts for uploads and session init
- Fix Go test build errors and non-canonical import ordering

### Tests
- Add E2E integration tests for full session lifecycle
- Add Go unit tests for ingestion handlers, middleware, and models
- Add Go unit tests for analytics handlers
- Add Rust unit and integration tests for processor and privacy engine
- Add Swift tests for macOS SDK
- Add web dashboard component and API client tests (Vitest)
- Add CI jobs for lint, build, test, and dependency audit across all services

### Documentation
- Add comprehensive security audit reports (Go, Rust, Swift, Infra)
- Add quality audit reports (Go, Rust, Swift, Infra)
- Add `BUILD_REPORT.md` and `SMOKE_TEST.md` for Phase 7 security fixes
- Add `AUDIT_VALIDATION_REPORT.md` tracking remediation status
- Rewrite all documentation to be accurate and up-to-date with latest code

## [0.1.0] - 2026-04-25

### Added
- macOS SDK (Swift + ScreenCaptureKit)
- Windows SDK (C++20 + WinRT Graphics Capture)
- Linux SDK (Rust + PipeWire/X11)
- Ingestion API (Go + Gin)
- Video Processor (Rust + FFmpeg)
- Privacy Engine (Rust C ABI)
- Replay Dashboard (React + Vite)
- Analytics API (Go + PostgreSQL)
- Landing Page (Next.js)
- GDPR compliance endpoints
- Docker Compose local development stack
- Protobuf capture schema

### Security
- API key authentication
- Basic rate limiting middleware
- CORS restrictions
- Input validation

### Known Limitations
- Video encoding requires FFmpeg (included in Docker image)
- Windows SDK builds only on CI windows-latest runner
- macOS SDK requires macOS 12.3+ for ScreenCaptureKit
