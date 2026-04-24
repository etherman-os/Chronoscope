use crate::config::Config;
use anyhow::Result;
use std::path::PathBuf;

pub async fn download_chunks(config: &Config, session_id: &str) -> Result<Vec<PathBuf>> {
    let prefix = format!("{}/", session_id);

    let list_resp = config
        .s3_client
        .list_objects_v2()
        .bucket(&config.bucket_name)
        .prefix(&prefix)
        .send()
        .await?;

    let contents = list_resp.contents.unwrap_or_default();
    let temp_dir = tempfile::tempdir()?;
    let mut paths = Vec::with_capacity(contents.len());

    for object in contents {
        let key = object.key.ok_or_else(|| anyhow::anyhow!("missing object key"))?;
        let get_resp = config
            .s3_client
            .get_object()
            .bucket(&config.bucket_name)
            .key(&key)
            .send()
            .await?;

        let data = get_resp.body.collect().await?;
        let file_name = key.rsplit('/').next().unwrap_or(&key);
        let file_path = temp_dir.path().join(file_name);
        tokio::fs::write(&file_path, data.into_bytes()).await?;
        paths.push(file_path);
    }

    // Keep temp_dir alive by leaking it (simpler for this pipeline)
    // In production you'd manage the TempDir lifecycle more carefully
    let _ = Box::leak(Box::new(temp_dir));

    Ok(paths)
}
