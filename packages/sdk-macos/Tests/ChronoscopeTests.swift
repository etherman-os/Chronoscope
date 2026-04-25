import XCTest
@testable import Chronoscope

final class ChronoscopeTests: XCTestCase {
    // MARK: - Helpers

    private func makeEndpoint() -> URL {
        guard let url = URL(string: "https://api.chronoscope.dev/v1") else {
            XCTFail("Failed to create test endpoint URL")
            return URL(fileURLWithPath: "")
        }
        return url
    }

    private func makeData(_ string: String) -> Data {
        guard let data = string.data(using: .utf8) else {
            XCTFail("Failed to encode string to Data")
            return Data()
        }
        return data
    }

    // MARK: - CaptureConfig

    func testCaptureConfigDefaults() {
        let endpoint = makeEndpoint()
        let config = CaptureConfig(apiKey: "test_key", endpoint: endpoint)

        XCTAssertEqual(config.apiKey, "test_key")
        XCTAssertEqual(config.endpoint, endpoint)
        XCTAssertEqual(config.captureMode, .hybrid)
        XCTAssertEqual(config.frameRate, 10)
        XCTAssertEqual(config.bufferSizeMB, 100)
        XCTAssertEqual(config.userId, "macos_user")
    }

    func testCaptureConfigUserId() {
        let endpoint = makeEndpoint()
        let config = CaptureConfig(apiKey: "test_key", endpoint: endpoint, userId: "alice")
        XCTAssertEqual(config.userId, "alice")
    }

    func testCaptureConfigValidationAcceptsValidValues() {
        let endpoint = makeEndpoint()
        let config = CaptureConfig(apiKey: "k", endpoint: endpoint, frameRate: 1, bufferSizeMB: 1)
        XCTAssertEqual(config.frameRate, 1)
        XCTAssertEqual(config.bufferSizeMB, 1)
    }

    // MARK: - CircularBuffer

    func testCircularBuffer() async {
        let buffer = CircularBuffer(capacity: 1024)
        let data = makeData("Hello, World!")
        await buffer.write(data)

        let chunk = await buffer.readChunk()
        XCTAssertNotNil(chunk)
        XCTAssertEqual(chunk, data)
    }

    func testCircularBufferWrapAround() async {
        let buffer = CircularBuffer(capacity: 16)
        let data1 = makeData("1234567890")
        let data2 = makeData("ABCDEFGHIJ")

        await buffer.write(data1)
        _ = await buffer.readChunk()

        await buffer.write(data2)
        let chunk = await buffer.readChunk()
        XCTAssertNotNil(chunk)
        XCTAssertEqual(chunk, data2)
    }

    func testCircularBufferEmptyReadReturnsNil() async {
        let buffer = CircularBuffer(capacity: 64)
        let chunk = await buffer.readChunk()
        XCTAssertNil(chunk)
    }

    func testCircularBufferOverwrite() async {
        let buffer = CircularBuffer(capacity: 8)
        let data1 = makeData("abcdefgh")
        let data2 = makeData("12345678")

        await buffer.write(data1)
        await buffer.write(data2)

        let chunk = await buffer.readChunk()
        XCTAssertNotNil(chunk)
        // After overwrite, the buffer should contain the latest bytes.
        XCTAssertEqual(chunk, data2)
    }

    // MARK: - ChunkUploader

    func testChunkUploaderURLConstruction() async {
        let endpoint = makeEndpoint()
        let uploader = ChunkUploader(endpoint: endpoint, apiKey: "key", sessionId: "sess-123")

        let chunkURL = await uploader.chunkURL()
        let expectedChunk = endpoint
            .appendingPathComponent("sessions")
            .appendingPathComponent("sess-123")
            .appendingPathComponent("chunks")
        XCTAssertEqual(chunkURL, expectedChunk)

        let finalizeURL = await uploader.finalizeURL()
        let expectedFinalize = endpoint
            .appendingPathComponent("sessions")
            .appendingPathComponent("sess-123")
            .appendingPathComponent("complete")
        XCTAssertEqual(finalizeURL, expectedFinalize)
    }

    func testChunkUploaderMultipartBodyDefaults() async {
        let endpoint = makeEndpoint()
        let uploader = ChunkUploader(endpoint: endpoint, apiKey: "key", sessionId: "s")
        let data = makeData("payload")
        let body = await uploader.createMultipartBody(data: data, index: 7, boundary: "XYZ")
        guard let bodyString = String(data: body, encoding: .utf8) else {
            XCTFail("Failed to decode multipart body")
            return
        }

        XCTAssertTrue(bodyString.contains("--XYZ\r\n"))
        XCTAssertTrue(bodyString.contains("name=\"chunk\""))
        XCTAssertTrue(bodyString.contains("filename=\"chunk_7.jpg\""))
        XCTAssertTrue(bodyString.contains("Content-Type: image/jpeg"))
        XCTAssertTrue(bodyString.contains("payload"))
        XCTAssertTrue(bodyString.contains("--XYZ--\r\n"))
    }

    func testChunkUploaderMultipartBodyCustom() async {
        let endpoint = makeEndpoint()
        let uploader = ChunkUploader(endpoint: endpoint, apiKey: "key", sessionId: "s")
        let data = makeData("payload")
        let body = await uploader.createMultipartBody(
            data: data,
            index: 3,
            boundary: "ABC",
            filename: "frame.hevc",
            mimeType: "video/hevc"
        )
        guard let bodyString = String(data: body, encoding: .utf8) else {
            XCTFail("Failed to decode multipart body")
            return
        }

        XCTAssertTrue(bodyString.contains("filename=\"frame.hevc\""))
        XCTAssertTrue(bodyString.contains("Content-Type: video/hevc"))
    }

    // MARK: - Chronoscope

    func testChronoscopeIsRunning() async {
        let chronoscope = Chronoscope()
        let running = await chronoscope.isRunning
        XCTAssertFalse(running)
    }

    func testChronoscopeInternalInit() {
        // Verify that the internal initializer is accessible for testing.
        let c1 = Chronoscope()
        let c2 = Chronoscope()
        XCTAssertTrue(c1 !== c2)
    }

    // MARK: - ChronoscopeError

    func testChronoscopeErrorStructuredPayload() {
        let err1 = ChronoscopeError.uploadFailed("Bad request", statusCode: 400)
        if case .uploadFailed(let msg, let code) = err1 {
            XCTAssertEqual(msg, "Bad request")
            XCTAssertEqual(code, 400)
        } else {
            XCTFail("Expected uploadFailed")
        }

        let err2 = ChronoscopeError.sessionInitFailed("Unauthorized", statusCode: 401)
        if case .sessionInitFailed(let msg, let code) = err2 {
            XCTAssertEqual(msg, "Unauthorized")
            XCTAssertEqual(code, 401)
        } else {
            XCTFail("Expected sessionInitFailed")
        }
    }

    // MARK: - PrivacyConfig

    func testPrivacyConfigCodingKeys() throws {
        let config = PrivacyConfig(
            detectCreditCards: false,
            detectEmails: true,
            detectPasswords: false,
            detectSSN: true,
            redactionMode: "blur",
            customPatterns: ["\\d{4}"],
            excludedApps: ["Safari"]
        )
        let data = try JSONEncoder().encode(config)
        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        XCTAssertEqual(json?["detect_credit_cards"] as? Bool, false)
        XCTAssertEqual(json?["detect_emails"] as? Bool, true)
        XCTAssertEqual(json?["redaction_mode"] as? String, "blur")
    }

    // MARK: - FrameCapture

    func testFrameCaptureInitialization() {
        let capture = FrameCapture(frameRate: 30)
        XCTAssertNotNil(capture)
    }

    // MARK: - CaptureSession

    func testCaptureSessionInitialization() {
        let endpoint = makeEndpoint()
        let config = CaptureConfig(apiKey: "key", endpoint: endpoint)
        let session = CaptureSession(config: config)
        XCTAssertNotNil(session)
    }
}
