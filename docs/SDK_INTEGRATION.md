# SDK Integration Guide

This guide explains how to integrate the Chronoscope capture SDK into your native desktop application.

---

## Table of Contents

- [macOS (Swift)](#macos-swift)
- [Linux (Rust)](#linux-rust)
- [Windows (C++)](#windows-c)
- [Privacy Configuration](#privacy-configuration)
- [Event Tracking](#event-tracking)

---

## macOS (Swift)

### Requirements

- macOS 12.3+
- Xcode 15+
- Swift 5.9+
- ScreenCaptureKit entitlement

### Installation

Add the SDK package to your `Package.swift`:

```swift
// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "YourApp",
    dependencies: [
        .package(url: "https://github.com/etherman-os/chronoscope.git", from: "0.1.0")
    ],
    targets: [
        .executableTarget(
            name: "YourApp",
            dependencies: [
                .product(name: "Chronoscope", package: "chronoscope")
            ]
        )
    ]
)
```

Or drag the `packages/sdk-macos` folder into your Xcode project.

### Basic Usage

```swift
import Chronoscope

let config = CaptureConfig(
    apiKey: "your-api-key",
    endpoint: URL(string: "https://chronoscope.example.com/v1")!,
    captureMode: .hybrid,
    frameRate: 10,
    bufferSizeMB: 100,
    userId: "user-123"
)

Task {
    try await Chronoscope.shared.start(config: config)
}

// Later, to stop capture
Task {
    await Chronoscope.shared.stop()
}
```

### Entitlements

Add to your `.entitlements` file:

```xml
<key>com.apple.security.screen-recording</key>
<true/>
```

---

## Linux (Rust)

### Requirements

- Rust 1.75+
- PipeWire (Wayland) or X11 development headers
- `libx11-dev`, `libxext-dev` (X11)
- `libpipewire-0.3-dev` (Wayland)

### Add Dependency

```toml
# Cargo.toml
[dependencies]
chronoscope-sdk = { git = "https://github.com/etherman-os/chronoscope.git", subdir = "packages/sdk-linux" }
```

### Basic Usage

```rust
use chronoscope_sdk::{CaptureConfig, CaptureMode};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let config = CaptureConfig::new("your-api-key", "https://chronoscope.example.com/v1");

    let mut capture = chronoscope_sdk::LinuxCapture::new(config)?;
    capture.start().await?;

    // ... application logic ...

    capture.stop().await?;
    Ok(())
}
```

The SDK auto-detects the display server via `WAYLAND_DISPLAY` or `DISPLAY` environment variables.

---

## Windows (C++)

### Requirements

- Windows 10 1903+ or Windows 11
- Visual Studio 2022
- C++20
- WinRT Graphics Capture API

### Integration

Link against the Chronoscope SDK static library:

```cmake
# CMakeLists.txt
add_subdirectory(packages/sdk-windows)
target_link_libraries(YourApp PRIVATE chronoscope_sdk)
```

### Basic Usage

```cpp
#include <chronoscope/sdk.h>

int main() {
    chronoscope::CaptureConfig config;
    config.api_key = "your-api-key";
    config.endpoint = "https://chronoscope.example.com/v1";
    config.mode = chronoscope::CaptureMode::Hybrid;
    config.frame_rate = 10;
    config.buffer_size_mb = 100;

    auto session = chronoscope::Chronoscope::Instance().StartSession(
        config,
        GetDesktopWindow(),
        [](const std::vector<uint8_t>& frame) {
            // Process frame
        }
    );

    // ... application logic ...

    chronoscope::Chronoscope::Instance().StopAllSessions();
    return 0;
}
```

---

## Privacy Configuration

All SDKs integrate with the **Privacy Engine** to redact sensitive UI elements before frames leave the device.

### macOS

```swift
let privacyConfig = PrivacyConfig(
    detectCreditCards: true,
    detectEmails: true,
    detectPasswords: true,
    detectSSN: false,
    redactionMode: "blackout",
    customPatterns: ["\\bCONFIDENTIAL\\b"],
    excludedApps: ["com.apple.keychainaccess"]
)

let engine = PrivacyEngine(config: privacyConfig)
```

### Linux

```rust
use chronoscope_privacy::{PrivacyConfig, RedactionMode};

let privacy_config = PrivacyConfig {
    detect_credit_cards: true,
    detect_emails: true,
    detect_passwords: true,
    detect_ssn: false,
    redaction_mode: RedactionMode::Blackout,
    custom_patterns: vec![],
    excluded_apps: vec![],
};

let engine = chronoscope_privacy::PrivacyEngine::new(privacy_config);
```

### Windows

The Windows SDK exposes `SetPrivacyFilter` to exclude specific window titles from capture:

```cpp
session->SetPrivacyFilter({
    "Password Manager",
    "Banking Portal"
});
```

> **Note**: Redaction happens locally inside the SDK before upload. No sensitive pixels are transmitted.

---

## Event Tracking

### Automatic Events

The SDKs automatically capture:
- Mouse clicks (x, y, target)
- Scroll events
- Window resize
- Focus changes

### Custom Events

Track application-specific actions via the REST API directly or through SDK wrappers.

Send custom events using the `events` endpoint:

```bash
curl -X POST "https://chronoscope.example.com/v1/sessions/${SESSION_ID}/events" \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {
        "event_type": "checkout_completed",
        "timestamp_ms": 5000,
        "payload": {"total": 49.99}
      }
    ]
  }'
```

---

## Best Practices

1. **Start early**: Initialize the SDK as soon as your app launches to capture the full session.
2. **Stop gracefully**: Always call `stop()` before app termination to finalize uploads.
3. **Respect user consent**: Do not start capture if the user has not consented.
4. **Use privacy rules**: Redact sensitive fields by default; never rely on server-side redaction alone.
5. **Rate limit custom events**: Avoid flooding with high-frequency custom events.

---

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| No video uploaded | Missing screen recording permission | Check OS entitlements / permissions |
| Events not appearing | API key invalid | Verify `X-API-Key` header value |
| High CPU usage | Frame rate too high | Lower `frameRate` to 5-10 |
| Blurry replay | Compression too aggressive | Adjust encoder settings in Processor config |
