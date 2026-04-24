# Swift Security Audit Report

**Scope**: `packages/sdk-macos/` — All Swift source and test files  
**Auditor**: Security & Quality Auditor (Swift)  
**Date**: 2026-04-25  
**Total Files Audited**: 10

---

## Executive Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 3 |
| HIGH     | 3 |
| MEDIUM   | 8 |
| LOW      | 4 |

---

## CRITICAL Findings

### [C-001] Privacy Filtering Bypassed — Raw Frames Uploaded Without PII Masking
- **File**: `packages/sdk-macos/Sources/Chronoscope/Capture/FrameCapture.swift`
- **Lines**: 76–79, 81–83
- **Issue**: The `SCStreamOutput` callback encodes frames to JPEG and immediately dispatches them to the upload pipeline. A `TODO` comment explicitly acknowledges that `PrivacyEngine.processFrame` is **not** invoked because the pipeline currently produces JPEG instead of raw RGBA. As a result, screen captures containing passwords, credit card numbers, emails, and other sensitive PII are uploaded to the server **unredacted**.
- **Impact**: Data breach / privacy violation. End-user credentials, financial data, and personal information are transmitted over the network without any client-side redaction.
- **Fix**: Refactor the frame pipeline to decode the `CVPixelBuffer` to raw RGBA, pass it through `PrivacyEngine.processFrame(_:width:height:stride:)`, then re-encode to JPEG before dispatching.

```swift
// Example fix (conceptual)
nonisolated public func stream(...) {
    guard outputType == .screen else { return }
    guard let pixelBuffer = sampleBuffer.imageBuffer else { return }
    
    // 1. Lock base address and extract raw RGBA
    CVPixelBufferLockBaseAddress(pixelBuffer, .readOnly)
    defer { CVPixelBufferUnlockBaseAddress(pixelBuffer, .readOnly) }
    
    guard let baseAddress = CVPixelBufferGetBaseAddress(pixelBuffer) else { return }
    let width = UInt32(CVPixelBufferGetWidth(pixelBuffer))
    let height = UInt32(CVPixelBufferGetHeight(pixelBuffer))
    let stride = UInt32(CVPixelBufferGetBytesPerRow(pixelBuffer))
    let frameSize = Int(height) * Int(stride)
    
    var frameData = Data(bytes: baseAddress, count: frameSize)
    
    // 2. Apply privacy filtering BEFORE encoding
    Task {
        await privacyEngine?.processFrame(&frameData, width: width, height: height, stride: stride)
        // 3. Encode filtered frame to JPEG and forward
        if let jpegData = encodeToJPEG(frameData, width: width, height: height) {
            await self.frameHandler?(jpegData)
        }
    }
}
```

---

### [C-002] Force-Unwrap Crash in PrivacyEngine Initializer
- **File**: `packages/sdk-macos/Sources/Chronoscope/Privacy/PrivacyEngine.swift`
- **Lines**: 8–9
- **Code**:
```swift
let jsonData = try! JSONEncoder().encode(config)
let jsonString = String(data: jsonData, encoding: .utf8)!
```
- **Issue**: Production code uses `try!` and a force unwrap (`!`). While `JSONEncoder` encoding a local `Codable` struct is unlikely to throw, any future change to `PrivacyConfig` (e.g., a non-Codable property) will cause an **unrecoverable runtime crash**.
- **Impact**: Denial of service. The entire capture session initialization crashes the host application.
- **Fix**:
```swift
guard let jsonData = try? JSONEncoder().encode(config),
      let jsonString = String(data: jsonData, encoding: .utf8) else {
    // Log error and fail gracefully
    return
}
```

---

### [C-003] CircularBuffer Crash on Zero or Negative Capacity
- **File**: `packages/sdk-macos/Sources/Chronoscope/Buffer/CircularBuffer.swift`
- **Lines**: 12–13, 26
- **Issue**: `CircularBuffer.init(capacity:)` accepts any `Int` without validation. If `capacity <= 0`:
  - `Data(count: capacity)` crashes on negative values.
  - `writeOffset = (writeOffset + 1) % capacity` crashes with division-by-zero when `capacity == 0`.
- **Impact**: Denial of service. A malformed or default-derived `CaptureConfig` can crash the app during session start.
- **Fix**:
```swift
public init(capacity: Int) {
    precondition(capacity > 0, "CircularBuffer capacity must be > 0")
    self.capacity = capacity
    self.storage = Data(count: capacity)
}
```
Additionally validate upstream in `CaptureConfig` or `CaptureSession.start()`.

---

## HIGH Findings

### [H-001] No Explicit Network Timeout Configured on URLSession
- **File**: `packages/sdk-macos/Sources/Chronoscope/Core/CaptureSession.swift` (line 111)  
  `packages/sdk-macos/Sources/Chronoscope/Network/ChunkUploader.swift` (lines 25, 41)
- **Issue**: All network requests use `URLSession.shared`, which relies on a 60-second default timeout. For large video chunk uploads or slow connections, this is insufficient and can lead to silent stalls or ambiguous cancellation. No custom `URLSessionConfiguration` with explicit `timeoutIntervalForRequest`, `timeoutIntervalForResource`, or retry logic is used.
- **Impact**: Uploads may hang indefinitely (or until the OS kills the task), causing memory bloat from buffered frames and potential data loss.
- **Fix**: Inject a configured `URLSession` instance:
```swift
let config = URLSessionConfiguration.default
timeoutIntervalForRequest = 30
timeoutIntervalForResource = 300
let session = URLSession(configuration: config)
```

---

### [H-002] Race Condition in Chronoscope Singleton Session Management
- **File**: `packages/sdk-macos/Sources/Chronoscope/Core/Chronoscope.swift`
- **Lines**: 3–20
- **Issue**: `Chronoscope` is a non-actor `class` with a mutable `session` property. `start(config:)` checks `session == nil`, then performs an `await` on `newSession.start()`, and only then assigns `session`. Two concurrent calls from different `Task`s can both pass the `guard`, creating two `CaptureSession` instances. The second assignment overwrites the first, leaking the underlying `SCStream` and upload tasks.
- **Impact**: Resource leak, multiple concurrent capture sessions, and undefined behavior.
- **Fix**: Make `Chronoscope` an actor, or protect `session` with a dedicated `NSLock`/`os_unfair_lock` and perform the check + assignment atomically.
```swift
public actor Chronoscope {
    public static let shared = Chronoscope()
    private var session: CaptureSession?
    private init() {}
    
    public func start(config: CaptureConfig) async throws {
        guard session == nil else { return }
        let newSession = CaptureSession(config: config)
        try await newSession.start()
        session = newSession
    }
    // ...
}
```

---

### [H-003] SCStream Started Without Error Delegate
- **File**: `packages/sdk-macos/Sources/Chronoscope/Capture/FrameCapture.swift`
- **Line**: 38
- **Code**:
```swift
let newStream = SCStream(filter: filter, configuration: configuration, delegate: nil)
```
- **Issue**: The `SCStream` delegate is `nil`. Per ScreenCaptureKit documentation, stream errors, permission revocations, and state changes are reported via the delegate. Without a delegate, capture failures are completely silent.
- **Impact**: Users and developers have no visibility when screen recording permission is revoked or the stream terminates unexpectedly. The SDK continues running in a broken state.
- **Fix**: Conform `FrameCapture` to `SCStreamDelegate` and set `delegate: self`:
```swift
extension FrameCapture: SCStreamDelegate {
    nonisolated public func stream(_ stream: SCStream, didStopWithError error: Error) {
        // Log error, notify caller, clean up
    }
}
```

---

## MEDIUM Findings

### [M-001] Force Unwraps in Multipart Body Construction
- **File**: `packages/sdk-macos/Sources/Chronoscope/Network/ChunkUploader.swift`
- **Lines**: 56–61
- **Code**:
```swift
body.append("--\(boundary)\r\n".data(using: .utf8)!)
body.append("Content-Disposition: form-data; name=\"chunk\"; filename=\"\(filename)\"\r\n".data(using: .utf8)!)
// ... etc
```
- **Issue**: Five force unwraps (`!`) on `.data(using: .utf8)`. While UTF-8 encoding of ASCII strings is technically safe, this pattern is brittle to future refactoring (e.g., injecting user-controlled filenames).
- **Impact**: Potential crash if injected strings contain invalid UTF-8 sequences.
- **Fix**: Use `guard let` or a helper that returns `Data` non-optionally:
```swift
private func utf8Data(_ string: String) -> Data {
    string.data(using: .utf8) ?? Data(string.utf8)
}
```

---

### [M-002] CIContext Instantiated Per Frame Causing Memory Pressure
- **File**: `packages/sdk-macos/Sources/Chronoscope/Capture/FrameCapture.swift`
- **Lines**: 65–66
- **Code**:
```swift
let ciImage = CIImage(cvPixelBuffer: pixelBuffer)
let context = CIContext()
```
- **Issue**: A new `CIContext` is created for every single frame. `CIContext` is expensive to initialize and can consume significant GPU/CPU resources.
- **Impact**: High CPU/GPU usage, frame drops, and increased memory pressure during screen capture.
- **Fix**: Cache a single `CIContext` as a lazy property on the actor:
```swift
private lazy var ciContext = CIContext()
```

---

### [M-003] Redundant NSLock Inside Actor
- **File**: `packages/sdk-macos/Sources/Chronoscope/Buffer/CircularBuffer.swift`
- **Line**: 9
- **Code**:
```swift
private let lock = NSLock()
```
- **Issue**: `CircularBuffer` is declared as `public actor CircularBuffer`. Swift actors already guarantee mutual exclusion for all actor-isolated methods. The explicit `NSLock` in `write(_:)` and `readChunk()` is redundant and adds unnecessary overhead.
- **Impact**: Slight performance overhead and code confusion.
- **Fix**: Remove `NSLock` and rely on actor isolation, or convert `CircularBuffer` to a non-actor class if low-level locking is preferred.

---

### [M-004] Session ID Not Explicitly URL-Encoded in Path
- **File**: `packages/sdk-macos/Sources/Chronoscope/Network/ChunkUploader.swift`
- **Line**: 15
- **Code**:
```swift
let url = endpoint.appendingPathComponent("sessions/\(sessionId)/chunks")
```
- **Issue**: The entire string `"sessions/\(sessionId)/chunks"` is passed as a single path component. While `appendingPathComponent` performs some percent-encoding, nested path separators or special characters in `sessionId` may produce unexpected URL structures or path traversal behavior depending on server decoding.
- **Impact**: Potential 404s, server-side path traversal, or session leakage if `sessionId` is malformed.
- **Fix**: Append each path segment individually:
```swift
let url = endpoint
    .appendingPathComponent("sessions")
    .appendingPathComponent(sessionId)
    .appendingPathComponent("chunks")
```

---

### [M-005] Integer Overflow in Buffer Capacity Calculation
- **File**: `packages/sdk-macos/Sources/Chronoscope/Core/CaptureSession.swift`
- **Line**: 24
- **Code**:
```swift
let bufferCapacity = config.bufferSizeMB * 1_024 * 1_024
```
- **Issue**: No bounds checking on `config.bufferSizeMB`. On 64-bit platforms the overflow threshold is high, but an extremely large value (e.g., from a malformed config file) could still wrap or cause an `EXC_BAD_ACCESS` when allocating `Data(count:)`.
- **Impact**: Unexpected memory allocation failure or crash.
- **Fix**: Clamp the value:
```swift
let mb = max(1, min(config.bufferSizeMB, 2_048)) // 1 MB – 2 GB
let bufferCapacity = mb * 1_024 * 1_024
```

---

### [M-006] Hardcoded User Identifier
- **File**: `packages/sdk-macos/Sources/Chronoscope/Core/CaptureSession.swift`
- **Line**: 101
- **Code**:
```swift
"user_id": "macos_user"
```
- **Issue**: A hardcoded `user_id` string is sent during session initialization. This breaks multi-user tracking and may violate privacy regulations by conflating distinct users.
- **Impact**: Analytics/auditing inaccuracy, potential compliance issue.
- **Fix**: Accept `userId` as a parameter in `CaptureConfig`, or derive it from a stable system identifier (with user consent).

---

### [M-007] Unsafe Build Flag Links to Local Relative Path
- **File**: `packages/sdk-macos/Package.swift`
- **Line**: 24
- **Code**:
```swift
.unsafeFlags(["-L", "../../services/privacy-engine/target/release"])
```
- **Issue**: The package uses `.unsafeFlags` to link against a library at a relative path outside the package root. This makes the build non-hermetic, breaks on different directory layouts, and could be exploited by placing a malicious library at that path.
- **Impact**: Build fragility and supply-chain risk.
- **Fix**: Vendor the binary into the package (e.g., `Sources/ChronoscopePrivacyC/lib/`) or consume it via a binary target / XCFramework.

---

### [M-008] Insufficient Test Coverage
- **File**: `packages/sdk-macos/Tests/ChronoscopeTests.swift`
- **Lines**: 1–40
- **Issue**: Only three unit tests exist, covering default config values and basic buffer read/write. There are **zero** tests for:
  - `CaptureSession` start/stop lifecycle
  - `FrameCapture` (requires mocking `SCStream`, but integration tests are still possible)
  - `ChunkUploader` network logic (URL construction, multipart body, error paths)
  - `PrivacyEngine` (C interop, text redaction)
  - `CircularBuffer` edge cases (zero capacity, overwrite behavior, concurrent writes)
  - Error handling paths (`sessionInitFailed`, `uploadFailed`)
- **Impact**: Regressions in capture, privacy, and networking logic go undetected.
- **Fix**: Expand the test suite. At minimum add:
  - `XCTest` cases for error paths in `ChunkUploader`
  - Mock `URLProtocol` to test network failures
  - Buffer overflow/wrap-around tests
  - Actor concurrency tests using `await` expectations

---

## LOW Findings

### [L-001] Print Statements Used Instead of Structured Logging
- **Files**:
  - `packages/sdk-macos/Sources/Chronoscope/Capture/FrameCapture.swift` (lines 27, 43)
  - `packages/sdk-macos/Sources/Chronoscope/Core/CaptureSession.swift` (line 86)
  - `packages/sdk-macos/Sources/Chronoscope/Network/ChunkUploader.swift` (lines 29, 45)
- **Issue**: `print()` statements are used for error and status logging. These bypass the unified logging system (`OSLog`), are invisible in production diagnostics, and may leak sensitive data into system logs.
- **Fix**: Replace with `Logger` from `OSLog`:
```swift
import OSLog
private let logger = Logger(subsystem: "dev.chronoscope.sdk", category: "CaptureSession")
// logger.error("Upload failed for chunk \(index): \(error)")
```

---

### [L-002] Missing Documentation on Public APIs
- **Files**: All public interfaces (`CaptureConfig.swift`, `Chronoscope.swift`, `CaptureSession.swift`, etc.)
- **Issue**: No DocC-style documentation comments (`///`) on public structs, enums, classes, or methods. This hinders developer adoption and IDE autocomplete quality.
- **Fix**: Add `///` documentation to all `public` declarations.

---

### [L-003] Errors Swallowed by `try?` in Upload Loop
- **File**: `packages/sdk-macos/Sources/Chronoscope/Core/CaptureSession.swift`
- **Line**: 46
- **Code**:
```swift
try? await Task.sleep(nanoseconds: 10_000_000_000)
```
- **Issue**: `Task.sleep` only throws if the task is cancelled. Using `try?` suppresses the cancellation error, which is actually the desired signal to exit the loop. The surrounding `while !Task.isCancelled` check makes this mostly harmless, but it is still a code-smell because it masks the structured-concurrency cancellation mechanism.
- **Fix**: Use a bare `try` and let cancellation propagate naturally:
```swift
try await Task.sleep(nanoseconds: 10_000_000_000)
```
(The loop will break when the task is cancelled and `stop()` nils the task.)

---

### [L-004] FrameCapture Missing Deinit Stream Cleanup
- **File**: `packages/sdk-macos/Sources/Chronoscope/Capture/FrameCapture.swift`
- **Lines**: 47–53
- **Issue**: `stop()` properly stops and nils the stream, but there is no `deinit`. If the `FrameCapture` actor is deallocated while `stream` is non-nil, the underlying `SCStream` may continue running until the OS reclaims it.
- **Fix**: Add a `deinit` (actors support `deinit`) that stops the stream synchronously or logs a warning:
```swift
deinit {
    if stream != nil {
        assertionFailure("FrameCapture deallocated without calling stop()")
    }
}
```

---

## Passed Security Checks

| Check | Status | Justification |
|-------|--------|---------------|
| No forced unwraps except in tests | ❌ FAIL | See C-002, M-001 |
| `SCStream` properly stopped and nil'd on deinit | ⚠️ PARTIAL | `stop()` handles it, but no `deinit` safety net (L-004) |
| Memory: no retain cycles in closures | ✅ PASS | `CaptureSession` uses `[weak self]` in all closures (lines 36, 37, 44) |
| Network: timeout configured on `URLSession` | ❌ FAIL | See H-001 |
| Privacy: PII masking applied BEFORE upload | ❌ FAIL | See C-001 |
| No hardcoded secrets | ✅ PASS | `apiKey` is injected via `CaptureConfig`; no embedded credentials found |
| Buffer bounds checking | ❌ FAIL | See C-003 |
| Input validation on public APIs | ❌ FAIL | `CaptureConfig` accepts arbitrary `frameRate`, `bufferSizeMB` without validation |
