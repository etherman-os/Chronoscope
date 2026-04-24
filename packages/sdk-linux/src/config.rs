#[derive(Debug, Clone)]
pub struct CaptureConfig {
    pub api_key: String,
    pub endpoint: String,
    pub capture_mode: CaptureMode,
    pub quality: CaptureQuality,
    pub frame_rate: u32,
    pub buffer_size_mb: usize,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CaptureMode {
    Video,
    Events,
    Hybrid,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CaptureQuality {
    Low,
    Medium,
    High,
}

impl Default for CaptureConfig {
    fn default() -> Self {
        Self {
            api_key: String::new(),
            endpoint: "http://localhost:8080".to_string(),
            capture_mode: CaptureMode::Hybrid,
            quality: CaptureQuality::Medium,
            frame_rate: 10,
            buffer_size_mb: 100,
        }
    }
}
