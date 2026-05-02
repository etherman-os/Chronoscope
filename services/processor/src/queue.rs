use crate::config::Config;
use anyhow::Result;
use redis::aio::MultiplexedConnection;
use redis::streams::StreamReadReply;
use tokio::time::{sleep, Duration};
use tracing::{error, info, warn};

const STREAM_KEY: &str = "chronoscope:process_queue";
const GROUP_NAME: &str = "chronoscope-processor";
const CONSUMER_NAME: &str = "processor-1";

async fn create_consumer_group(con: &mut MultiplexedConnection) -> Result<()> {
    let result: redis::RedisResult<()> = redis::cmd("XGROUP")
        .arg("CREATE")
        .arg(STREAM_KEY)
        .arg(GROUP_NAME)
        .arg("0")
        .arg("MKSTREAM")
        .query_async(con)
        .await;

    if let Err(e) = result {
        let err_str = e.to_string();
        if err_str.contains("BUSYGROUP") {
            info!("Consumer group already exists");
        } else {
            return Err(e.into());
        }
    } else {
        info!("Created consumer group '{}'", GROUP_NAME);
    }

    Ok(())
}

pub async fn queue_listener(config: Config, tx: tokio::sync::mpsc::Sender<String>) -> Result<()> {
    let mut con = config
        .redis_client
        .get_multiplexed_async_connection()
        .await?;
    create_consumer_group(&mut con).await?;

    let mut backoff_secs = 5u64;

    loop {
        let result: redis::RedisResult<StreamReadReply> = redis::cmd("XREADGROUP")
            .arg("GROUP")
            .arg(GROUP_NAME)
            .arg(CONSUMER_NAME)
            .arg("COUNT")
            .arg(1)
            .arg("BLOCK")
            .arg(5000)
            .arg("STREAMS")
            .arg(STREAM_KEY)
            .arg(">")
            .query_async(&mut con)
            .await;

        match result {
            Ok(reply) => {
                backoff_secs = 5;
                for stream in reply.keys {
                    for entry in stream.ids {
                        let session_id = entry.get::<String>("session_id").unwrap_or_default();

                        if session_id.is_empty() {
                            warn!("Empty session_id in stream entry {}", entry.id);
                            let _: redis::RedisResult<()> = redis::cmd("XACK")
                                .arg(STREAM_KEY)
                                .arg(GROUP_NAME)
                                .arg(&entry.id)
                                .query_async(&mut con)
                                .await;
                            continue;
                        }

                        info!("Received session_id from stream: {}", session_id);

                        if let Err(e) = tx.send(session_id).await {
                            error!("Failed to send session_id to processor channel: {}", e);
                        }

                        let _: redis::RedisResult<()> = redis::cmd("XACK")
                            .arg(STREAM_KEY)
                            .arg(GROUP_NAME)
                            .arg(&entry.id)
                            .query_async(&mut con)
                            .await;
                    }
                }
            }
            Err(e) => {
                warn!(
                    "Redis XREADGROUP error: {}. Reconnecting in {}s...",
                    e, backoff_secs
                );
                sleep(Duration::from_secs(backoff_secs)).await;
                match config.redis_client.get_multiplexed_async_connection().await {
                    Ok(new_con) => {
                        con = new_con;
                        create_consumer_group(&mut con).await?;
                        info!("Redis reconnected successfully");
                    }
                    Err(reconnect_err) => {
                        error!("Redis reconnect failed: {}. Retrying...", reconnect_err);
                    }
                }
                backoff_secs = (backoff_secs * 2).min(60);
                continue;
            }
        }
    }
}
