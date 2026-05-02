#[derive(Debug, Clone)]
pub struct CaptureConfig {
    pub api_key: String,
    pub endpoint: String,
    pub user_id: String,
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

impl CaptureConfig {
    pub fn new(api_key: impl Into<String>, endpoint: impl Into<String>) -> Self {
        Self {
            api_key: api_key.into(),
            endpoint: endpoint.into(),
            user_id: "linux-user".to_string(),
            capture_mode: CaptureMode::Hybrid,
            quality: CaptureQuality::Medium,
            frame_rate: 10,
            buffer_size_mb: 100,
        }
    }

    pub fn with_user_id(mut self, user_id: impl Into<String>) -> Self {
        self.user_id = user_id.into();
        self
    }
}
