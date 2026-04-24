# Rust Security Audit Report

**Project**: Chronoscope  
**Scope**: `services/processor/`, `services/privacy-engine/`, `packages/sdk-linux/`  
**Date**: 2026-04-25  
**Auditor**: Security & Quality Auditor (Rust)

---

## Summary of Findings

| Severity | Count |
|----------|-------|
| CRITICAL | 6 |
| HIGH     | 8 |
| MEDIUM   | 10 |
| LOW      | 6 |

---

## CRITICAL Findings

### [C-001] Null Pointer Dereference in FFI Init Function
- **File**: `services/privacy-engine/src/ffi.rs`
- **Line**: 10
- **Code**:
  ```rust
  let config_str = unsafe { CStr::from_ptr(config_json).to_string_lossy() };
  ```
- **Issue**: `chronoscope_privacy_init` does not validate that `config_json` is non-null before dereferencing it. A C caller passing `NULL` causes an immediate segmentation fault.
- **Impact**: Denial of service (crash) of any host application. If the engine is embedded in a critical process, this is a total availability failure.
- **Fix**: Add an explicit null guard before the `unsafe` block:
  ```rust
  if config_json.is_null() {
      return std::ptr::null_mut();
  }
  ```

### [C-002] Integer Overflow in FFI Frame Buffer Size Calculation
- **File**: `services/privacy-engine/src/ffi.rs`
- **Line**: 27–28
- **Code**:
  ```rust
  let len = (height * stride) as usize;
  let frame = unsafe { std::slice::from_raw_parts_mut(frame_data, len) };
  ```
- **Issue**: `height` and `stride` are `u32`. Their product can overflow the 32-bit integer before the cast to `usize`, yielding a tiny slice length. The subsequent `engine.process_frame` then reads/writes out of bounds on the actual frame buffer.
- **Impact**: Memory corruption, potential arbitrary code execution, or information disclosure.
- **Fix**: Use checked multiplication and abort on overflow:
  ```rust
  let len = usize::from(height)
      .checked_mul(usize::from(stride))
      .expect("frame dimension overflow");
  if len == 0 {
      return;
  }
  let frame = unsafe { std::slice::from_raw_parts_mut(frame_data, len) };
  ```

### [C-003] Hardcoded Database Credentials in Fallback
- **File**: `services/processor/src/config.rs`
- **Line**: 18
- **Code**:
  ```rust
  let db_url = std::env::var("DATABASE_URL")
      .unwrap_or_else(|_| "postgres://chronoscope:chronoscope@localhost:5432/chronoscope".to_string());
  ```
- **Issue**: The application falls back to a hardcoded database username and password (`chronoscope` / `chronoscope`) if the `DATABASE_URL` environment variable is absent.
- **Impact**: Information disclosure and unauthorized database access in any deployment where the operator forgets to set the environment variable.
- **Fix**: Remove the fallback and fail fast:
  ```rust
  let db_url = std::env::var("DATABASE_URL")
      .context("DATABASE_URL must be set")?;
  ```

### [C-004] Hardcoded AWS/MinIO Credentials in Fallback
- **File**: `services/processor/src/config.rs`
- **Line**: 26–29
- **Code**:
  ```rust
  let access_key = std::env::var("AWS_ACCESS_KEY_ID")
      .unwrap_or_else(|_| "chronoscope".to_string());
  let secret_key = std::env::var("AWS_SECRET_ACCESS_KEY")
      .unwrap_or_else(|_| "chronoscope123".to_string());
  ```
- **Issue**: Default AWS credentials are embedded in source code.
- **Impact**: Cloud storage compromise if deployed without explicit environment configuration.
- **Fix**: Require both variables or return an error:
  ```rust
  let access_key = std::env::var("AWS_ACCESS_KEY_ID")
      .context("AWS_ACCESS_KEY_ID must be set")?;
  let secret_key = std::env::var("AWS_SECRET_ACCESS_KEY")
      .context("AWS_SECRET_ACCESS_KEY must be set")?;
  ```

### [C-005] Intentional Temporary Directory Resource Leak
- **File**: `services/processor/src/downloader.rs`
- **Line**: 37–39
- **Code**:
  ```rust
  // Keep temp_dir alive by leaking it (simpler for this pipeline)
  // In production you'd manage the TempDir lifecycle more carefully
  let _ = Box::leak(Box::new(temp_dir));
  ```
- **Issue**: The `TempDir` is deliberately leaked, leaving all downloaded S3 objects on disk forever. The comment explicitly acknowledges this is not production-safe.
- **Impact**: Disk space exhaustion (DoS) over time. Sensitive session chunks may persist in `/tmp` and be exposed to other local users.
- **Fix**: Return the `TempDir` alongside `paths` (e.g., wrap both in a struct that owns the directory) or use an RAII guard that cleans up after the pipeline finishes.

### [C-006] Path Traversal in Video Encoder Output Path
- **File**: `services/processor/src/encoder.rs`
- **Line**: 20
- **Code**:
  ```rust
  let output_path = std::env::temp_dir().join(format!("{}.mp4", session_id));
  ```
- **Issue**: `session_id` is taken directly from the Redis queue and interpolated into a filesystem path without sanitization. A malicious session ID such as `../../../etc/cron.d/evil` causes the encoder to write the MP4 file (or create/truncate the target) outside the temporary directory.
- **Impact**: Arbitrary file write on the processor host, potentially leading to remote code execution (e.g., dropping a cron job or overwriting a system binary).
- **Fix**: Validate that `session_id` matches a strict allowlist (e.g., `^[a-zA-Z0-9_-]+$`) or map it to a safe UUID-derived filename:
  ```rust
  let safe_id = session_id.replace(['/', '\\', '\0'], "_");
  let output_path = std::env::temp_dir().join(format!("{}.mp4", safe_id));
  ```
  A stronger fix is to hash the session_id and use the hash as the filename.

---

## HIGH Findings

### [H-001] Panic on Variable Frame Dimensions in Encoder
- **File**: `services/processor/src/encoder.rs`
- **Line**: 69–70
- **Code**:
  ```rust
  let raw = img.into_raw();
  let mut rgb_frame = frame::Video::new(ffmpeg::format::Pixel::RGB24, width as u32, height as u32);
  rgb_frame.data_mut(0).copy_from_slice(&raw);
  ```
- **Issue**: `width` and `height` are derived from the **first** frame only. If any subsequent frame in `frames` has different dimensions, `img.into_raw()` produces a buffer of a different length than `rgb_frame.data_mut(0)`, causing `copy_from_slice` to panic.
- **Impact**: Denial of service (panic) of the processing pipeline.
- **Fix**: Resize or skip mismatched frames gracefully:
  ```rust
  let raw = img.into_raw();
  if raw.len() != rgb_frame.data(0).len() {
      tracing::warn!("Skipping frame with mismatched dimensions: {}", path.display());
      continue;
  }
  rgb_frame.data_mut(0).copy_from_slice(&raw);
  ```

### [H-002] Overlapping Detections Can Panic in Text Redaction
- **File**: `services/privacy-engine/src/lib.rs`
- **Line**: 71–77
- **Code**:
  ```rust
  for detection in detections.iter().rev() {
      let replacement = match &self.config.redaction_mode { ... };
      result.replace_range(detection.start..detection.end, &replacement);
  }
  ```
- **Issue**: `detections` may overlap when multiple regexes match intersecting ranges (e.g., a password regex spanning an email). Replacing the inner range first shifts the string length, invalidating the outer range indices. `String::replace_range` panics on out-of-bounds ranges.
- **Impact**: Denial of service via panic on crafted input such as `"password: user@example.com"`.
- **Fix**: Merge or clip overlapping detections before replacement:
  ```rust
  // After sorting, merge overlapping ranges
  let mut merged: Vec<Detection> = Vec::new();
  for d in detections {
      match merged.last_mut() {
          Some(last) if d.start < last.end => last.end = last.end.max(d.end),
          _ => merged.push(d),
      }
  }
  ```

### [H-003] ReDoS via Unvalidated Custom Regex Patterns
- **File**: `services/privacy-engine/src/detector.rs`
- **Line**: 78–89
- **Code**:
  ```rust
  for pattern in &config.custom_patterns {
      if let Ok(re) = Regex::new(pattern) {
          for mat in re.find_iter(text) { ... }
      }
  }
  ```
- **Issue**: User-supplied `custom_patterns` are compiled directly into regexes without length limits, complexity checks, or execution timeouts. A malicious pattern like `(a+)+$` against a long input causes catastrophic backtracking.
- **Impact**: CPU exhaustion (ReDoS) in the privacy engine.
- **Fix**: Use `regex::RegexBuilder` with a size limit and consider running regex matching in a separate thread with a timeout:
  ```rust
  let re = regex::RegexBuilder::new(pattern)
      .size_limit(1 << 20)
      .build()?;
  ```

### [H-004] Unsafe Blocks Lack Detailed Safety Justifications
- **File**: `services/privacy-engine/src/ffi.rs`
- **Line**: 10, 26, 28, 42, 43, 52, 60
- **Issue**: Every `unsafe` block in the FFI boundary has only minimal or no `// SAFETY:` comments. For example, line 9 says "caller must pass valid null-terminated string" but does not explain why the code is sound even with that precondition, nor what happens when it is violated.
- **Impact**: Maintenance risk; future changes may violate unwritten invariants, leading to undefined behavior.
- **Fix**: Add explicit `// SAFETY:` annotations before each block documenting (1) the invariant being relied upon, (2) how the caller upholds it, and (3) why the operation is sound.

### [H-005] Unbounded Dimensions in Frame Processing FFI
- **File**: `services/privacy-engine/src/ffi.rs`
- **Line**: 18–30
- **Issue**: `width`, `height`, and `stride` are passed from C without any upper-bound validation. Extremely large values (e.g., `u32::MAX`) can cause memory exhaustion when computing `height * stride`.
- **Impact**: Denial of service via out-of-memory.
- **Fix**: Validate dimensions against sensible limits before use:
  ```rust
  const MAX_DIMENSION: u32 = 16384;
  if width == 0 || height == 0 || stride == 0 || width > MAX_DIMENSION || height > MAX_DIMENSION {
      return;
  }
  ```

### [H-006] Silent Failover to Default Privacy Config on Invalid JSON
- **File**: `services/privacy-engine/src/ffi.rs`
- **Line**: 11
- **Code**:
  ```rust
  let config: PrivacyConfig = serde_json::from_str(&config_str).unwrap_or_default();
  ```
- **Issue**: If the caller provides malformed JSON, the engine silently uses `PrivacyConfig::default()`, which enables **all** detections. A caller attempting to disable detection (e.g., `detect_emails: false`) but making a JSON typo would have all privacy features enabled—exactly the opposite of their intent.
- **Impact**: Privacy bypass / unintended data exposure due to misconfiguration.
- **Fix**: Return `null_mut()` on parse failure rather than defaulting:
  ```rust
  let config: PrivacyConfig = match serde_json::from_str(&config_str) {
      Ok(c) => c,
      Err(_) => return std::ptr::null_mut(),
  };
  ```

### [H-007] In-Place Blur Corrupts Its Own Source Data
- **File**: `services/privacy-engine/src/redaction.rs`
- **Line**: 21–52
- **Issue**: The blur kernel reads neighboring pixels from `frame` while simultaneously writing blurred results back into the same buffer. Pixels that have already been blurred are used as source data for subsequent pixels, producing an incorrect (and weaker) blur.
- **Impact**: Privacy degradation—sensitive text or faces may remain legible because the blur is mathematically incorrect.
- **Fix**: Clone the source region before blurring, or use a separate destination buffer for the blur pass.

### [H-008] No Graceful Shutdown or Signal Handling in Processor
- **File**: `services/processor/src/main.rs`
- **Line**: 13–32
- **Issue**: `main` runs an infinite `while let Some` loop with no `tokio::select!` for `ctrl_c` or SIGTERM. If the pod/process is terminated, in-flight session processing is aborted mid-way without cleanup, potentially leaving leaked temp files or inconsistent DB states.
- **Impact**: Resource leaks and data inconsistency during deployments or scaling events.
- **Fix**: Listen for shutdown signals and propagate cancellation:
  ```rust
  let (tx, mut rx) = tokio::sync::mpsc::channel::<String>(100);
  tokio::spawn(queue::queue_listener(config.clone(), tx));
  
  let mut shutdown = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())?;
  loop {
      tokio::select! {
          Some(session_id) = rx.recv() => { ... }
          _ = tokio::signal::ctrl_c() => break,
          _ = shutdown.recv() => break,
      }
  }
  ```

---

## MEDIUM Findings

### [M-001] Database Field Errors Silently Defaulted
- **File**: `services/processor/src/sync.rs`
- **Line**: 25–28
- **Code**:
  ```rust
  event_type: row.try_get(0).unwrap_or_default(),
  timestamp_ms: row.try_get::<_, i32>(1).unwrap_or(0) as u64,
  x: row.try_get(2).unwrap_or(0),
  y: row.try_get(3).unwrap_or(0),
  ```
- **Issue**: Type mismatches or unexpected nulls in the `events` table are silently converted to default values, masking data corruption.
- **Impact**: Silent loss of event data integrity.
- **Fix**: Propagate errors:
  ```rust
  event_type: row.try_get(0)?,
  timestamp_ms: row.try_get::<_, i64>(1)? as u64,
  ```

### [M-002] Redis Listener Dies on First Connection Error
- **File**: `services/processor/src/queue.rs`
- **Line**: 9–23
- **Issue**: The `loop` returns `Err` immediately if `query_async` fails. The spawned task then terminates with no restart logic.
- **Impact**: Complete queue consumption halt after a transient Redis blip.
- **Fix**: Log the error, add a backoff delay, and `continue` the loop instead of returning.

### [M-003] X11 Capture Task Runs Forever Without Cancellation
- **File**: `packages/sdk-linux/src/capture/x11.rs`
- **Line**: 42–66
- **Issue**: The `loop` has no `break` condition or cancellation token. The TODO at line 63 admits this is missing.
- **Impact**: Once started, the capture task cannot be stopped, leaking a tokio task and X11 connection.
- **Fix**: Accept a `tokio_util::sync::CancellationToken` and `break` on cancellation.

### [M-004] Missing UUID Validation Before Database Cast
- **File**: `services/processor/src/db.rs`
- **Line**: 18
- **Code**:
  ```rust
  &[&status, &video_path, &metadata, &session_id],
  ```
- **Issue**: `session_id` is passed to PostgreSQL with a `::uuid` cast without client-side validation.
- **Impact**: Unnecessary database errors and potential for error-based information probing.
- **Fix**: Validate with `uuid::Uuid::parse_str(session_id)?` before the query.

### [M-005] Redaction Function Ignores Height Parameter
- **File**: `services/privacy-engine/src/redaction.rs`
- **Line**: 4
- **Issue**: `_height` is unused. The redaction logic assumes `detection.start..detection.end` is a valid linear range but never checks it against the image height.
- **Impact**: Potential incorrect redaction if a detector supplies out-of-bounds ranges.
- **Fix**: Use `_height` for bounds checking or remove the parameter.

### [M-006] Missing Documentation on Public APIs
- **File**: Multiple
- **Line**: Various
- **Issue**: Many public structs and functions lack `///` doc comments, including:
  - `packages/sdk-linux/src/buffer.rs`: `CircularBuffer`
  - `packages/sdk-linux/src/upload.rs`: `ChunkUploader`
  - `services/privacy-engine/src/lib.rs`: `PrivacyEngine::process_frame`, `process_text`
- **Impact**: Developers may misuse APIs.
- **Fix**: Add rustdoc comments to all `pub` items.

### [M-007] Hardcoded S3/Redis Endpoint Fallbacks
- **File**: `services/processor/src/config.rs`
- **Line**: 24, 49, 55–58
- **Issue**: Fallback values exist for `AWS_ENDPOINT_URL`, `REDIS_URL`, and bucket names.
- **Impact**: Risk of connecting to wrong infrastructure in production if env vars are missing.
- **Fix**: Remove fallbacks for infrastructure endpoints and fail fast.

### [M-008] Regexes Recompiled on Every Text Scan
- **File**: `services/privacy-engine/src/detector.rs`
- **Line**: 30, 42, 54, 67
- **Issue**: Hardcoded regex patterns are compiled inside `scan_text` on every call.
- **Impact**: Unnecessary CPU overhead.
- **Fix**: Use `lazy_static!` or `once_cell::sync::Lazy`:
  ```rust
  static EMAIL_RE: Lazy<Regex> = Lazy::new(|| {
      Regex::new(r"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}").unwrap()
  });
  ```

### [M-009] Consent Mechanism Is a Non-Functional Stub
- **File**: `services/privacy-engine/src/consent.rs`
- **Line**: 3–5
- **Issue**: `get_status` always returns `ConsentStatus::Granted` regardless of user.
- **Impact**: Privacy consent framework is bypassed entirely.
- **Fix**: Implement actual consent storage and lookup.

### [M-010] Processor Has Zero Unit/Integration Tests
- **File**: `services/processor/`
- **Line**: N/A
- **Issue**: The processor crate has no `tests/` directory and no `#[cfg(test)]` modules.
- **Impact**: No automated verification of pipeline correctness, regressions likely.
- **Fix**: Add tests for downloader, encoder, deduplicator, and DB operations using mock S3/PostgreSQL.

---

## LOW Findings

### [L-001] Large Commented-Out Code Blocks in X11 Capture
- **File**: `packages/sdk-linux/src/capture/x11.rs`
- **Line**: 46–61
- **Issue**: Extensive commented-out implementation code clutters the file.
- **Fix**: Remove or extract to a design document.

### [L-002] Large Commented-Out Code Blocks in Wayland Capture
- **File**: `packages/sdk-linux/src/capture/wayland.rs`
- **Line**: 24–50
- **Issue**: Same as L-001.
- **Fix**: Remove or extract to a design document.

### [L-003] Unreachable Code After Infinite Loop
- **File**: `packages/sdk-linux/src/capture/x11.rs`
- **Line**: 68
- **Issue**: `Ok(())` after a `loop {}` that never breaks.
- **Fix**: Remove or implement cancellation to make the `Ok(())` reachable.

### [L-004] Default API Key is Empty String
- **File**: `packages/sdk-linux/src/config.rs`
- **Line**: 28
- **Issue**: `api_key: String::new()` in `Default` means uploads will fail if the caller relies on defaults.
- **Fix**: Remove `Default` or make `api_key` a required constructor parameter with no default.

### [L-005] Missing FFI Error for Null-Byte Text Input
- **File**: `services/privacy-engine/src/ffi.rs`
- **Line**: 45
- **Code**:
  ```rust
  CString::new(result).map(|s| s.into_raw()).unwrap_or(std::ptr::null_mut())
  ```
- **Issue**: If `result` contains an interior null byte, the function returns `null_mut()`. The C caller cannot distinguish this from a null-input error.
- **Fix**: Document the behavior or sanitize/replace null bytes before creating `CString`.

### [L-006] Redundant Integration Tests for CircularBuffer
- **File**: `packages/sdk-linux/tests/integration_tests.rs`
- **Line**: 45–64
- **Issue**: The integration tests duplicate the unit tests in `src/buffer.rs`.
- **Fix**: Focus integration tests on higher-level workflows (e.g., `LinuxCapture` lifecycle) rather than re-testing `CircularBuffer` internals.

---

## Checklist Results

### Security
- [x] **Unsafe blocks without comment justification** — ❌ **FAIL** (see H-004)
- [x] **FFI boundary: null pointer checks on all C ABI functions** — ❌ **FAIL** (see C-001)
- [x] **`Box::into_raw` has matching `Box::from_raw` / free function** — ✅ **PASS** (`chronoscope_privacy_free` correctly mirrors `chronoscope_privacy_init`)
- [x] **No `unwrap()` or `expect()` in production paths** — ⚠️ **PARTIAL FAIL**
  - `downloader.rs:16` `unwrap_or_default()` — safe
  - `downloader.rs:31` `unwrap_or(&key)` — safe
  - `sync.rs:25-28` `unwrap_or_default()` / `unwrap_or(0)` — silently masks errors (M-001)
  - `privacy-engine/src/ffi.rs:11` `unwrap_or_default()` on JSON parse — dangerous (H-006)
  - `privacy-engine/src/detector.rs:30,42,54,67` `unwrap()` on hardcoded regex compile — acceptable because patterns are static and known-good
- [x] **Resource cleanup on panic (RAII pattern)** — ⚠️ **PARTIAL FAIL**
  - `downloader.rs` intentionally leaks `TempDir` (C-005)
  - `encoder.rs` writes to temp dir but file is not explicitly cleaned up after upload
  - Otherwise `spawn_blocking` tasks and DB connections use RAII correctly
- [x] **Regex patterns are bounded (no catastrophic backtracking)** — ⚠️ **PARTIAL FAIL**
  - Hardcoded patterns are bounded ✅
  - `config.custom_patterns` is unvalidated (H-003) ❌
- [x] **File paths sanitized (directory traversal prevention)** — ❌ **FAIL**
  - `encoder.rs` uses raw `session_id` in filename (C-006)
  - `downloader.rs` extracts basename via `rsplit('/')`, which is safe ✅
- [x] **No hardcoded secrets** — ❌ **FAIL** (see C-003, C-004)
- [x] **Input validation on all public functions** — ❌ **FAIL**
  - `ffi.rs` lacks dimension validation (H-005)
  - `encoder.rs` lacks session_id validation (C-006)
  - `db.rs` lacks UUID validation (M-004)

### Quality
- [x] **Proper error propagation (Result/Option)** — ⚠️ **PARTIAL FAIL**
  - Most functions return `anyhow::Result` ✅
  - `sync.rs` defaults instead of propagating DB type errors (M-001) ❌
- [x] **No dead code** — ⚠️ **PARTIAL FAIL**
  - `redaction.rs:4` `_height` is unused (M-005)
  - `input.rs`, `consent.rs`, `wayland.rs`, `x11.rs` are largely stubs / unimplemented
  - Large commented-out blocks in capture modules (L-001, L-002)
- [x] **Consistent async/await usage** — ✅ **PASS**
  - CPU-bound work correctly offloaded via `tokio::task::spawn_blocking`
  - I/O operations are properly `await`ed
- [x] **Proper resource cleanup** — ⚠️ **PARTIAL FAIL**
  - `downloader.rs` leaks `TempDir` (C-005)
  - `x11.rs` task never terminates (M-003)
  - No explicit cleanup of encoder output file after upload
- [x] **Thread safety where needed** — ✅ **PASS**
  - `CircularBuffer` is wrapped in `Arc<Mutex<>>` where shared
  - `PrivacyEngine` is `Send` (contains `PrivacyConfig` and `AuditLogger`, both `Send`)
- [x] **Documentation completeness** — ❌ **FAIL** (see M-006)
  - Many `pub` items lack doc comments
  - `unsafe` blocks lack `SAFETY` annotations (H-004)
