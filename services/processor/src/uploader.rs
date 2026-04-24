use crate::config::Config;
use anyhow::Result;
use aws_sdk_s3::primitives::ByteStream;
use std::path::Path;

pub async fn upload_video(
    config: &Config,
    session_id: &str,
    video_path: &Path,
) -> Result<()> {
    let body = ByteStream::from_path(video_path).await?;
    let key = format!("{}/session.mp4", session_id);

    config
        .s3_client
        .put_object()
        .bucket(&config.processed_bucket_name)
        .key(&key)
        .content_type("video/mp4")
        .body(body)
        .send()
        .await?;

    Ok(())
}
