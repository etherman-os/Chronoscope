import Foundation
import ChronoscopePrivacyC

public actor PrivacyEngine {
    private var engine: OpaquePointer?
    
    public init?(config: PrivacyConfig = PrivacyConfig()) {
        guard let jsonData = try? JSONEncoder().encode(config),
              let jsonString = String(data: jsonData, encoding: .utf8) else {
            return nil
        }
        engine = jsonString.withCString { ptr in
            chronoscope_privacy_init(ptr)
        }
    }
    
    deinit {
        if let engine = engine {
            chronoscope_privacy_free(engine)
        }
    }
    
    public func processFrame(_ frameData: inout Data, width: UInt32, height: UInt32, stride: UInt32) {
        guard let engine = engine else { return }
        frameData.withUnsafeMutableBytes { rawBuffer in
            if let baseAddress = rawBuffer.baseAddress {
                chronoscope_privacy_process_frame(
                    engine,
                    baseAddress.assumingMemoryBound(to: UInt8.self),
                    width,
                    height,
                    stride
                )
            }
        }
    }
    
    public func processText(_ text: String) -> String {
        guard let engine = engine else { return text }
        let result = text.withCString { textPtr in
            chronoscope_privacy_process_text(engine, textPtr)
        }
        defer {
            if let result = result {
                chronoscope_privacy_free_string(result)
            }
        }
        return result.map { String(cString: $0) } ?? text
    }
}

public struct PrivacyConfig: Codable {
    public var detectCreditCards: Bool = true
    public var detectEmails: Bool = true
    public var detectPasswords: Bool = true
    public var detectSSN: Bool = false
    public var redactionMode: String = "blackout"
    public var customPatterns: [String] = []
    public var excludedApps: [String] = []
    
    enum CodingKeys: String, CodingKey {
        case detectCreditCards = "detect_credit_cards"
        case detectEmails = "detect_emails"
        case detectPasswords = "detect_passwords"
        case detectSSN = "detect_ssn"
        case redactionMode = "redaction_mode"
        case customPatterns = "custom_patterns"
        case excludedApps = "excluded_apps"
    }
}
