use chronoscope_processor::{config, db, deduplicator, downloader, encoder, indexer, queue, sync, uploader};
use tracing::{error, info};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();
    let config = config::Config::from_env().await?;
    info!("Chronoscope Processor starting...");

    let (tx, mut rx) = tokio::sync::mpsc::channel::<String>(100);

    // Spawn queue listener (Redis)
    tokio::spawn(queue::queue_listener(config.clone(), tx));

    // Process sessions with graceful shutdown
    let mut shutdown = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())?;
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
            _ = shutdown.recv() => {
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

    info!("Session {} processed successfully", session_id);
    Ok(())
}
