import Foundation

public actor CaptureSession {
    private let config: CaptureConfig
    private var sessionId: String?
    private var buffer: CircularBuffer?
    private var uploader: ChunkUploader?
    private var frameCapture: FrameCapture?
    private var privacyEngine: PrivacyEngine?
    private var uploadTask: Task<Void, Never>?
    private var chunkIndex: Int = 0
    private var isRunning: Bool = false

    init(config: CaptureConfig) {
        self.config = config
    }

    func start() async throws {
        guard !isRunning else { return }

        let sessionResponse = try await initializeSession()
        self.sessionId = sessionResponse.sessionId

        let bufferCapacity = config.bufferSizeMB * 1_024 * 1_024
        self.buffer = CircularBuffer(capacity: bufferCapacity)
        self.uploader = ChunkUploader(
            endpoint: config.endpoint,
            apiKey: config.apiKey,
            sessionId: sessionResponse.sessionId
        )

        if config.captureMode != .events {
            let privacyEngine = PrivacyEngine()
            self.privacyEngine = privacyEngine
            self.frameCapture = FrameCapture(frameRate: config.frameRate, privacyEngine: privacyEngine)
            await frameCapture?.start { [weak self] data in
                Task { [weak self] in
                    await self?.buffer?.write(data)
                }
            }
        }

        isRunning = true
        uploadTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 10_000_000_000)
                guard let self = self else { break }
                await self.uploadLoop()
            }
        }
    }

    func stop() async {
        guard isRunning else { return }
        isRunning = false

        uploadTask?.cancel()
        uploadTask = nil

        if config.captureMode != .events {
            await frameCapture?.stop()
        }
        frameCapture = nil
        privacyEngine = nil

        await uploadLoop()

        if let uploader = uploader {
            await uploader.finalize()
        }

        uploader = nil
        buffer = nil
        sessionId = nil
        chunkIndex = 0
    }

    private func uploadLoop() async {
        guard let buffer = buffer, let uploader = uploader else { return }
        if let chunk = await buffer.readChunk() {
            let index = chunkIndex
            chunkIndex += 1
            do {
                try await uploader.uploadChunk(data: chunk, index: index)
            } catch {
                print("Upload failed for chunk \(index): \(error)")
            }
        }
    }

    private func initializeSession() async throws -> SessionInitResponse {
        var request = URLRequest(url: config.endpoint.appendingPathComponent("sessions/init"))
        request.httpMethod = "POST"
        request.setValue(config.apiKey, forHTTPHeaderField: "X-API-Key")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        let osVersion = ProcessInfo.processInfo.operatingSystemVersionString
        let appVersion = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "unknown"

        let body: [String: Any] = [
            "user_id": "macos_user",
            "capture_mode": config.captureMode.rawValue,
            "metadata": [
                "os_version": osVersion,
                "app_version": appVersion
            ]
        ]

        request.httpBody = try JSONSerialization.data(withJSONObject: body)

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 201 else {
            let message = String(data: data, encoding: .utf8) ?? "Unknown error"
            throw ChronoscopeError.sessionInitFailed(message)
        }

        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        guard let sessionId = json?["session_id"] as? String else {
            throw ChronoscopeError.sessionInitFailed("Missing session_id in response")
        }

        return SessionInitResponse(sessionId: sessionId)
    }
}

private struct SessionInitResponse {
    let sessionId: String
}
