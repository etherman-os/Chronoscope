import XCTest
@testable import Chronoscope

final class ChronoscopeTests: XCTestCase {
    func testCaptureConfigDefaults() {
        let endpoint = URL(string: "https://api.chronoscope.dev/v1")!
        let config = CaptureConfig(apiKey: "test_key", endpoint: endpoint)

        XCTAssertEqual(config.apiKey, "test_key")
        XCTAssertEqual(config.endpoint, endpoint)
        XCTAssertEqual(config.captureMode, .hybrid)
        XCTAssertEqual(config.quality, .medium)
        XCTAssertEqual(config.frameRate, 10)
        XCTAssertEqual(config.bufferSizeMB, 100)
    }

    func testCircularBuffer() async {
        let buffer = CircularBuffer(capacity: 1024)
        let data = "Hello, World!".data(using: .utf8)!
        await buffer.write(data)

        let chunk = await buffer.readChunk()
        XCTAssertNotNil(chunk)
        XCTAssertEqual(chunk, data)
    }

    func testCircularBufferWrapAround() async {
        let buffer = CircularBuffer(capacity: 16)
        let data1 = "1234567890".data(using: .utf8)!
        let data2 = "ABCDEFGHIJ".data(using: .utf8)!

        await buffer.write(data1)
        _ = await buffer.readChunk()

        await buffer.write(data2)
        let chunk = await buffer.readChunk()
        XCTAssertNotNil(chunk)
        XCTAssertEqual(chunk, data2)
    }
}
