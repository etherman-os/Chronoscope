use crate::PrivacyConfig;
use once_cell::sync::Lazy;
use regex::Regex;

static EMAIL_RE: Lazy<Regex> =
    Lazy::new(|| Regex::new(r"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}").unwrap());

static CC_RE: Lazy<Regex> = Lazy::new(|| {
    Regex::new(r"\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13})\b").unwrap()
});

static SSN_RE: Lazy<Regex> = Lazy::new(|| Regex::new(r"\b\d{3}-\d{2}-\d{4}\b").unwrap());

static PASS_RE: Lazy<Regex> = Lazy::new(|| Regex::new(r"(?i)password\s*[:=]\s*\S+").unwrap());

#[derive(Debug, Clone)]
pub struct Detection {
    pub start: usize,
    pub end: usize,
    pub detection_type: DetectionType,
    pub confidence: f32,
}

#[derive(Debug, Clone)]
pub enum DetectionType {
    CreditCard,
    Email,
    Password,
    SSN,
    Custom(String),
}

pub fn scan_frame(
    _frame: &[u8],
    _width: u32,
    _height: u32,
    _config: &PrivacyConfig,
) -> Vec<Detection> {
    // Frame-based OCR detection is not implemented in MVP.
    vec![]
}

pub fn scan_text(text: &str, config: &PrivacyConfig) -> Vec<Detection> {
    let mut detections = Vec::new();

    if config.detect_emails {
        for mat in EMAIL_RE.find_iter(text) {
            detections.push(Detection {
                start: mat.start(),
                end: mat.end(),
                detection_type: DetectionType::Email,
                confidence: 0.95,
            });
        }
    }

    if config.detect_credit_cards {
        for mat in CC_RE.find_iter(text) {
            detections.push(Detection {
                start: mat.start(),
                end: mat.end(),
                detection_type: DetectionType::CreditCard,
                confidence: 0.98,
            });
        }
    }

    if config.detect_ssn {
        for mat in SSN_RE.find_iter(text) {
            detections.push(Detection {
                start: mat.start(),
                end: mat.end(),
                detection_type: DetectionType::SSN,
                confidence: 0.99,
            });
        }
    }

    if config.detect_passwords {
        for mat in PASS_RE.find_iter(text) {
            detections.push(Detection {
                start: mat.start(),
                end: mat.end(),
                detection_type: DetectionType::Password,
                confidence: 0.85,
            });
        }
    }

    for pattern in &config.custom_patterns {
        if pattern.len() > 10_000 {
            continue;
        }
        let re = match regex::RegexBuilder::new(pattern)
            .size_limit(1 << 20)
            .dfa_size_limit(1 << 20)
            .build()
        {
            Ok(r) => r,
            Err(_) => continue,
        };
        for mat in re.find_iter(text) {
            detections.push(Detection {
                start: mat.start(),
                end: mat.end(),
                detection_type: DetectionType::Custom(pattern.clone()),
                confidence: 0.90,
            });
        }
    }

    detections.sort_by_key(|d| d.start);
    detections
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_scan_text_email() {
        let config = PrivacyConfig {
            detect_credit_cards: false,
            detect_emails: true,
            detect_passwords: false,
            detect_ssn: false,
            redaction_mode: crate::RedactionMode::Blur,
            custom_patterns: vec![],
            excluded_apps: vec![],
        };
        let detections = scan_text("Contact me at user@example.com", &config);
        assert_eq!(detections.len(), 1);
        assert!(matches!(detections[0].detection_type, DetectionType::Email));
    }

    #[test]
    fn test_scan_text_credit_card() {
        let config = PrivacyConfig {
            detect_credit_cards: true,
            detect_emails: false,
            detect_passwords: false,
            detect_ssn: false,
            redaction_mode: crate::RedactionMode::Blur,
            custom_patterns: vec![],
            excluded_apps: vec![],
        };
        let detections = scan_text("My card is 4111111111111111", &config);
        assert_eq!(detections.len(), 1);
        assert!(matches!(
            detections[0].detection_type,
            DetectionType::CreditCard
        ));
    }

    #[test]
    fn test_scan_text_custom_pattern() {
        let config = PrivacyConfig {
            detect_credit_cards: false,
            detect_emails: false,
            detect_passwords: false,
            detect_ssn: false,
            redaction_mode: crate::RedactionMode::Blur,
            custom_patterns: vec![r"\bABC-\d{4}\b".to_string()],
            excluded_apps: vec![],
        };
        let detections = scan_text("Token: ABC-1234", &config);
        assert_eq!(detections.len(), 1);
        assert!(matches!(
            detections[0].detection_type,
            DetectionType::Custom(_)
        ));
    }

    #[test]
    fn test_custom_pattern_rejected_if_too_long() {
        let config = PrivacyConfig {
            detect_credit_cards: false,
            detect_emails: false,
            detect_passwords: false,
            detect_ssn: false,
            redaction_mode: crate::RedactionMode::Blur,
            custom_patterns: vec!["a".repeat(10_001)],
            excluded_apps: vec![],
        };
        let detections = scan_text("abc", &config);
        assert!(detections.is_empty());
    }
}
