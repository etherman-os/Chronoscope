use anyhow::Result;
use img_hash::HasherConfig;
use std::path::PathBuf;

const SIMILARITY_THRESHOLD: u32 = 5; // Hamming distance

pub async fn deduplicate(chunk_paths: Vec<PathBuf>) -> Result<Vec<PathBuf>> {
    let unique = tokio::task::spawn_blocking(move || {
        let hasher = HasherConfig::new().to_hasher();
        let mut unique_paths: Vec<PathBuf> = Vec::new();
        let mut last_hash: Option<img_hash::ImageHash> = None;

        for path in chunk_paths {
            match image::open(&path) {
                Ok(img) => {
                    let hash = hasher.hash_image(&img);
                    let is_duplicate = if let Some(ref lh) = last_hash {
                        hash.dist(lh) < SIMILARITY_THRESHOLD
                    } else {
                        false
                    };

                    if !is_duplicate {
                        last_hash = Some(hash);
                        unique_paths.push(path);
                    }
                }
                Err(e) => {
                    tracing::warn!("Failed to open image {}: {}", path.display(), e);
                }
            }
        }

        unique_paths
    })
    .await?;

    Ok(unique)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_deduplicate_skips_invalid_files() {
        let temp_dir = tempfile::tempdir().unwrap();
        let bad_path = temp_dir.path().join("not_an_image.txt");
        std::fs::write(&bad_path, "hello").unwrap();

        let result = deduplicate(vec![bad_path]).await.unwrap();
        assert!(result.is_empty());
    }
}
