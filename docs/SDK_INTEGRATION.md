# SDK Integration Guide

This guide explains how to integrate the Chronoscope capture SDK into your native desktop application.

---

## Table of Contents

- [macOS (Swift)](#macos-swift)
- [Windows (C++)](#windows-c)
- [Linux (Rust)](#linux-rust)
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

Add the SDK package to your project:

```swift
// Package.swift
dependencies: [
    .package(url: "https://github.com/etherman-os/chronoscope.git", from: "0.1.0")
]
```

Or drag the `packages/sdk-macos` folder into your Xcode project.

### Basic Usage

```swift
import Chronoscope

let config = CaptureConfig(
    apiKey: "your-api-key",
    apiEndpoint: "https://chronoscope.example.com/v1",
    captureMode: .hybrid,        // .video, .events, or .hybrid
    maxFrameRate: 10,
    privacyRules: [
        .redact(selector: "#password"),
        .redact(selector: "#credit-card")
    ]
)

let session = ChronoscopeSession(config: config)

// Start capture
session.start(userId: "user-123", metadata: [
    "app_version": "1.2.3"
])

// Track custom events
session.trackEvent(
    type: "custom_checkout",
    payload: "{\"amount\": 99.99}"
)

// Stop capture
session.stop()
```

### Entitlements

Add to your `.entitlements` file:

```xml
<key>com.apple.security.screen-recording</key>
<true/>
```

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
    chronoscope::Config config;
    config.api_key = "your-api-key";
    config.api_endpoint = "https://chronoscope.example.com/v1";
    config.capture_mode = CaptureMode::Hybrid;
    config.max_frame_rate = 10;

    chronoscope::Session session(config);

    session.Start("user-123", R"({"app_version":"1.2.3"})");

    // Track a custom event
    session.TrackEvent("custom_purchase", R"({"sku":"ABC123"})");

    // ... application logic ...

    session.Stop();
    return 0;
}
```

### Permissions

Your application manifest must declare the `graphicsCaptureProgrammatic` capability:

```xml
<Package ...>
  <Capabilities>
    <Capability Name="graphicsCaptureProgrammatic"/>
  </Capabilities>
</Package>
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
chronoscope-sdk = { path = "../packages/sdk-linux" }
```

### Basic Usage

```rust
use chronoscope_sdk::{Config, CaptureMode, Session};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let config = Config {
        api_key: "your-api-key".to_string(),
        api_endpoint: "https://chronoscope.example.com/v1".to_string(),
        capture_mode: CaptureMode::Hybrid,
        max_frame_rate: 10,
        privacy_rules: vec![
            PrivacyRule::Redact("#ssn".to_string()),
        ],
    };

    let mut session = Session::new(config);

    session.start("user-123", Some(r#"{"app_version":"1.2.3"}"#)).await?;

    session.track_event("custom_login", Some(r#"{"provider":"oauth"}"#)).await?;

    // ... application logic ...

    session.stop().await?;
    Ok(())
}
```

---

## Privacy Configuration

All SDKs support a privacy rule system to redact sensitive UI elements before frames leave the device.

### Rule Types

| Rule | Description |
|------|-------------|
| `Redact(selector)` | Blackout the region matching the selector |
| `Blur(selector, radius)` | Gaussian blur the region |
| `Replace(selector, text)` | Overlay replacement text |

### macOS Example

```swift
config.privacyRules = [
    .redact(selector: "input[type=password]"),
    .blur(selector: ".sensitive-document", radius: 20),
    .replace(selector: "#email", text: "[EMAIL REDACTED]")
]
```

### Windows Example

```cpp
config.privacy_rules = {
    PrivacyRule::Redact("input[type=password]"),
    PrivacyRule::Blur(".sensitive-document", 20),
    PrivacyRule::Replace("#email", "[EMAIL REDACTED]")
};
```

### Linux Example

```rust
config.privacy_rules = vec![
    PrivacyRule::Redact("input[type=password]".to_string()),
    PrivacyRule::Blur(".sensitive-document".to_string(), 20),
    PrivacyRule::Replace("#email".to_string(), "[EMAIL REDACTED]".to_string()),
];
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

Track application-specific actions:

```swift
// macOS
session.trackEvent(type: "checkout_completed", payload: "{\"total\": 49.99}")
```

```cpp
// Windows
session.TrackEvent("checkout_completed", R"({"total":49.99})");
```

```rust
// Linux
session.track_event("checkout_completed", Some(r#"{"total":49.99}"#)).await?;
```

### Event Schema

| Field | Type | Description |
|-------|------|-------------|
| `event_type` | string | Category of event |
| `timestamp_ms` | int | Milliseconds since session start |
| `x` | int | Screen X coordinate (if applicable) |
| `y` | int | Screen Y coordinate (if applicable) |
| `target` | string | CSS-like selector or element ID |
| `payload` | string | JSON-encoded extra data |

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
| High CPU usage | Frame rate too high | Lower `maxFrameRate` to 5-10 |
| Blurry replay | Compression too aggressive | Adjust encoder settings in Processor config |
