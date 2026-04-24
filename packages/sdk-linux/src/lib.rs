pub mod capture;
pub mod input;
pub mod buffer;
pub mod upload;
pub mod config;

use std::sync::Arc;
use tokio::sync::Mutex;
use anyhow::Result;

pub use config::{CaptureConfig, CaptureMode, CaptureQuality};

pub struct LinuxCapture {
    config: CaptureConfig,
    buffer: Arc<Mutex<buffer::CircularBuffer>>,
    uploader: upload::ChunkUploader,
    privacy: chronoscope_privacy_engine::PrivacyEngine,
}

impl LinuxCapture {
    pub fn new(config: CaptureConfig) -> Result<Self> {
        let buffer = Arc::new(Mutex::new(buffer::CircularBuffer::new(config.buffer_size_mb * 1024 * 1024)));
        let uploader = upload::ChunkUploader::new(&config)?;
        let privacy_config = chronoscope_privacy_engine::PrivacyConfig {
            detect_credit_cards: true,
            detect_emails: true,
            detect_passwords: true,
            detect_ssn: false,
            redaction_mode: chronoscope_privacy_engine::RedactionMode::Blackout,
            custom_patterns: vec![],
            excluded_apps: vec![],
        };
        let privacy = chronoscope_privacy_engine::PrivacyEngine::new(privacy_config);
        Ok(Self { config, buffer, uploader, privacy })
    }

    pub async fn start(&mut self) -> Result<()> {
        let display_server = detect_display_server()?;
        match display_server {
            DisplayServer::Wayland => self.start_wayland().await,
            DisplayServer::X11 => self.start_x11().await,
        }
    }

    pub async fn stop(&mut self) -> Result<()> {
        // Stop capture, flush buffer, finalize upload
        Ok(())
    }

    async fn start_wayland(&mut self) -> Result<()> {
        capture::wayland::start_capture(self.buffer.clone(), self.config.frame_rate).await
    }

    async fn start_x11(&mut self) -> Result<()> {
        capture::x11::start_capture(self.buffer.clone(), self.config.frame_rate).await
    }
}

#[derive(Debug, Clone)]
pub enum DisplayServer {
    Wayland,
    X11,
}

pub fn detect_display_server() -> Result<DisplayServer> {
    if std::env::var("WAYLAND_DISPLAY").is_ok() {
        Ok(DisplayServer::Wayland)
    } else if std::env::var("DISPLAY").is_ok() {
        Ok(DisplayServer::X11)
    } else {
        Err(anyhow::anyhow!("No display server detected. Set WAYLAND_DISPLAY or DISPLAY."))
    }
}
