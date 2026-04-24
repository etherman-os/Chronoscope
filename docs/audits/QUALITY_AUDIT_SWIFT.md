# Swift Quality Audit Report

**Scope**: `packages/sdk-macos/` — All Swift source and test files  
**Auditor**: Security & Quality Auditor (Swift)  
**Date**: 2026-04-25  
**Total Files Audited**: 10

---

## Force Unwraps

| File | Line(s) | Code | Severity | Recommended Fix |
|------|---------|------|----------|-----------------|
| `PrivacyEngine.swift` | 8–9 | `try! JSONEncoder().encode(config)`<br>`String(data: jsonData, encoding: .utf8)!` | CRITICAL | Use `try?` / `guard let` and propagate failure |
| `ChunkUploader.swift` | 56–61 | `.data(using: .utf8)!` (×5) | MEDIUM | Use `guard let` or non-optional helper |
| `ChronoscopeTests.swift` | 6, 7, 19, 29, 30 | `URL(string: ...)!`, `.data(using: .utf8)!` | N/A (Tests) | Acceptable in unit tests |

**Summary**: Two production files contain force unwraps. The `PrivacyEngine` ones are especially dangerous because they sit in an initializer and will crash the host app on any encoding anomaly.

---

## Memory Management

### Retain Cycle Analysis
- **CaptureSession → FrameCapture**: `CaptureSession` holds `frameCapture` strongly. The frame handler closure uses `[weak self]` (line 36), and the inner `Task` also uses `[weak self]` (line 37). **No cycle**.
- **FrameCapture → frameHandler**: `FrameCapture` holds the closure strongly, but the closure only captures `self` weakly. **No cycle**.
- **CaptureSession → uploadTask**: The `Task` uses `[weak self]`. **No cycle**.
- **ChunkUploader**: No closures that capture `self`.

### Resource Leaks
- **CIContext Leak (FrameCapture.swift:65–66)**: A new `CIContext` is created on every frame. This is not a leak in the ARC sense, but it creates severe memory pressure. The context should be cached.
- **SCStream Deinit Gap (FrameCapture.swift)**: If `FrameCapture` is deallocated without calling `stop()`, the `SCStream` may not be stopped. Add a `deinit` assertion or synchronous cleanup.

### Actor Isolation & Memory Safety
- `CircularBuffer` is an `actor` but also contains an `NSLock`. This is redundant because actor isolation already serializes access. The lock should be removed to avoid double-synchronization overhead.

---

## Error Handling

### Incomplete Error Propagation
| File | Line | Issue |
|------|------|-------|
| `CaptureSession.swift` | 46 | `try?` suppresses `Task.sleep` cancellation. Use bare `try`. |
| `FrameCapture.swift` | 27, 43 | Errors are logged to `print()` and swallowed. Callers have no way to detect capture failure. |
| `CaptureSession.swift` | 86 | Upload errors are printed but not propagated or retried. Silent data loss. |

### Error Type Design
- **ChronoscopeError.swift**: The enum is minimal and `Sendable`, which is good. However, associated values are plain `String`s. Consider using structured error payloads (e.g., `HTTPURLResponse?`, `URLError?`) to aid diagnostics.

### Missing Error Paths in Tests
- No tests verify that `sessionInitFailed` or `uploadFailed` are thrown under the correct conditions.
- No tests verify behavior when `SCShareableContent.current` throws.

---

## Concurrency Safety

### Actor Isolation Correctness
| Component | Isolation | Assessment |
|-----------|-----------|------------|
| `CaptureSession` | `actor` | Correct. All mutable state is protected. |
| `FrameCapture` | `actor` + `NSObject` | Correct, but `SCStreamOutput` callback is `nonisolated` and correctly uses `await` to call actor-isolated `frameHandler`. |
| `CircularBuffer` | `actor` + `NSLock` | **Over-isolated**. Remove `NSLock`. |
| `ChunkUploader` | `actor` | Correct. No mutable state beyond constants. |
| `PrivacyEngine` | `actor` | Correct. Protects the opaque C pointer. |
| `Chronoscope` | `class` (non-actor) | **Race condition**. `session` is unprotected across suspension points. See Security Audit [H-002]. |

### `Task` Lifecycle
- **CaptureSession.swift:44–50**: The upload loop `Task` is created with `[weak self]`, which is good. It checks `Task.isCancelled` before sleeping.
- **CaptureSession.swift:36–40**: The frame handler spawns a new `Task` for every frame. Under high frame rates this can create a large number of pending tasks. Consider using a single `AsyncStream` or serial task to backpressure the buffer.

---

## API Design

### CaptureConfig.swift
- **Missing validation**: `frameRate`, `bufferSizeMB`, and `quality` are accepted without range checks. A `frameRate` of `0` or `-1` will cause division-by-zero or infinite frame intervals in `FrameCapture`.
- **Missing documentation**: No DocC comments on any public member.
- **Missing `userId`**: The session initialization hardcodes `"macos_user"` because `CaptureConfig` does not expose a user identifier field.

### Chronoscope.swift
- **Singleton pattern**: `shared` is convenient but limits testability. Consider allowing dependency injection of the `CaptureSession` for unit testing.
- **Missing `isRunning` / `isCapturing`**: Callers cannot query the current state.

### FrameCapture.swift
- **No error callback**: `start(handler:)` only takes a success handler. There is no way for consumers to learn that capture failed.
- **Delegate nil**: `SCStream` errors are ignored because `delegate` is `nil`.

### ChunkUploader.swift
- **Multipart construction**: `createMultipartBody` is private and untested. It should accept `filename` and `mimeType` as parameters rather than hardcoding them, to support future formats (e.g., HEVC, ProRes).

---

## Test Coverage Gaps

### Current Tests (ChronoscopeTests.swift)
1. `testCaptureConfigDefaults` — verifies struct defaults.
2. `testCircularBuffer` — single write + read.
3. `testCircularBufferWrapAround` — write, read, write, read.

### Missing Coverage (High Priority)
| Component | Missing Scenarios |
|-----------|-------------------|
| `CaptureSession` | Start/stop lifecycle, double-start, double-stop, error during `initializeSession`, upload loop behavior. |
| `FrameCapture` | Start/stop, stream error handling, pixel buffer conversion, privacy integration. |
| `ChunkUploader` | `uploadChunk` success/failure, `finalize` success/failure, multipart body correctness, URL path construction. |
| `PrivacyEngine` | `processFrame` with valid/invalid data, `processText` redaction, C-string memory safety, engine init failure. |
| `CircularBuffer` | Zero/negative capacity, overwrite behavior, empty read, concurrent writes (actor tests), large data > capacity. |
| `Chronoscope` | Singleton start/stop, concurrent start calls, error propagation to caller. |

### Test Infrastructure Gaps
- No mock `URLProtocol` for network testing.
- No dependency injection harness for `SCStream`.
- No performance/benchmark tests for `CircularBuffer` or frame encoding.

---

## Dead Code

- `CaptureConfig.swift`: `CaptureQuality` enum (`.low`, `.medium`, `.high`) is defined but **never referenced** in the capture or encoding pipeline. `FrameCapture` hardcodes JPEG compression factor `0.7` regardless of `CaptureQuality`.
- `ChronoscopeError.captureFailed(String)`: Defined but never instantiated in the current codebase.

---

## Documentation Completeness

| File | Public API Count | Documented | Coverage |
|------|------------------|------------|----------|
| `Chronoscope.swift` | 3 | 0 | 0% |
| `CaptureConfig.swift` | 2 + 2 enums | 0 | 0% |
| `CaptureSession.swift` | 2 | 0 | 0% |
| `FrameCapture.swift` | 3 | 0 | 0% |
| `ChunkUploader.swift` | 3 | 0 | 0% |
| `PrivacyEngine.swift` | 3 + 1 struct | 0 | 0% |
| `CircularBuffer.swift` | 3 | 0 | 0% |
| `ChronoscopeError.swift` | 1 enum | 0 | 0% |

**Recommendation**: Add DocC (`///`) comments to every public declaration, including parameter descriptions and thrown errors.

---

## Summary of Recommendations (Priority Order)

1. **Apply privacy filtering before JPEG encoding** (Security C-001).
2. **Remove all force unwraps from production code** (Security C-002, M-001).
3. **Validate `CircularBuffer` capacity** and all `CaptureConfig` inputs (Security C-003, M-005).
4. **Make `Chronoscope` an actor** to eliminate the session race condition (Security H-002).
5. **Configure explicit URLSession timeouts** (Security H-001).
6. **Cache `CIContext`** and add `SCStreamDelegate` (Security H-003, Quality M-002).
7. **Remove redundant `NSLock`** from `CircularBuffer` (Quality M-003).
8. **Expand unit tests** to cover error paths, network mocks, and buffer edge cases (Quality M-008).
9. **Replace `print()` with `OSLog.Logger`** (Quality L-001).
10. **Add DocC documentation** to all public APIs (Quality L-002).
