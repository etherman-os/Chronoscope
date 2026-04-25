import Foundation
import OSLog

private let logger = Logger(subsystem: "dev.chronoscope.sdk", category: "ChunkUploader")

/// Uploads capture chunks to the Chronoscope service.
public actor ChunkUploader {
    private let endpoint: URL
    private let apiKey: String
    private let sessionId: String
    private let session: URLSession

    /// Creates a new chunk uploader.
    /// - Parameters:
    ///   - endpoint: Base service endpoint.
    ///   - apiKey: API key for authentication.
    ///   - sessionId: Active session identifier.
    public init(endpoint: URL, apiKey: String, sessionId: String) {
        self.endpoint = endpoint
        self.apiKey = apiKey
        self.sessionId = sessionId
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 300
        self.session = URLSession(configuration: config)
    }

    /// Uploads a single chunk.
    /// - Parameters:
    ///   - data: Raw chunk data.
    ///   - index: Chunk sequence index.
    ///   - filename: Optional filename. Defaults to `"chunk_<index>.jpg"`.
    ///   - mimeType: Optional MIME type. Defaults to `"image/jpeg"`.
    public func uploadChunk(
        data: Data,
        index: Int,
        filename: String? = nil,
        mimeType: String? = nil
    ) async throws {
        let url = chunkURL()
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
        request.setValue("\(index)", forHTTPHeaderField: "X-Chunk-Index")

        let boundary = UUID().uuidString
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        request.httpBody = createMultipartBody(
            data: data,
            index: index,
            boundary: boundary,
            filename: filename,
            mimeType: mimeType
        )

        let (_, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw ChronoscopeError.uploadFailed("Invalid response")
        }
        logger.info("Chunk upload status: \(httpResponse.statusCode)")
        guard (200...299).contains(httpResponse.statusCode) else {
            throw ChronoscopeError.uploadFailed("HTTP \(httpResponse.statusCode)", statusCode: httpResponse.statusCode)
        }
    }

    /// Finalizes the upload session.
    public func finalize() async throws {
        let url = finalizeURL()
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")

        let (_, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw ChronoscopeError.uploadFailed("Invalid response")
        }
        logger.info("Finalize status: \(httpResponse.statusCode)")
        guard (200...299).contains(httpResponse.statusCode) else {
            throw ChronoscopeError.uploadFailed("HTTP \(httpResponse.statusCode)", statusCode: httpResponse.statusCode)
        }
    }

    /// Returns the URL for chunk uploads.
    internal func chunkURL() -> URL {
        endpoint
            .appendingPathComponent("sessions")
            .appendingPathComponent(sessionId)
            .appendingPathComponent("chunks")
    }

    /// Returns the URL for session finalization.
    internal func finalizeURL() -> URL {
        endpoint
            .appendingPathComponent("sessions")
            .appendingPathComponent(sessionId)
            .appendingPathComponent("complete")
    }

    internal func createMultipartBody(
        data: Data,
        index: Int,
        boundary: String,
        filename: String? = nil,
        mimeType: String? = nil
    ) -> Data {
        var body = Data()
        let file = filename ?? "chunk_\(index).jpg"
        let mime = mimeType ?? "image/jpeg"

        body.append(utf8Data("--\(boundary)\r\n"))
        body.append(utf8Data("Content-Disposition: form-data; name=\"chunk\"; filename=\"\(file)\"\r\n"))
        body.append(utf8Data("Content-Type: \(mime)\r\n\r\n"))
        body.append(data)
        body.append(utf8Data("\r\n"))
        body.append(utf8Data("--\(boundary)--\r\n"))

        return body
    }
}

/// Returns the UTF-8 encoded representation of a string, falling back to the string's `.utf8` view.
private func utf8Data(_ string: String) -> Data {
    string.data(using: .utf8) ?? Data(string.utf8)
}
