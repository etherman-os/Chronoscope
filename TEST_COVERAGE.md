# Test Coverage Report

This report tracks the automated checks expected to pass for the current repository state. Use `make test` for the broad local suite.

## Verified Test Commands

| Component | Command | Status |
|-----------|---------|--------|
| Ingestion API | `cd services/ingestion && go test ./...` | Passing |
| Analytics API | `cd services/analytics && go test ./...` | Passing |
| Shared middleware | `cd pkg/middleware && go test ./...` | Passing |
| Web dashboard | `cd services/web && npm test` | Passing |
| Processor | `cd services/processor && cargo test --locked` | Passing |
| Privacy engine | `cd services/privacy-engine && cargo test --locked` | Passing |
| Linux SDK | `cd packages/sdk-linux && cargo test --locked` | Passing |
| Demo session CLI | `cd packages/sdk-linux && cargo test --locked --bin chronoscope-demo-session` | Covered by Linux SDK test command |

## Manual / Platform-Gated Checks

| Scenario | Status |
|----------|--------|
| Linux X11 recording via `make record-linux` | Requires a Linux X11 desktop session |
| macOS SDK build/tests | macOS-only; not verified on Linux CI/local Linux |
| Windows SDK build/tests | Experimental; CI build remains roadmap |
| Synthetic replay E2E via `make demo-session` | Verified locally through ingestion, processor, and video GET |
| Full browser replay E2E with real X11 capture | Covered by smoke guide; should be run before release |
