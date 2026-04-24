# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-04-25

### Validation
- Pre-release validation passed:
  - Git history credential scan: CLEAN (no leaked secrets)
  - E2E test: PASSED (full session lifecycle)
  - Rust unit tests: 10/10 PASSED
  - Go unit tests: middleware 29.8% coverage, handlers 30.1% coverage
  - Dev API key rotated to random 32-hex value
  - `go.sum` regenerated via `go mod tidy`
  - FFmpeg verified in processor Dockerfile

### Added
- macOS SDK (Swift + ScreenCaptureKit)
- Windows SDK (C++20 + WinRT)
- Linux SDK (Rust + PipeWire/X11)
- Ingestion API (Go + Gin)
- Video Processor (Rust + FFmpeg)
- Privacy Engine (Rust C ABI)
- Replay Dashboard (React + Vite)
- Analytics API (Go + PostgreSQL)
- Landing Page (Next.js)
- GDPR compliance endpoints
- Load testing (k6)

### Security
- Security audit completed
- 12 CRITICAL and 26 HIGH findings fixed
- API key authentication
- Rate limiting
- CORS restrictions
- Input validation

### Known Limitations
- Test coverage ~30.1% (MVP baseline, more tests in next iteration)
- Video encoding requires FFmpeg (included in Docker image)
- Windows SDK builds only on CI windows-latest runner
- macOS SDK requires macOS 12.3+ for ScreenCaptureKit
