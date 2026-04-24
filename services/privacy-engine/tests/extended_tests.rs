use chronoscope_privacy::{PrivacyEngine, PrivacyConfig, RedactionMode};
use std::ffi::CString;
use std::ptr;

#[test]
fn test_detect_credit_card() {
    let config = PrivacyConfig {
        detect_credit_cards: true,
        detect_emails: true,
        detect_passwords: true,
        detect_ssn: true,
        redaction_mode: RedactionMode::Blur,
        custom_patterns: vec![],
        excluded_apps: vec![],
    };
    let mut engine = PrivacyEngine::new(config);
    let result = engine.process_text("My card is 4111111111111111");
    assert!(!result.contains("4111111111111111"));
    assert!(result.contains("[REDACTED]"));
}

#[test]
fn test_detect_email() {
    let config = PrivacyConfig {
        detect_credit_cards: true,
        detect_emails: true,
        detect_passwords: true,
        detect_ssn: true,
        redaction_mode: RedactionMode::Blackout,
        custom_patterns: vec![],
        excluded_apps: vec![],
    };
    let mut engine = PrivacyEngine::new(config);
    let result = engine.process_text("Contact me at user@example.com today");
    assert!(!result.contains("user@example.com"));
    assert!(result.contains("█") || result.contains("[REDACTED]"));
}

#[test]
fn test_detect_password() {
    let config = PrivacyConfig {
        detect_credit_cards: true,
        detect_emails: true,
        detect_passwords: true,
        detect_ssn: true,
        redaction_mode: RedactionMode::Replace("[HIDDEN]".to_string()),
        custom_patterns: vec![],
        excluded_apps: vec![],
    };
    let mut engine = PrivacyEngine::new(config);
    let result = engine.process_text("password: secret123 or Password = hunter2");
    assert!(!result.contains("secret123"));
    assert!(!result.contains("hunter2"));
    assert!(result.contains("[HIDDEN]"));
}

#[test]
fn test_detect_ssn() {
    let config = PrivacyConfig {
        detect_credit_cards: true,
        detect_emails: true,
        detect_passwords: true,
        detect_ssn: true,
        redaction_mode: RedactionMode::Blur,
        custom_patterns: vec![],
        excluded_apps: vec![],
    };
    let mut engine = PrivacyEngine::new(config);
    let result = engine.process_text("SSN: 123-45-6789");
    assert!(!result.contains("123-45-6789"));
    assert!(result.contains("[REDACTED]"));
}

#[test]
fn test_ffi_null_safety() {
    // chronoscope_privacy_init with null should return null
    let engine = chronoscope_privacy::ffi::chronoscope_privacy_init(ptr::null());
    assert!(engine.is_null());

    // chronoscope_privacy_process_text with null engine should return null
    let text = CString::new("hello").unwrap();
    let result = chronoscope_privacy::ffi::chronoscope_privacy_process_text(ptr::null_mut(), text.as_ptr());
    assert!(result.is_null());

    // chronoscope_privacy_process_text with null text should return null
    let config_json = CString::new(r#"{"detect_credit_cards":true,"detect_emails":true,"detect_passwords":true,"detect_ssn":true,"redaction_mode":"Blur","custom_patterns":[],"excluded_apps":[]}"#).unwrap();
    let engine = chronoscope_privacy::ffi::chronoscope_privacy_init(config_json.as_ptr());
    assert!(!engine.is_null());
    let result = chronoscope_privacy::ffi::chronoscope_privacy_process_text(engine, ptr::null());
    assert!(result.is_null());

    // chronoscope_privacy_process_frame with null engine should not panic
    let mut frame = vec![0u8; 16 * 16 * 4];
    chronoscope_privacy::ffi::chronoscope_privacy_process_frame(
        ptr::null_mut(),
        frame.as_mut_ptr(),
        16,
        16,
        64,
    );

    // chronoscope_privacy_process_frame with null frame should not panic
    chronoscope_privacy::ffi::chronoscope_privacy_process_frame(
        engine,
        ptr::null_mut(),
        16,
        16,
        64,
    );

    // Clean up
    chronoscope_privacy::ffi::chronoscope_privacy_free(engine);
}

#[test]
fn test_regex_bounded() {
    // Test that regex patterns don't cause catastrophic backtracking on long inputs.
    let config = PrivacyConfig {
        detect_credit_cards: true,
        detect_emails: true,
        detect_passwords: true,
        detect_ssn: true,
        redaction_mode: RedactionMode::Blur,
        custom_patterns: vec![],
        excluded_apps: vec![],
    };
    let mut engine = PrivacyEngine::new(config);

    // Very long string without any sensitive data should complete quickly.
    let long_input = "a".repeat(100_000);
    let result = engine.process_text(&long_input);
    assert_eq!(result.len(), long_input.len());

    // Long string with many @ symbols but no valid emails.
    let mut noisy = String::new();
    for _ in 0..10_000 {
        noisy.push_str("foo@bar ");
    }
    let result = engine.process_text(&noisy);
    // No valid TLD, so no emails should match.
    assert_eq!(result, noisy);
}

#[test]
fn test_custom_patterns() {
    let config = PrivacyConfig {
        detect_credit_cards: false,
        detect_emails: false,
        detect_passwords: false,
        detect_ssn: false,
        redaction_mode: RedactionMode::Replace("[CUSTOM]".to_string()),
        custom_patterns: vec![r"\bABC-\d{4}\b".to_string()],
        excluded_apps: vec![],
    };
    let mut engine = PrivacyEngine::new(config);
    let result = engine.process_text("Token: ABC-1234 and ABC-9999");
    assert!(!result.contains("ABC-1234"));
    assert!(!result.contains("ABC-9999"));
    assert!(result.contains("[CUSTOM]"));
}

#[test]
fn test_overlapping_detections() {
    let config = PrivacyConfig {
        detect_credit_cards: true,
        detect_emails: true,
        detect_passwords: true,
        detect_ssn: true,
        redaction_mode: RedactionMode::Blur,
        custom_patterns: vec![],
        excluded_apps: vec![],
    };
    let mut engine = PrivacyEngine::new(config);
    // Email that also looks like it contains numbers (no real overlap in this case,
    // but tests the merge logic for overlapping ranges).
    let result = engine.process_text("Reach me at user@example.com");
    assert!(!result.contains("user@example.com"));
}
