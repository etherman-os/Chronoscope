//! # Chronoscope Linux SDK (Experimental)
//!
//! This SDK is experimental and not recommended for production use.
//! X11 capture is functional but limited. Wayland support is preliminary.

pub mod buffer;
pub mod capture;
pub mod config;
pub mod input;
pub mod upload;

use anyhow::Result;
use std::sync::Arc;
use std::time::Instant;
use tokio::task::JoinHandle;
use tokio_util::sync::CancellationToken;

pub use config::{CaptureConfig, CaptureMode, CaptureQuality};
pub use upload::SessionEvent;

pub struct LinuxCapture {
    config: CaptureConfig,
    uploader: Option<Arc<upload::ChunkUploader>>,
    _privacy: chronoscope_privacy::PrivacyEngine,
    cancel_token: CancellationToken,
    tasks: Vec<JoinHandle<Result<()>>>,
    session_id: Option<String>,
    started_at: Option<Instant>,
}

impl LinuxCapture {
    pub fn new(config: CaptureConfig) -> Result<Self> {
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
            uploader: None,
            _privacy: privacy,
            cancel_token,
            tasks: Vec::new(),
            session_id: None,
            started_at: None,
        })
    }

    pub async fn start(&mut self) -> Result<()> {
        if self.uploader.is_some() {
            return Ok(());
        }

        let mut uploader = upload::ChunkUploader::new(&self.config)?;
        let session_id = uploader
            .initialize_session(&self.config.user_id, &self.config.capture_mode)
            .await?;
        let uploader = Arc::new(uploader);
        self.session_id = Some(session_id);
        self.uploader = Some(uploader.clone());
        self.started_at = Some(Instant::now());

        let display_server = detect_display_server()?;
        if self.config.capture_mode != CaptureMode::Events {
            let (frame_tx, frame_rx) = tokio::sync::mpsc::channel::<Vec<u8>>(16);
            self.spawn_upload_loop(uploader.clone(), frame_rx);

            match display_server {
                DisplayServer::Wayland => self.spawn_wayland(frame_tx),
                DisplayServer::X11 => self.spawn_x11(frame_tx),
            }
        }

        if self.config.capture_mode != CaptureMode::Video {
            self.spawn_input_capture(uploader);
        }

        Ok(())
    }

    pub async fn stop(&mut self) -> Result<()> {
        self.cancel_token.cancel();
        while let Some(task) = self.tasks.pop() {
            match task.await {
                Ok(Ok(())) => {}
                Ok(Err(err)) => tracing::warn!("Chronoscope task stopped with error: {}", err),
                Err(err) => tracing::warn!("Chronoscope task join error: {}", err),
            }
        }

        if let Some(uploader) = &self.uploader {
            let duration = self
                .started_at
                .map(|started_at| started_at.elapsed().as_millis());
            uploader.finalize_with_duration(duration).await?;
        }

        self.uploader = None;
        self.session_id = None;
        self.started_at = None;
        self.cancel_token = CancellationToken::new();
        Ok(())
    }

    pub fn session_id(&self) -> Option<&str> {
        self.session_id.as_deref()
    }

    fn spawn_x11(&mut self, frame_tx: tokio::sync::mpsc::Sender<Vec<u8>>) {
        let frame_rate = self.config.frame_rate;
        let token = self.cancel_token.child_token();
        self.tasks.push(tokio::spawn(async move {
            capture::x11::start_capture(frame_tx, frame_rate, token).await
        }));
    }

    fn spawn_wayland(&mut self, frame_tx: tokio::sync::mpsc::Sender<Vec<u8>>) {
        let frame_rate = self.config.frame_rate;
        let token = self.cancel_token.child_token();
        self.tasks.push(tokio::spawn(async move {
            capture::wayland::start_capture(frame_tx, frame_rate, token).await
        }));
    }

    fn spawn_upload_loop(
        &mut self,
        uploader: Arc<upload::ChunkUploader>,
        mut frame_rx: tokio::sync::mpsc::Receiver<Vec<u8>>,
    ) {
        self.tasks.push(tokio::spawn(async move {
            let mut chunk_index = 0u32;
            while let Some(frame) = frame_rx.recv().await {
                uploader.upload_chunk(frame, chunk_index).await?;
                chunk_index = chunk_index.saturating_add(1);
            }
            Ok(())
        }));
    }

    fn spawn_input_capture(&mut self, uploader: Arc<upload::ChunkUploader>) {
        let token = self.cancel_token.child_token();
        let started_at = Instant::now();
        self.tasks.push(tokio::spawn(async move {
            input::start_input_capture(uploader, started_at, token).await
        }));
    }
}

#[derive(Debug, Clone)]
pub enum DisplayServer {
    Wayland,
    X11,
}

pub fn detect_display_server() -> Result<DisplayServer> {
    if std::env::var("DISPLAY").is_ok() {
        Ok(DisplayServer::X11)
    } else if std::env::var("WAYLAND_DISPLAY").is_ok() {
        Ok(DisplayServer::Wayland)
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

        std::env::remove_var("WAYLAND_DISPLAY");
        std::env::set_var("DISPLAY", ":0");
        assert!(matches!(
            detect_display_server().unwrap(),
            DisplayServer::X11
        ));

        std::env::set_var("WAYLAND_DISPLAY", "wayland-1");
        assert!(matches!(
            detect_display_server().unwrap(),
            DisplayServer::X11
        ));

        std::env::remove_var("DISPLAY");
        assert!(matches!(
            detect_display_server().unwrap(),
            DisplayServer::Wayland
        ));

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
