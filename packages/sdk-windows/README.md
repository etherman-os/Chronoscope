# Chronoscope Windows SDK

Native Windows capture SDK for the Chronoscope platform.

## Requirements

- Windows 10 version 1903+ (for `Windows.Graphics.Capture`)
- Visual Studio 2022 (v17+) with C++20 support
- CMake 3.20 or later
- Windows SDK 10.0.22000.0 or later

## Build Instructions

```powershell
cd packages/sdk-windows
cmake -B build -S . -A x64
cmake --build build --config Release
```

## Run Tests

```powershell
cd build
ctest -C Release --output-on-failure
```

## Usage Example

```cpp
#include <chronoscope/sdk.h>

int main() {
    auto& sdk = chronoscope::Chronoscope::Instance();

    chronoscope::CaptureConfig cfg;
    cfg.api_key = "YOUR_API_KEY";
    cfg.endpoint = "https://ingest.chronoscope.io";
    cfg.mode = chronoscope::CaptureMode::Hybrid;
    cfg.quality = chronoscope::CaptureQuality::High;
    cfg.frame_rate = 30;

    auto session = sdk.StartSession(cfg, hwnd, [](const std::vector<uint8_t>& frame) {
        // Optional per-frame callback
    });

    // ... later ...
    session->Stop();
    return 0;
}
```

## API Overview

- `Chronoscope::Instance()` — singleton entry point
- `StartSession(config, hwnd, callback)` — begins capture for a window
- `Session::Start/Stop/Pause` — lifecycle control
- `Session::SetPrivacyFilter(titles)` — redact windows by title substring
- `CaptureMode` — Video, Events, or Hybrid
- `CaptureQuality` — Low, Medium, High

## Architecture

- **GraphicsCapture** — WinRT `Windows.Graphics.Capture` + D3D11 frame pool
- **CircularBuffer** — thread-safe ring buffer for encoded frames
- **ChunkUploader** — WinHTTP multipart/form-data upload
- **PrivacyFilter** — window-title matching for redaction
- **InputHook** — opt-in low-level mouse/keyboard hooks

## Building with Privacy Engine

### Prerequisites
1. Install Rust: https://rustup.rs/
2. Install Visual Studio 2022 with C++ workload

### Build Steps
1. Build the privacy engine:
   ```cmd
   cd services\privacy-engine
   cargo build --release --target x86_64-pc-windows-msvc
   ```

2. Build the Windows SDK:
   ```cmd
   cd packages\sdk-windows
   cmake -B build -S . -A x64
   cmake --build build --config Release
   ```

## License

MIT
