import Foundation

/// Main entry point for the Chronoscope SDK.
public actor Chronoscope {
    /// Shared singleton instance.
    public static let shared = Chronoscope()
    private var session: CaptureSession?

    /// Internal initializer for testability.
    internal init() {}

    /// Whether a capture session is currently active.
    public var isRunning: Bool {
        session != nil
    }

    /// Starts a new capture session with the given configuration.
    /// - Parameter config: Capture configuration.
    public func start(config: CaptureConfig) async throws {
        guard session == nil else {
            return
        }
        let newSession = CaptureSession(config: config)
        try await newSession.start()
        session = newSession
    }

    /// Stops the current capture session.
    public func stop() async {
        await session?.stop()
        session = nil
    }
}
