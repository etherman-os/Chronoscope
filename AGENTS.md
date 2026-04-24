# Agent Instructions

## Overview
Chronoscope is a session replay infrastructure for native desktop applications (macOS, Windows, Linux).

## Technology Stack
- Backend: Go + Gin
- Processor: Rust + FFmpeg
- Frontend: React + Vite + Next.js
- SDKs: Swift (macOS), C++20 (Windows), Rust (Linux)
- Infrastructure: Docker Compose, PostgreSQL, Redis, MinIO

## Development
- All commits use Conventional Commits format
- Security audit reports in `docs/audits/`
- E2E tests via `docker-compose.test.yml`
- Release process: `scripts/release.sh`
