use regex::Regex;
use crate::PrivacyConfig;

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

pub fn scan_frame(_frame: &[u8], _width: u32, _height: u32, _config: &PrivacyConfig) -> Vec<Detection> {
    // Frame-based OCR detection is not implemented in MVP.
    vec![]
}

pub fn scan_text(text: &str, config: &PrivacyConfig) -> Vec<Detection> {
    let mut detections = Vec::new();

    if config.detect_emails {
        let email_re = Regex::new(r"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}").unwrap();
        for mat in email_re.find_iter(text) {
            detections.push(Detection {
                start: mat.start(),
                end: mat.end(),
                detection_type: DetectionType::Email,
                confidence: 0.95,
            });
        }
    }

    if config.detect_credit_cards {
        let cc_re = Regex::new(r"\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13})\b").unwrap();
        for mat in cc_re.find_iter(text) {
            detections.push(Detection {
                start: mat.start(),
                end: mat.end(),
                detection_type: DetectionType::CreditCard,
                confidence: 0.98,
            });
        }
    }

    if config.detect_ssn {
        let ssn_re = Regex::new(r"\b\d{3}-\d{2}-\d{4}\b").unwrap();
        for mat in ssn_re.find_iter(text) {
            detections.push(Detection {
                start: mat.start(),
                end: mat.end(),
                detection_type: DetectionType::SSN,
                confidence: 0.99,
            });
        }
    }

    if config.detect_passwords {
        // Heuristic: look for "password" or "passwd" followed by separator and value
        let pass_re = Regex::new(r"(?i)password\s*[:=]\s*\S+").unwrap();
        for mat in pass_re.find_iter(text) {
            detections.push(Detection {
                start: mat.start(),
                end: mat.end(),
                detection_type: DetectionType::Password,
                confidence: 0.85,
            });
        }
    }

    for pattern in &config.custom_patterns {
        if let Ok(re) = Regex::new(pattern) {
            for mat in re.find_iter(text) {
                detections.push(Detection {
                    start: mat.start(),
                    end: mat.end(),
                    detection_type: DetectionType::Custom(pattern.clone()),
                    confidence: 0.90,
                });
            }
        }
    }

    detections.sort_by_key(|d| d.start);
    detections
}
