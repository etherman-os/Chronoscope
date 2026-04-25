import Foundation

/// Configuration for a Chronoscope capture session.
public struct CaptureConfig {
    /// API key for authentication with the Chronoscope service.
    public let apiKey: String
    /// Base endpoint URL for the Chronoscope service.
    public let endpoint: URL
    /// Capture mode (video, events, or hybrid).
    public let captureMode: CaptureMode
    /// Target frame rate for screen capture.
    public let frameRate: Int
    /// Buffer size in megabytes (clamped to 1–2048 MB at runtime).
    public let bufferSizeMB: Int
    /// User identifier sent during session initialization.
    public let userId: String

    /// Creates a new capture configuration.
    /// - Parameters:
    ///   - apiKey: API key for authentication.
    ///   - endpoint: Base endpoint URL.
    ///   - captureMode: Desired capture mode. Defaults to `.hybrid`.
    ///   - frameRate: Target frame rate. Must be greater than 0. Defaults to `10`.
    ///   - bufferSizeMB: Buffer size in megabytes. Must be greater than 0. Defaults to `100`.
    ///   - userId: User identifier. Defaults to `"macos_user"`.
    public init(
        apiKey: String,
        endpoint: URL,
        captureMode: CaptureMode = .hybrid,
        frameRate: Int = 10,
        bufferSizeMB: Int = 100,
        userId: String = "macos_user"
    ) {
        precondition(frameRate > 0, "frameRate must be greater than 0")
        precondition(bufferSizeMB > 0, "bufferSizeMB must be greater than 0")
        self.apiKey = apiKey
        self.endpoint = endpoint
        self.captureMode = captureMode
        self.frameRate = frameRate
        self.bufferSizeMB = bufferSizeMB
        self.userId = userId
    }
}

/// Capture mode options.
public enum CaptureMode: String, Sendable {
    case video, events, hybrid
}
