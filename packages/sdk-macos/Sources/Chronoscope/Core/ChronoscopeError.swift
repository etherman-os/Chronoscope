import Foundation

/// Errors thrown by the Chronoscope SDK.
public enum ChronoscopeError: Error, Sendable {
    /// Session initialization failed.
    /// - Parameters:
    ///   - message: Human-readable failure reason.
    ///   - statusCode: Optional HTTP status code.
    case sessionInitFailed(String, statusCode: Int? = nil)
    /// Chunk upload failed.
    /// - Parameters:
    ///   - message: Human-readable failure reason.
    ///   - statusCode: Optional HTTP status code.
    case uploadFailed(String, statusCode: Int? = nil)
}
