use anyhow::Result;
use std::sync::Arc;
use tokio::sync::Mutex;
use tokio_util::sync::CancellationToken;
use x11rb::connection::Connection;
use x11rb::protocol::xproto::*;
use x11rb::rust_connection::RustConnection;

pub async fn start_capture(
    buffer: Arc<Mutex<crate::buffer::CircularBuffer>>,
    frame_rate: u32,
    cancel_token: CancellationToken,
) -> Result<()> {
    tracing::info!("Starting X11 SHM capture at {} fps", frame_rate);

    let (conn, screen_num) = RustConnection::connect(None)?;
    let screen = &conn.setup().roots[screen_num];
    let root = screen.root;

    let geom = conn.get_geometry(root)?.reply()?;
    let width = geom.width;
    let height = geom.height;

    tracing::debug!("X11 root window: {}x{}", width, height);

    let interval_ms = 1000 / frame_rate.max(1);
    let mut interval =
        tokio::time::interval(tokio::time::Duration::from_millis(interval_ms as u64));

    loop {
        tokio::select! {
            _ = interval.tick() => {
                // Placeholder: actual capture logic will be implemented here.
                let _ = buffer;
                let _ = width;
                let _ = height;
            }
            _ = cancel_token.cancelled() => {
                tracing::info!("X11 capture cancelled, shutting down");
                break;
            }
        }
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;
    use tokio::sync::Mutex;
    use tokio_util::sync::CancellationToken;
    use crate::buffer::CircularBuffer;

    #[tokio::test]
    async fn test_x11_capture_without_display() {
        let orig = std::env::var("DISPLAY").ok();
        std::env::remove_var("DISPLAY");
        let buffer = Arc::new(Mutex::new(CircularBuffer::new(1024)));
        let token = CancellationToken::new();
        let result = start_capture(buffer, 1, token).await;
        if let Some(v) = orig {
            std::env::set_var("DISPLAY", v);
        }
        assert!(result.is_err());
    }
}
