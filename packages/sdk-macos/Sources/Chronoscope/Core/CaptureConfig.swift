import Foundation

public struct CaptureConfig {
    public let apiKey: String
    public let endpoint: URL
    public let captureMode: CaptureMode
    public let quality: CaptureQuality
    public let frameRate: Int
    public let bufferSizeMB: Int

    public init(
        apiKey: String,
        endpoint: URL,
        captureMode: CaptureMode = .hybrid,
        quality: CaptureQuality = .medium,
        frameRate: Int = 10,
        bufferSizeMB: Int = 100
    ) {
        self.apiKey = apiKey
        self.endpoint = endpoint
        self.captureMode = captureMode
        self.quality = quality
        self.frameRate = frameRate
        self.bufferSizeMB = bufferSizeMB
    }
}

public enum CaptureMode: String, Sendable {
    case video, events, hybrid
}

public enum CaptureQuality: String, Sendable {
    case low, medium, high
}
