use chronoscope_privacy::{PrivacyEngine, PrivacyConfig, RedactionMode};

#[test]
fn test_detect_email() {
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
    let result = engine.process_text("Contact me at user@example.com");
    assert!(!result.contains("user@example.com"));
    assert!(result.contains("[REDACTED]") || result.contains("█"));
}

#[test]
fn test_detect_credit_card() {
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
    let result = engine.process_text("My card is 4111111111111111");
    assert!(!result.contains("4111111111111111"));
    assert!(result.contains("█") || result.contains("[REDACTED]"));
}
