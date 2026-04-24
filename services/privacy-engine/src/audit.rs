use crate::detector::Detection;

pub struct AuditLogger;

impl AuditLogger {
    pub fn new() -> Self {
        Self
    }

    pub fn log_redaction(&self, detection: &Detection) {
        // In MVP: print to stderr. Later: write to file/queue.
        eprintln!(
            "[AUDIT] Redacted {:?} at {}-{}",
            detection.detection_type, detection.start, detection.end
        );
    }
}
