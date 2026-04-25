use crate::config::Config;
use anyhow::Result;
use tokio::time::{sleep, Duration};
use tracing::{error, info, warn};

pub async fn queue_listener(config: Config, tx: tokio::sync::mpsc::Sender<String>) -> Result<()> {
    let mut con = config.redis_client;
    loop {
        let result: redis::RedisResult<Option<(String, String)>> = redis::cmd("BRPOP")
            .arg("chronoscope:process_queue")
            .arg(0)
            .query_async(&mut con)
            .await;

        match result {
            Ok(Some((_, session_id))) => {
                info!("Received session_id from queue: {}", session_id);
                if let Err(e) = tx.send(session_id).await {
                    error!("Failed to send session_id to processor channel: {}", e);
                }
            }
            Ok(None) => {
                // No item received, continue polling
                continue;
            }
            Err(e) => {
                warn!("Redis BRPOP error: {}. Reconnecting in 5s...", e);
                sleep(Duration::from_secs(5)).await;
                continue;
            }
        }
    }
}
