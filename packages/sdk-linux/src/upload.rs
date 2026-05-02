use reqwest::multipart;
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::time::Duration;

#[derive(Clone)]
pub struct ChunkUploader {
    client: reqwest::Client,
    endpoint: String,
    api_key: String,
    session_id: Option<String>,
}

impl ChunkUploader {
    pub fn new(config: &super::CaptureConfig) -> anyhow::Result<Self> {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(30))
            .build()?;
        Ok(Self {
            client,
            endpoint: config.endpoint.trim_end_matches('/').to_string(),
            api_key: config.api_key.clone(),
            session_id: None,
        })
    }

    #[cfg(test)]
    pub fn session_id(&self) -> Option<&str> {
        self.session_id.as_deref()
    }

    pub async fn initialize_session(
        &mut self,
        user_id: &str,
        capture_mode: &super::CaptureMode,
    ) -> anyhow::Result<String> {
        let url = self.api_url("sessions/init");
        let response = self
            .client
            .post(url)
            .header("X-API-Key", &self.api_key)
            .json(&json!({
                "user_id": user_id,
                "capture_mode": capture_mode.as_api_value(),
                "metadata": {
                    "sdk": "chronoscope-sdk-linux",
                    "display_server": std::env::var("XDG_SESSION_TYPE").unwrap_or_else(|_| "unknown".to_string())
                }
            }))
            .send()
            .await?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            anyhow::bail!("Session init failed: {} - {}", status, body);
        }

        let body: InitSessionResponse = response.json().await?;
        self.session_id = Some(body.session_id.clone());
        tracing::info!("Initialized Chronoscope session {}", body.session_id);
        Ok(body.session_id)
    }

    pub async fn upload_chunk(&self, data: Vec<u8>, index: u32) -> anyhow::Result<()> {
        let session_id = self.require_session_id()?;
        let url = self.api_url(&format!("sessions/{}/chunks", session_id));

        let part = multipart::Part::bytes(data)
            .file_name(format!("chunk_{:04}.jpg", index))
            .mime_str("image/jpeg")?;

        let form = multipart::Form::new().part("chunk", part);

        let response = self
            .client
            .post(&url)
            .header("X-API-Key", &self.api_key)
            .header("X-Chunk-Index", index.to_string())
            .multipart(form)
            .send()
            .await?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_else(|_| String::new());
            anyhow::bail!("Upload failed: {} - {}", status, body);
        }

        tracing::info!("Uploaded chunk {} for session {}", index, session_id);
        Ok(())
    }

    pub async fn upload_events(&self, events: Vec<SessionEvent>) -> anyhow::Result<()> {
        if events.is_empty() {
            return Ok(());
        }

        let session_id = self.require_session_id()?;
        let url = self.api_url(&format!("sessions/{}/events", session_id));
        let response = self
            .client
            .post(url)
            .header("X-API-Key", &self.api_key)
            .json(&EventBatch { events })
            .send()
            .await?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            anyhow::bail!("Event upload failed: {} - {}", status, body);
        }

        Ok(())
    }

    pub async fn finalize(&self) -> anyhow::Result<()> {
        self.finalize_with_duration(None).await
    }

    pub async fn finalize_with_duration(&self, duration_ms: Option<u128>) -> anyhow::Result<()> {
        let session_id = self.require_session_id()?;
        let url = self.api_url(&format!("sessions/{}/complete", session_id));
        let mut request = self.client.post(url).header("X-API-Key", &self.api_key);

        if let Some(duration_ms) = duration_ms {
            request = request.json(&json!({
                "duration_ms": duration_ms.min(i32::MAX as u128) as i32
            }));
        }

        let response = request.send().await?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_else(|_| String::new());
            anyhow::bail!("Finalize failed: {} - {}", status, body);
        }

        tracing::info!("Finalized session {}", session_id);
        Ok(())
    }

    fn api_url(&self, path: &str) -> String {
        if self.endpoint.ends_with("/v1") {
            format!("{}/{}", self.endpoint, path.trim_start_matches('/'))
        } else {
            format!("{}/v1/{}", self.endpoint, path.trim_start_matches('/'))
        }
    }

    fn require_session_id(&self) -> anyhow::Result<&str> {
        self.session_id
            .as_deref()
            .ok_or_else(|| anyhow::anyhow!("session has not been initialized"))
    }
}

impl super::CaptureMode {
    fn as_api_value(self) -> &'static str {
        match self {
            super::CaptureMode::Video => "video",
            super::CaptureMode::Events => "events",
            super::CaptureMode::Hybrid => "hybrid",
        }
    }
}

#[derive(Deserialize)]
struct InitSessionResponse {
    session_id: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct SessionEvent {
    pub event_type: String,
    pub timestamp_ms: i32,
    pub x: i32,
    pub y: i32,
    pub target: String,
    pub payload: serde_json::Value,
}

#[derive(Serialize)]
struct EventBatch {
    events: Vec<SessionEvent>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::CaptureConfig;

    #[test]
    fn test_chunk_uploader_new() {
        let config = CaptureConfig::new("test_key", "http://localhost:8080");
        let uploader = ChunkUploader::new(&config).unwrap();
        assert!(uploader.session_id().is_none());
    }

    #[tokio::test]
    async fn test_upload_chunk_network_error() {
        let config = CaptureConfig::new("test_key", "http://localhost:1");
        let uploader = ChunkUploader::new(&config).unwrap();
        let result = uploader.upload_chunk(vec![0u8; 100], 0).await;
        assert!(result.is_err());
        assert!(result
            .unwrap_err()
            .to_string()
            .contains("not been initialized"));
    }

    #[tokio::test]
    async fn test_finalize_network_error() {
        let config = CaptureConfig::new("test_key", "http://localhost:1");
        let uploader = ChunkUploader::new(&config).unwrap();
        let result = uploader.finalize().await;
        assert!(result.is_err());
    }
}
