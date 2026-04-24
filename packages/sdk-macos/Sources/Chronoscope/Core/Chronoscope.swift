import Foundation

public final class Chronoscope {
    public static let shared = Chronoscope()
    private var session: CaptureSession?
    private init() {}

    public func start(config: CaptureConfig) async throws {
        guard session == nil else {
            return
        }
        let newSession = CaptureSession(config: config)
        try await newSession.start()
        session = newSession
    }

    public func stop() async {
        await session?.stop()
        session = nil
    }
}
