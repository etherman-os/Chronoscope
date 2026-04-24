use crate::config::Config;
use crate::indexer::VideoIndex;
use anyhow::Result;

pub async fn update_session_status(
    config: &Config,
    session_id: &str,
    status: &str,
    index: &VideoIndex,
) -> Result<()> {
    let client = config.db_pool.get().await?;
    let metadata = serde_json::to_value(index)?;
    let video_path = format!("s3://{}/{}/session.mp4", config.processed_bucket_name, session_id);

    client
        .execute(
            "UPDATE sessions SET status = $1, processed_at = NOW(), video_path = $2, metadata = metadata || $3 WHERE id = $4::uuid",
            &[&status, &video_path, &metadata, &session_id],
        )
        .await?;

    Ok(())
}
