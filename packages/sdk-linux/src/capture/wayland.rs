use anyhow::Result;
use tokio_util::sync::CancellationToken;

pub async fn start_capture(
    _frame_tx: tokio::sync::mpsc::Sender<Vec<u8>>,
    frame_rate: u32,
    cancel_token: CancellationToken,
) -> Result<()> {
    tracing::info!(
        "Starting Wayland capture via PipeWire at {} fps",
        frame_rate
    );

    loop {
        tokio::select! {
            _ = tokio::time::sleep(tokio::time::Duration::from_secs(1)) => {
                tracing::warn!("Wayland/PipeWire capture is not implemented yet; use an X11 session for recording");
            }
            _ = cancel_token.cancelled() => {
                tracing::info!("Wayland capture cancelled, shutting down");
                break;
            }
        }
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio_util::sync::CancellationToken;

    #[tokio::test]
    async fn test_wayland_capture_cancellable() {
        let (tx, _rx) = tokio::sync::mpsc::channel(1);
        let token = CancellationToken::new();
        let child = token.child_token();
        let handle = tokio::spawn(async move { start_capture(tx, 1, child).await });
        tokio::time::sleep(tokio::time::Duration::from_millis(50)).await;
        token.cancel();
        let result = handle.await.unwrap();
        assert!(result.is_ok());
    }
}
