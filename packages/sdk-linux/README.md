# Chronoscope Linux SDK

Rust-based screen capture SDK for Linux, supporting both **Wayland** (via PipeWire) and **X11** (via MIT-SHM).

## Requirements

- Linux kernel 5.10+
- Rust 1.75+
- **Wayland**: PipeWire 0.3+, xdg-desktop-portal
- **X11**: X11 display server with MIT-SHM extension

## Display Server Auto-Detection

The SDK automatically detects the display server at runtime:

1. Checks `WAYLAND_DISPLAY` environment variable first
2. Falls back to `DISPLAY` environment variable
3. Returns an error if neither is set

## Build Instructions

```bash
cd packages/sdk-linux
cargo build --release
```

## Running Tests

```bash
cargo test
```

## Privacy Engine Integration

Unlike other platform SDKs, the Linux SDK integrates with the Chronoscope Privacy Engine as a **direct Rust crate dependency** (not via C ABI). The privacy engine is referenced from `../../services/privacy-engine`.

The privacy engine is initialized during `LinuxCapture::new()` with the following default configuration:

- Credit card detection: enabled
- Email detection: enabled
- Password detection: enabled
- SSN detection: disabled
- Redaction mode: blackout

## Usage Example

```rust
use chronoscope_sdk_linux::{CaptureConfig, LinuxCapture, CaptureMode, CaptureQuality};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let config = CaptureConfig {
        api_key: "your-api-key".to_string(),
        endpoint: "https://api.chronoscope.dev".to_string(),
        capture_mode: CaptureMode::Hybrid,
        quality: CaptureQuality::High,
        frame_rate: 10,
        buffer_size_mb: 100,
    };

    let mut capture = LinuxCapture::new(config)?;
    capture.start().await?;

    // Capture runs in background...
    tokio::time::sleep(std::time::Duration::from_secs(60)).await;

    capture.stop().await?;
    Ok(())
}
```

## Architecture

- `src/capture/wayland.rs` – PipeWire-based screen capture for Wayland compositors
- `src/capture/x11.rs` – MIT-SHM-based screen capture for X11
- `src/buffer.rs` – Thread-safe circular buffer for frame data
- `src/upload.rs` – HTTP chunk uploader (multipart/form-data)
- `src/input.rs` – Input event capture stub (evdev/libinput planned)
- `src/config.rs` – Capture configuration and quality settings
