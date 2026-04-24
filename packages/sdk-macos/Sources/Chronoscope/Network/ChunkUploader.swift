import Foundation

public actor ChunkUploader {
    private let endpoint: URL
    private let apiKey: String
    private let sessionId: String
    private let session: URLSession

    public init(endpoint: URL, apiKey: String, sessionId: String) {
        self.endpoint = endpoint
        self.apiKey = apiKey
        self.sessionId = sessionId
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 300
        self.session = URLSession(configuration: config)
    }

    public func uploadChunk(data: Data, index: Int) async throws {
        let url = endpoint.appendingPathComponent("sessions/\(sessionId)/chunks")
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
        request.setValue("\(index)", forHTTPHeaderField: "X-Chunk-Index")

        let boundary = UUID().uuidString
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        request.httpBody = createMultipartBody(data: data, index: index, boundary: boundary)

        let (_, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw ChronoscopeError.uploadFailed("Invalid response")
        }
        print("Chunk upload status: \(httpResponse.statusCode)")
        guard (200...299).contains(httpResponse.statusCode) else {
            throw ChronoscopeError.uploadFailed("HTTP \(httpResponse.statusCode)")
        }
    }

    public func finalize() async throws {
        let url = endpoint.appendingPathComponent("sessions/\(sessionId)/complete")
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")

        let (_, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw ChronoscopeError.uploadFailed("Invalid response")
        }
        print("Finalize status: \(httpResponse.statusCode)")
        guard (200...299).contains(httpResponse.statusCode) else {
            throw ChronoscopeError.uploadFailed("HTTP \(httpResponse.statusCode)")
        }
    }

    private func createMultipartBody(data: Data, index: Int, boundary: String) -> Data {
        var body = Data()
        let filename = "chunk_\(index).jpg"
        let mimeType = "image/jpeg"

        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"chunk\"; filename=\"\(filename)\"\r\n".data(using: .utf8)!)
        body.append("Content-Type: \(mimeType)\r\n\r\n".data(using: .utf8)!)
        body.append(data)
        body.append("\r\n".data(using: .utf8)!)
        body.append("--\(boundary)--\r\n".data(using: .utf8)!)

        return body
    }
}
