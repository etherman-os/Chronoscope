use chronoscope_processor::{
    config, db, deduplicator, downloader, encoder, indexer, queue, sync, uploader,
};
use tracing::{error, info};

#[cfg(unix)]
async fn wait_sigterm() {
    let mut sig = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
        .expect("failed to install SIGTERM handler");
    sig.recv().await;
}

#[cfg(not(unix))]
async fn wait_sigterm() {
    std::future::pending::<()>().await;
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();
    let config = config::Config::from_env().await?;
    info!("Chronoscope Processor starting...");

    let (tx, mut rx) = tokio::sync::mpsc::channel::<String>(100);

    tokio::spawn(queue::queue_listener(config.clone(), tx));

    loop {
        tokio::select! {
            Some(session_id) = rx.recv() => {
                if let Err(e) = process_session(&config, &session_id).await {
                    error!("Failed to process session {}: {}", session_id, e);
                }
            }
            _ = tokio::signal::ctrl_c() => {
                info!("Received Ctrl+C, shutting down...");
                break;
            }
            _ = wait_sigterm() => {
                info!("Received SIGTERM, shutting down...");
                break;
            }
        }
    }

    Ok(())
}

async fn process_session(config: &config::Config, session_id: &str) -> anyhow::Result<()> {
    info!("Processing session: {}", session_id);

    // 1. Download chunks from MinIO/S3
    let (_temp_dir, chunks) = downloader::download_chunks(config, session_id).await?;

    // 2. Deduplicate frames using perceptual hash
    let unique_frames = deduplicator::deduplicate(chunks).await?;

    // 3. Encode to H.264 MP4 using FFmpeg
    let video_path = encoder::encode_h264(config, session_id, unique_frames).await?;

    // 4. Sync events with timeline
    let timeline = sync::synchronize_events(config, session_id, &video_path).await?;

    // 5. Generate keyframe index
    let index = indexer::generate_index(&video_path, &timeline).await?;

    // 6. Upload processed video to MinIO/S3
    uploader::upload_video(config, session_id, &video_path).await?;

    // 7. Update DB: status = 'ready', video_path, metadata
    db::update_session_status(config, session_id, "ready", &index).await?;

    // 8. Clean up temporary encoded video file
    if let Err(e) = tokio::fs::remove_file(&video_path).await {
        tracing::warn!(
            "Failed to remove temp video file {}: {}",
            video_path.display(),
            e
        );
    }

    info!("Session {} processed successfully", session_id);
    Ok(())
}
