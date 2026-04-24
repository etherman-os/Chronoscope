use reqwest::multipart;
use uuid::Uuid;

pub struct ChunkUploader {
    client: reqwest::Client,
    endpoint: String,
    api_key: String,
    session_id: String,
}

impl ChunkUploader {
    pub fn new(config: &super::CaptureConfig) -> anyhow::Result<Self> {
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()?;
        let session_id = Uuid::new_v4().to_string();
        Ok(Self {
            client,
            endpoint: config.endpoint.clone(),
            api_key: config.api_key.clone(),
            session_id,
        })
    }

    pub async fn upload_chunk(&self, data: Vec<u8>, index: u32) -> anyhow::Result<()> {
        let url = format!("{}/v1/sessions/{}/chunks", self.endpoint, self.session_id);

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
            anyhow::bail!(
                "Upload failed: {} - {}",
                response.status(),
                response.text().await.unwrap_or_default()
            );
        }

        tracing::info!("Uploaded chunk {} for session {}", index, self.session_id);
        Ok(())
    }

    pub async fn finalize(&self) -> anyhow::Result<()> {
        let url = format!("{}/v1/sessions/{}/complete", self.endpoint, self.session_id);

        let response = self
            .client
            .post(&url)
            .header("X-API-Key", &self.api_key)
            .send()
            .await?;

        if !response.status().is_success() {
            anyhow::bail!(
                "Finalize failed: {} - {}",
                response.status(),
                response.text().await.unwrap_or_default()
            );
        }

        tracing::info!("Finalized session {}", self.session_id);
        Ok(())
    }
}
