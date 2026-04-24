pub mod detector;
pub mod redaction;
pub mod consent;
pub mod audit;
pub mod ffi;

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PrivacyConfig {
    pub detect_credit_cards: bool,
    pub detect_emails: bool,
    pub detect_passwords: bool,
    pub detect_ssn: bool,
    pub redaction_mode: RedactionMode,
    pub custom_patterns: Vec<String>,
    pub excluded_apps: Vec<String>,
}

impl Default for PrivacyConfig {
    fn default() -> Self {
        Self {
            detect_credit_cards: true,
            detect_emails: true,
            detect_passwords: true,
            detect_ssn: true,
            redaction_mode: RedactionMode::Blur,
            custom_patterns: Vec::new(),
            excluded_apps: Vec::new(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RedactionMode {
    Blur,
    Blackout,
    Replace(String),
}

impl Default for RedactionMode {
    fn default() -> Self {
        RedactionMode::Blur
    }
}

pub struct PrivacyEngine {
    config: PrivacyConfig,
    audit_log: audit::AuditLogger,
}

impl PrivacyEngine {
    pub fn new(config: PrivacyConfig) -> Self {
        Self {
            config,
            audit_log: audit::AuditLogger::new(),
        }
    }

    pub fn process_frame(&mut self, frame: &mut [u8], width: u32, height: u32) {
        let detections = detector::scan_frame(frame, width, height, &self.config);
        for detection in &detections {
            redaction::apply(frame, width, height, detection, &self.config.redaction_mode);
            self.audit_log.log_redaction(detection);
        }
    }

    pub fn process_text(&mut self, text: &str) -> String {
        let detections = detector::scan_text(text, &self.config);
        let mut result = text.to_string();
        for detection in detections.iter().rev() {
            let replacement = match &self.config.redaction_mode {
                RedactionMode::Blur => "[REDACTED]".to_string(),
                RedactionMode::Blackout => "█".repeat(detection.end - detection.start),
                RedactionMode::Replace(s) => s.clone(),
            };
            result.replace_range(detection.start..detection.end, &replacement);
            self.audit_log.log_redaction(detection);
        }
        result
    }

    pub fn check_consent(&self, user_id: &str) -> ConsentStatus {
        consent::get_status(user_id)
    }
}

#[derive(Debug, Clone)]
pub enum ConsentStatus {
    Granted,
    Denied,
    Pending,
}
