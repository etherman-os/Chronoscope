pub mod buffer;
pub mod capture;
pub mod config;
pub mod input;
pub mod upload;

use anyhow::Result;
use std::sync::Arc;
use tokio::sync::Mutex;
use tokio_util::sync::CancellationToken;

pub use config::{CaptureConfig, CaptureMode, CaptureQuality};

pub struct LinuxCapture {
    config: CaptureConfig,
    buffer: Arc<Mutex<buffer::CircularBuffer>>,
    _uploader: upload::ChunkUploader,
    _privacy: chronoscope_privacy::PrivacyEngine,
    cancel_token: CancellationToken,
}

impl LinuxCapture {
    pub fn new(config: CaptureConfig) -> Result<Self> {
        let buffer = Arc::new(Mutex::new(buffer::CircularBuffer::new(
            config.buffer_size_mb * 1024 * 1024,
        )));
        let uploader = upload::ChunkUploader::new(&config)?;
        let privacy_config = chronoscope_privacy::PrivacyConfig {
            detect_credit_cards: true,
            detect_emails: true,
            detect_passwords: true,
            detect_ssn: false,
            redaction_mode: chronoscope_privacy::RedactionMode::Blackout,
            custom_patterns: vec![],
            excluded_apps: vec![],
        };
        let privacy = chronoscope_privacy::PrivacyEngine::new(privacy_config);
        let cancel_token = CancellationToken::new();
        Ok(Self {
            config,
            buffer,
            _uploader: uploader,
            _privacy: privacy,
            cancel_token,
        })
    }

    pub async fn start(&mut self) -> Result<()> {
        let display_server = detect_display_server()?;
        match display_server {
            DisplayServer::Wayland => self.start_wayland().await,
            DisplayServer::X11 => self.start_x11().await,
        }
    }

    pub async fn stop(&mut self) -> Result<()> {
        self.cancel_token.cancel();
        Ok(())
    }

    async fn start_wayland(&mut self) -> Result<()> {
        capture::wayland::start_capture(
            self.buffer.clone(),
            self.config.frame_rate,
            self.cancel_token.child_token(),
        )
        .await
    }

    async fn start_x11(&mut self) -> Result<()> {
        capture::x11::start_capture(
            self.buffer.clone(),
            self.config.frame_rate,
            self.cancel_token.child_token(),
        )
        .await
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
        Err(anyhow::anyhow!(
            "No display server detected. Set WAYLAND_DISPLAY or DISPLAY."
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_detect_display_server() {
        let orig_wayland = std::env::var("WAYLAND_DISPLAY").ok();
        let orig_display = std::env::var("DISPLAY").ok();

        std::env::set_var("WAYLAND_DISPLAY", "wayland-1");
        std::env::remove_var("DISPLAY");
        assert!(matches!(detect_display_server().unwrap(), DisplayServer::Wayland));

        std::env::remove_var("WAYLAND_DISPLAY");
        std::env::set_var("DISPLAY", ":0");
        assert!(matches!(detect_display_server().unwrap(), DisplayServer::X11));

        std::env::remove_var("WAYLAND_DISPLAY");
        std::env::remove_var("DISPLAY");
        assert!(detect_display_server().is_err());

        if let Some(v) = orig_wayland {
            std::env::set_var("WAYLAND_DISPLAY", v);
        } else {
            std::env::remove_var("WAYLAND_DISPLAY");
        }
        if let Some(v) = orig_display {
            std::env::set_var("DISPLAY", v);
        } else {
            std::env::remove_var("DISPLAY");
        }
    }

    #[test]
    fn test_linux_capture_new_and_stop() {
        let config = crate::config::CaptureConfig::new("key", "http://localhost:8080");
        let mut cap = LinuxCapture::new(config).unwrap();
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            cap.stop().await.unwrap();
        });
    }
}
