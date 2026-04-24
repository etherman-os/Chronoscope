use crate::config::Config;
use anyhow::Result;
use tracing::{error, info};

pub async fn queue_listener(
    config: Config,
    tx: tokio::sync::mpsc::Sender<String>,
) -> Result<()> {
    let mut con = config.redis_client;
    loop {
        let result: Option<(String, String)> = redis::cmd("BRPOP")
            .arg("chronoscope:process_queue")
            .arg(0)
            .query_async(&mut con)
            .await?;

        if let Some((_, session_id)) = result {
            info!("Received session_id from queue: {}", session_id);
            if let Err(e) = tx.send(session_id).await {
                error!("Failed to send session_id to processor channel: {}", e);
            }
        }
    }
}
