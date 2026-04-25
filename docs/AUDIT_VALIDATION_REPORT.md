# Audit Validation Report

**Date**: 2026-04-25
**Scope**: All 8 audit files in `docs/audits/`

## Summary

| Audit | Already Fixed | Partially Fixed | Still Valid |
|-------|--------------|-----------------|-------------|
| SECURITY_AUDIT_GO.md | 7 | 1 | 12 |
| QUALITY_AUDIT_GO.md | 4 | 2 | 14 |
| SECURITY_AUDIT_RUST.md | 18 | 1 | 12 |
| QUALITY_AUDIT_RUST.md | 18 | 1 | 30+ |
| SECURITY_AUDIT_SWIFT.md | 6 | 0 | 11 |
| QUALITY_AUDIT_SWIFT.md | 12 | 1 | 26 |
| SECURITY_AUDIT_INFRA.md | 14 | 1 | 17 |
| QUALITY_AUDIT_INFRA.md | 9 | 3 | 20 |

---

## Go Services (`services/ingestion/`, `services/analytics/`)

### Still Valid
- **M-001**: No global request body size limits
- **M-002**: Missing database query timeouts
- **M-003**: Hardcoded MinIO `Secure: false`
- **M-004**: Inconsistent row-scan failure strategies
- **M-005**: No maximum event batch size
- **M-006**: No maximum chunk index validation
- **M-007**: Generated session token never validated
- **M-008**: Transaction/storage consistency gap in `DeleteUserData`
- **L-001**: Dead code — unused `storage/minio.go` wrapper
- **L-002**: Ignored `RowsAffected` error
- **L-003**: Duplicate CORS and auth middleware across services
- **LINT-001**: Non-canonical import ordering (partial — 4 files still wrong)
- **LINT-002**: Struct tag alignment in `stats.go`
- **DEAD-001**: Unused `storage` package
- **DEAD-002**: Session token generated but never consumed
- **STYLE-001**: Inconsistent slice initialization
- **STYLE-002**: Duplicate middleware code across services
- **ERR-001**: Silent suppression of `rows.Scan` errors
- **ERR-002**: Inconsistent row-scan failure strategies
- **ERR-004**: `RowsAffected` error discarded
- **ERR-005**: Missing `eventRows` defer close
- **TEST-002**: Zero unit tests in analytics service
- **Q-001**: Magic numbers (hardcoded pagination/TTL)
- **Q-002**: Missing connection pool configuration
- **Q-003**: Missing `Content-Type` validation on JSON endpoints

---

## Rust Services (`services/processor/`, `services/privacy-engine/`, `packages/sdk-linux/`)

### Still Valid
- **M-001**: Database field errors silently defaulted in `sync.rs`
- **M-003**: X11 capture task runs forever without cancellation
- **M-004**: Missing UUID validation before database cast
- **M-005**: Redaction function ignores height parameter
- **M-006**: Missing documentation on public APIs
- **M-008**: Regexes recompiled on every text scan
- **M-009**: Consent mechanism is a non-functional stub
- **L-001**: Large commented-out code blocks in X11 capture
- **L-002**: Large commented-out code blocks in Wayland capture
- **L-004**: Default API key is empty string
- **L-006**: Redundant integration tests for CircularBuffer
- **Unwrap**: Multiple `unwrap_or_default()`/`unwrap()` in production paths
- **Resource leaks**: `encoder.rs` temp file not cleaned up, `x11.rs` uncancelable loop
- **Tests**: Many modules lack unit/integration tests
- **Code clutter**: Commented-out blocks, stubs without tracking

---

## Swift SDK (`packages/sdk-macos/`)

### Still Valid
- **M-001**: Force unwraps in multipart body construction
- **M-003**: Redundant `NSLock` inside actor
- **M-004**: Session ID not explicitly URL-encoded in path
- **M-005**: Integer overflow in buffer capacity calculation
- **M-006**: Hardcoded user identifier (`"macos_user"`)
- **M-007**: Unsafe build flag links to local relative path
- **M-008**: Insufficient test coverage
- **L-001**: Print statements instead of structured logging
- **L-002**: Missing documentation on public APIs
- **L-003**: Errors swallowed by `try?` in upload loop
- **L-004**: FrameCapture missing deinit stream cleanup
- **F2**: Force unwraps in `ChunkUploader`
- **M6**: SCStream deinit gap
- **M7**: CircularBuffer double synchronization
- **E1/E2/E3**: Errors swallowed/printed instead of propagated
- **A1/A2/A3**: Missing validation, docs, userId in CaptureConfig
- **T1..T9**: Missing tests for all major components
- **D1/D2**: Dead code (CaptureQuality enum, captureFailed error)

---

## Infrastructure (`services/web/`, `services/landing/`, `docker/`, `.github/`)

### Still Valid
- **C-001**: API key exposed in client bundle (architectural issue remains)
- **M-001**: Internal services bound to host ports in compose
- **M-003**: No centralized error handling in API client
- **M-004**: `setInterval` in VideoPlayer
- **M-005**: Potential workflow command injection via commit messages
- **M-006**: No `.env.example` for web dashboard
- **M-007**: Bind mount uses parent directory traversal
- **M-010**: Release script pushes without verifying remote state
- **L-001**: Excessive inline styles
- **L-002**: `key` prop relies on array index
- **L-003**: Interactive divs lack accessibility attributes
- **L-004**: `sed -i` portability issue
- **L-005**: No `HEALTHCHECK` in Dockerfiles
- **L-006**: Makefile `proto` lacks prerequisite checks
- **L-007**: Makefile `test` stops on first failure
- **L-008**: Button without explicit `type`
- **L-009**: Error objects swallowed in dashboard
- **Missing ESLint config**: Config files present but no `.eslintrc`
- **Missing tests**: No test script in web package.json
- **Containers run as root**: No `USER` instruction
- **No HEALTHCHECK**: All Dockerfiles missing
- **No frontend testing docs**: CONTRIBUTING.md lacks test instructions

---

## Next Steps

All `STILL_VALID` findings will be addressed in **Faz 1** (Go, Rust, Swift, Infra fix agents), followed by test writing, lint fixing, CSS refactoring, and documentation updates.
