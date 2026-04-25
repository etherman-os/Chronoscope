use crate::config::Config;
use anyhow::Result;
use std::path::PathBuf;

pub async fn download_chunks(
    config: &Config,
    session_id: &str,
) -> Result<(tempfile::TempDir, Vec<PathBuf>)> {
    let prefix = format!("{}/", session_id);
    let temp_dir = tempfile::tempdir()?;
    let mut paths = Vec::new();
    let mut continuation_token = None::<String>;

    loop {
        let mut req = config
            .s3_client
            .list_objects_v2()
            .bucket(&config.bucket_name)
            .prefix(&prefix);

        if let Some(ref token) = continuation_token {
            req = req.continuation_token(token);
        }

        let list_resp = req.send().await?;
        let contents = list_resp.contents.unwrap_or_default();

        for object in contents {
            let key = object
                .key
                .ok_or_else(|| anyhow::anyhow!("missing object key"))?;
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

        if let Some(token) = list_resp.next_continuation_token {
            continuation_token = Some(token);
        } else {
            break;
        }
    }

    Ok((temp_dir, paths))
}
