# Chronoscope Privacy Engine

Cross-platform privacy engine written in Rust. Exposes a C ABI for use by Swift, C++, and Rust SDKs.

## Building

```bash
cargo build --release
```

This produces:
- `libchronoscope_privacy.so` (Linux)
- `libchronoscope_privacy.dylib` (macOS)
- `chronoscope_privacy.dll` (Windows)

## C ABI Usage

Include `include/chronoscope_privacy.h` and link against the library.

```c
#include "chronoscope_privacy.h"

ChronoscopePrivacyEngine* engine = chronoscope_privacy_init("{\"detect_emails\":true}");
char* redacted = chronoscope_privacy_process_text(engine, "email: test@example.com");
// use redacted...
chronoscope_privacy_free_string(redacted);
chronoscope_privacy_free(engine);
```

## Swift Integration

Copy `bindings/Swift/ChronoscopePrivacy.swift` into your Xcode project and link the Rust library.

```swift
let config = PrivacyConfig(detectCreditCards: true, detectEmails: true, detectPasswords: true, detectSSN: true)
let engine = PrivacyEngine(config: config)
let redacted = engine.processText("Contact me at user@example.com")
```

## Audit Logging

Redactions are logged to stderr by default. Future versions will support structured logging to file or queue.
