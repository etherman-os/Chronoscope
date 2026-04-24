import Foundation

public final class PrivacyEngine {
    private var engine: OpaquePointer?
    
    public init(config: PrivacyConfig) {
        let json = try! JSONEncoder().encode(config)
        let jsonString = String(data: json, encoding: .utf8)!
        engine = jsonString.withCString { ptr in
            chronoscope_privacy_init(ptr)
        }
    }
    
    deinit {
        if let engine = engine {
            chronoscope_privacy_free(engine)
        }
    }
    
    public func processText(_ text: String) -> String {
        guard let engine = engine else { return text }
        let result = text.withCString { textPtr in
            chronoscope_privacy_process_text(engine, textPtr)
        }
        defer { chronoscope_privacy_free_string(result) }
        return String(cString: result!)
    }
}

public struct PrivacyConfig: Codable {
    public var detectCreditCards: Bool
    public var detectEmails: Bool
    public var detectPasswords: Bool
    public var detectSSN: Bool
    // ... etc
}

// C function declarations
@_silgen_name("chronoscope_privacy_init")
private func chronoscope_privacy_init(_ config: UnsafePointer<CChar>) -> OpaquePointer?

@_silgen_name("chronoscope_privacy_process_text")
private func chronoscope_privacy_process_text(_ engine: OpaquePointer?, _ text: UnsafePointer<CChar>) -> UnsafeMutablePointer<CChar>?

@_silgen_name("chronoscope_privacy_free_string")
private func chronoscope_privacy_free_string(_ s: UnsafeMutablePointer<CChar>?)

@_silgen_name("chronoscope_privacy_free")
private func chronoscope_privacy_free(_ engine: OpaquePointer?)
