# Rust Quality Audit Report

**Project**: Chronoscope  
**Scope**: `services/processor/`, `services/privacy-engine/`, `packages/sdk-linux/`  
**Date**: 2026-04-25  
**Auditor**: Security & Quality Auditor (Rust)

---

## Unsafe Blocks

### Summary
Only one file contains `unsafe` code: `services/privacy-engine/src/ffi.rs`. There are **7 unsafe blocks** total.

| # | File | Line | Context | Issue |
|---|------|------|---------|-------|
| 1 | `services/privacy-engine/src/ffi.rs` | 10 | `CStr::from_ptr(config_json)` | Missing null check before dereference (C-001) |
| 2 | `services/privacy-engine/src/ffi.rs` | 26 | `&mut *engine` | Null check exists at line 25, but missing SAFETY comment |
| 3 | `services/privacy-engine/src/ffi.rs` | 28 | `slice::from_raw_parts_mut(frame_data, len)` | Integer overflow in `len` calculation (C-002) |
| 4 | `services/privacy-engine/src/ffi.rs` | 42 | `&mut *engine` | Null check exists, missing SAFETY comment |
| 5 | `services/privacy-engine/src/ffi.rs` | 43 | `CStr::from_ptr(text)` | Null check exists, missing SAFETY comment |
| 6 | `services/privacy-engine/src/ffi.rs` | 52 | `CString::from_raw(s)` | Null check exists, missing SAFETY comment |
| 7 | `services/privacy-engine/src/ffi.rs` | 60 | `Box::from_raw(engine)` | Null check exists, missing SAFETY comment |

### Assessment
- **All unsafe blocks are in the FFI boundary**, which is expected.
- **None have detailed `// SAFETY:` comments** explaining the upheld invariants (H-004).
- **One block dereferences a potentially null pointer** (line 10) — this is a critical defect.
- **One block relies on an arithmetic result (`height * stride`) that can overflow** (line 28) — this is a critical defect.

### Recommendation
1. Add explicit null guards before every raw pointer dereference.
2. Use checked arithmetic for buffer size calculations.
3. Add a `// SAFETY:` comment before each unsafe block documenting:
   - What invariant is being relied upon.
   - Who is responsible for upholding it (caller vs. callee).
   - Why the operation is sound given that invariant.

---

## unwrap()/expect() Usage

### Summary
The codebase contains several `unwrap()` / `unwrap_or()` / `unwrap_or_default()` / `expect()` calls. Below is a line-by-line inventory with risk assessment.

#### `services/processor/src/downloader.rs`
- **Line 16**: `list_resp.contents.unwrap_or_default();`  
  **Risk**: Low. Safe fallback for empty bucket.
- **Line 31**: `key.rsplit('/').next().unwrap_or(&key);`  
  **Risk**: Low. `rsplit` on a non-empty string always yields at least one element.

#### `services/processor/src/sync.rs`
- **Line 25**: `row.try_get(0).unwrap_or_default();`  
  **Risk**: Medium. Silently masks database schema/type mismatches (M-001).
- **Line 26**: `row.try_get::<_, i32>(1).unwrap_or(0) as u64;`  
  **Risk**: Medium. Silently masks nulls or type errors. Also casts `i32` to `u64` without range check.
- **Line 27**: `row.try_get(2).unwrap_or(0);`  
  **Risk**: Medium.
- **Line 28**: `row.try_get(3).unwrap_or(0);`  
  **Risk**: Medium.

#### `services/privacy-engine/src/ffi.rs`
- **Line 11**: `serde_json::from_str(&config_str).unwrap_or_default();`  
  **Risk**: High. Invalid JSON silently enables all detection features, violating caller intent (H-006).
- **Line 45**: `CString::new(result).map(|s| s.into_raw()).unwrap_or(std::ptr::null_mut());`  
  **Risk**: Low. Returns null on error, which is an idiomatic C ABI failure signal, but should be documented.

#### `services/privacy-engine/src/detector.rs`
- **Line 30**: `Regex::new(r"...").unwrap();`  
  **Risk**: Low. Hardcoded pattern, compile-time verified.
- **Line 42**: `Regex::new(r"...").unwrap();`  
  **Risk**: Low. Hardcoded pattern.
- **Line 54**: `Regex::new(r"...").unwrap();`  
  **Risk**: Low. Hardcoded pattern.
- **Line 67**: `Regex::new(r"...").unwrap();`  
  **Risk**: Low. Hardcoded pattern.

#### `packages/sdk-linux/src/upload.rs`
- **Line 47**: `response.text().await.unwrap_or_default();`  
  **Risk**: Low. Used only for error message formatting.
- **Line 69**: `response.text().await.unwrap_or_default();`  
  **Risk**: Low. Used only for error message formatting.

#### `packages/sdk-linux/tests/integration_tests.rs`
- **Line 18**: `detect_display_server().unwrap();`  
  **Risk**: Acceptable. Test code.
- **Line 27**: `detect_display_server().unwrap();`  
  **Risk**: Acceptable. Test code.

#### `packages/sdk-linux/src/buffer.rs` (unit tests)
- **Line 64**: `buf.read_chunk().unwrap();`  
  **Risk**: Acceptable. Test code.
- **Line 75**: `buf.read_chunk().unwrap();`  
  **Risk**: Acceptable. Test code.

### Assessment
- **Production paths with `unwrap_or_default()` masking errors**: `sync.rs` (4x), `ffi.rs` (1x).
- **Hardcoded-regex `unwrap()` calls**: Acceptable because the patterns are static literals.
- **Tests**: `unwrap()` usage is acceptable in test code.

### Recommendation
1. Replace `unwrap_or_default()` in `sync.rs` with `?` error propagation.
2. In `ffi.rs`, return `null_mut()` on JSON parse failure instead of defaulting.
3. Consider pre-compiling hardcoded regexes with `once_cell::Lazy` to remove even the low-risk `unwrap()` calls and improve performance.

---

## Error Handling

### Summary
The project predominantly uses `anyhow::Result` for error propagation, which is a reasonable choice for application code. However, there are notable inconsistencies and anti-patterns.

#### `services/processor/src/sync.rs`
- **Pattern**: `row.try_get::<_, i32>(1).unwrap_or(0) as u64`
- **Problem**: Database type mismatches or nulls are silently swallowed. If `timestamp_ms` is `NULL`, it becomes `0`, corrupting the event timeline without any warning.
- **Fix**: Propagate the error:
  ```rust
  timestamp_ms: row.try_get::<_, i64>(1)? as u64,
  ```

#### `services/processor/src/downloader.rs`
- **Pattern**: `object.key.ok_or_else(|| anyhow::anyhow!("missing object key"))?;`
- **Assessment**: Good. Explicitly converts `Option` to `Result`.

#### `services/processor/src/encoder.rs`
- **Pattern**: `image::open(path)` with `continue` on error.
- **Assessment**: Good. Warns and skips malformed frames rather than failing the entire pipeline.
- **Gap**: No handling for dimension mismatch between frames (see H-001).

#### `services/privacy-engine/src/ffi.rs`
- **Pattern**: `serde_json::from_str(&config_str).unwrap_or_default();`
- **Problem**: Failures are silently converted to defaults (H-006).
- **Fix**: Return an error indicator (null pointer) to the caller.

#### `services/privacy-engine/src/detector.rs`
- **Pattern**: `if let Ok(re) = Regex::new(pattern) { ... }`
- **Assessment**: Good for custom patterns. Errors are silently ignored, which is acceptable for user-supplied patterns, but could be logged.

### Recommendation
1. Adopt a consistent policy: **never silently default on I/O or database errors**.
2. Use structured errors (e.g., `thiserror`) for library crates (`privacy-engine`, `sdk-linux`) so callers can match on specific error types.
3. Log every skipped frame, invalid regex, and parse failure at `warn` level or higher.

---

## FFI Safety

### Summary
The `privacy-engine` exposes a C ABI via `ffi.rs`. The interface is:

| Function | Null Checks | `Box::into_raw` / `Box::from_raw` Pair | Safety Docs |
|----------|-------------|----------------------------------------|-------------|
| `chronoscope_privacy_init` | ❌ No | N/A (creates) | ❌ |
| `chronoscope_privacy_process_frame` | ✅ Yes | N/A | ❌ |
| `chronoscope_privacy_process_text` | ✅ Yes | N/A | ❌ |
| `chronoscope_privacy_free_string` | ✅ Yes | N/A (frees string) | ❌ |
| `chronoscope_privacy_free` | ✅ Yes | ✅ Yes (mirrors init) | ❌ |

### Specific Issues
1. **Missing null check in `init`** (C-001).
2. **Integer overflow in `process_frame`** (C-002).
3. **Silent JSON parse failure** (H-006).
4. **No validation of `width/height/stride`** (H-005).
5. **CString conversion failure ambiguity** (L-005).

### Recommendation
1. Add `#[deny(clippy::missing_safety_doc)]` to the crate.
2. Document every function with `/// # Safety` rustdoc explaining preconditions.
3. Consider using a crate like `safer_ffi` or `cbindgen` with safer wrapper generation to reduce manual unsafe code.

---

## Resource Management

### Summary

#### RAII Usage
- **Database connections** (`deadpool_postgres`): ✅ Correctly pooled and returned automatically.
- **Redis connection** (`MultiplexedConnection`): ✅ Cloned/shared, no leak.
- **S3 clients** (`aws_sdk_s3`): ✅ Reused across requests.
- **FFI engine allocation**: ✅ Properly paired `Box::into_raw` / `Box::from_raw`.
- **FFI string allocation**: ✅ Properly paired `CString::into_raw` / `CString::from_raw`.

#### Resource Leaks
- **`services/processor/src/downloader.rs`**: Intentionally leaks `TempDir` (C-005). All downloaded S3 objects persist in `/tmp`.
- **`services/processor/src/encoder.rs`**: Encoded MP4 is written to temp dir but never explicitly deleted after upload. If the processor crashes between encoding and upload, the file remains.
- **`packages/sdk-linux/src/capture/x11.rs`**: X11 connection is never closed because the loop never breaks (M-003).

#### Panic Safety
- **`tokio::task::spawn_blocking` tasks**: If the blocking task panics, `await`ing the JoinHandle returns an error, which is propagated in `encode_h264`, `generate_index`, and `deduplicate`. This is safe.
- **TempDir on panic**: If a panic occurs inside `downloader.rs` before the leak, the `TempDir` is dropped and cleaned up. After the leak, cleanup is impossible.

### Recommendation
1. Remove the `Box::leak` in `downloader.rs` and pass the `TempDir` ownership through the pipeline.
2. Add a cleanup step in `process_session` (or a `Drop` guard) to remove the encoded MP4 after upload.
3. Implement cancellation tokens for capture loops so connections are closed promptly.

---

## Test Coverage Gaps

### `services/processor/` — Zero Tests
- **Status**: ❌ **NO TESTS**
- **Files without coverage**:
  - `main.rs` — No unit or integration tests.
  - `downloader.rs` — No mock S3 tests.
  - `encoder.rs` — No FFmpeg encoding tests.
  - `deduplicator.rs` — No perceptual hash tests.
  - `indexer.rs` — No keyframe extraction tests.
  - `sync.rs` — No database synchronization tests.
  - `uploader.rs` — No mock S3 upload tests.
  - `db.rs` — No database update tests.
  - `queue.rs` — No Redis queue tests.
  - `config.rs` — No configuration parsing tests.
- **Risk**: The entire video processing pipeline is unverified. Changes to FFmpeg parameters, S3 SDK, or PostgreSQL schema can break production silently.
- **Recommendation**: Add `#[cfg(test)]` modules using `mockall` or `wiremock` for external services. For FFmpeg, include a small fixture video in `tests/fixtures/`.

### `services/privacy-engine/` — Minimal Tests
- **File**: `tests/integration_tests.rs`
- **Coverage**:
  - ✅ Email detection
  - ✅ Credit card detection
  - ❌ SSN detection
  - ❌ Password detection
  - ❌ Custom pattern detection
  - ❌ Frame-based redaction (`process_frame`)
  - ❌ FFI boundary (C ABI round-trip)
  - ❌ Overlapping detection handling
  - ❌ Blur/Blackout/Replace modes
- **Risk**: The most critical privacy features (frame redaction, FFI) have no automated verification.
- **Recommendation**: Add tests for all `RedactionMode` variants, overlapping detections, and a C-side test harness for the FFI.

### `packages/sdk-linux/` — Partial Tests
- **File**: `tests/integration_tests.rs`
- **Coverage**:
  - ✅ Display server detection (X11 / Wayland)
  - ✅ CircularBuffer basic ops
  - ❌ CircularBuffer concurrent access (thread safety)
  - ❌ `ChunkUploader` upload and finalize
  - ❌ `LinuxCapture` start/stop lifecycle
  - ❌ X11 / Wayland capture (even stub validation)
  - ❌ Input capture stub
- **Risk**: The SDK's core upload and capture paths are untested.
- **Recommendation**: Add `mockito` or `wiremock` tests for `ChunkUploader`. Add cancellation and lifecycle tests for `LinuxCapture`.

### `packages/sdk-linux/src/buffer.rs` — Unit Tests Only
- **Coverage**:
  - ✅ Basic write/read
  - ✅ Wrap-around
  - ❌ Empty read
  - ❌ Full buffer overwrite
  - ❌ Concurrent read/write (thread safety)
- **Recommendation**: Add property-based tests (e.g., `proptest`) for the circular buffer to catch edge cases.

---

## Additional Quality Observations

### Async Patterns
- **Good**: CPU-intensive work (encoding, hashing, indexing) is offloaded to `tokio::task::spawn_blocking`, preventing event-loop blocking.
- **Bad**: `x11.rs` uses `tokio::time::interval` inside a `loop` with no cancellation, making graceful shutdown impossible (M-003).

### Dependencies
- `ffmpeg-next` (GPL/LGPL depending on FFmpeg build) — Ensure compliance with license obligations in distribution.
- `regex` — Up-to-date (1.10), but lacks ReDoS protection for custom patterns (H-003).
- `aws-sdk-s3` (1.15) and `aws-config` (1.1) — Modern versions, good.

### Code Clutter
- Large blocks of commented-out code in `x11.rs` and `wayland.rs` (L-001, L-002) should be removed. They belong in a roadmap or design document, not source control.

### Dead Code / Stubs
- `packages/sdk-linux/src/input.rs` — Empty stub.
- `services/privacy-engine/src/consent.rs` — Always returns `Granted`.
- `services/privacy-engine/src/audit.rs` — Prints to stderr only.
- These are acknowledged as MVP stubs but should be tracked in a backlog or annotated with `#[allow(dead_code)]` and a `TODO` issue reference.

---

## Overall Quality Rating

| Category         | Rating | Notes |
|------------------|--------|-------|
| Error Handling   | ⚠️ C   | Silently swallows DB and JSON errors |
| FFI Safety       | ❌ D   | Null dereference, overflow, missing docs |
| Resource Mgmt    | ⚠️ C   | Intentional TempDir leak, uncancelable loops |
| Test Coverage    | ❌ D   | Processor has zero tests; others are thin |
| Documentation    | ⚠️ C   | Many public APIs undocumented |
| Code Clarity     | ⚠️ C   | Commented-out code, stubs without tracking |
| Async Correctness| ✅ B+  | Good use of `spawn_blocking`, poor shutdown |
