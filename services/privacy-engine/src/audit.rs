use crate::detector::Detection;

// TODO: Replace stderr logging with structured file/queue output.
#[allow(dead_code)]
pub struct AuditLogger;

impl Default for AuditLogger {
    fn default() -> Self {
        Self::new()
    }
}

impl AuditLogger {
    pub fn new() -> Self {
        Self
    }

    pub fn log_redaction(&self, detection: &Detection) {
        eprintln!(
            "[AUDIT] Redacted {:?} at {}-{}",
            detection.detection_type, detection.start, detection.end
        );
    }
}
