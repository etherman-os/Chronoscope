import Foundation

public enum ChronoscopeError: Error, Sendable {
    case sessionInitFailed(String)
    case uploadFailed(String)
    case captureFailed(String)
}
