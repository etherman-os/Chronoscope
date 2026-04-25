use crate::config::Config;
use crate::indexer::VideoIndex;
use anyhow::{Context, Result};

pub async fn update_session_status(
    config: &Config,
    session_id: &str,
    status: &str,
    index: &VideoIndex,
) -> Result<()> {
    let session_uuid = uuid::Uuid::parse_str(session_id)
        .with_context(|| format!("session_id '{}' is not a valid UUID", session_id))?;

    let client = config.db_pool.get().await?;
    let metadata = serde_json::to_value(index)?;
    let video_path = format!(
        "s3://{}/{}/session.mp4",
        config.processed_bucket_name, session_id
    );

    client
        .execute(
            "UPDATE sessions SET status = $1, processed_at = NOW(), video_path = $2, metadata = metadata || $3 WHERE id = $4::uuid",
            &[&status, &video_path, &metadata, &session_uuid],
        )
        .await?;

    Ok(())
}

#[cfg(test)]
mod tests {
    #[test]
    fn test_uuid_validation() {
        let valid = "550e8400-e29b-41d4-a716-446655440000";
        let invalid = "not-a-uuid";

        assert!(uuid::Uuid::parse_str(valid).is_ok());
        assert!(uuid::Uuid::parse_str(invalid).is_err());
    }
}
