# Chronoscope macOS SDK

A Swift SDK for capturing screen sessions and events on macOS using ScreenCaptureKit.

## Requirements

- macOS 12.3+
- Swift 5.9+
- Xcode 15.0+

## Installation

Add the following to your `Package.swift` dependencies:

```swift
dependencies: [
    .package(url: "https://github.com/chronoscope/sdk-macos.git", from: "1.0.0")
]
```

Or add it via Xcode:
1. Go to **File → Add Package Dependencies**
2. Enter the repository URL
3. Select the `Chronoscope` product

## Usage

```swift
import Chronoscope

let config = CaptureConfig(
    apiKey: "your-api-key",
    endpoint: URL(string: "https://api.chronoscope.dev/v1")!,
    captureMode: .hybrid,
    frameRate: 10
)

// Start capturing
try await Chronoscope.shared.start(config: config)

// Stop capturing
await Chronoscope.shared.stop()
```

## Capture Modes

- **video**: Captures screen frames only
- **events**: Captures user interaction events only
- **hybrid**: Captures both frames and events (default)

## Privacy

The SDK includes a `PrivacyEngine` for future PII masking capabilities.

## Building with Privacy Engine

The SDK links against `libchronoscope_privacy` via the `ChronoscopePrivacyC` system-library target. Ensure the library is available in the linker search path (e.g., via `LD_LIBRARY_PATH` or by vendoring the binary into the package), then build:

```bash
swift build
```

## License

MIT
