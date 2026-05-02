use crate::upload::{ChunkUploader, SessionEvent};
use serde_json::json;
use std::sync::Arc;
use std::time::Instant;
use tokio_util::sync::CancellationToken;
use x11rb::connection::Connection;
use x11rb::protocol::xproto::{ConnectionExt, KeyButMask};
use x11rb::rust_connection::RustConnection;

pub async fn start_input_capture(
    uploader: Arc<ChunkUploader>,
    started_at: Instant,
    cancel_token: CancellationToken,
) -> anyhow::Result<()> {
    let (conn, screen_num) = RustConnection::connect(None)?;
    let root = conn.setup().roots[screen_num].root;
    let mut previous_mask = KeyButMask::default();
    let mut pending = Vec::new();
    let mut interval = tokio::time::interval(tokio::time::Duration::from_millis(50));

    loop {
        tokio::select! {
            _ = interval.tick() => {
                let pointer = conn.query_pointer(root)?.reply()?;
                let mask = pointer.mask;
                let pressed_bits = u16::from(mask) & !u16::from(previous_mask);

                if pressed_bits & u16::from(KeyButMask::BUTTON1) != 0 {
                    pending.push(click_event("click", started_at, pointer.root_x, pointer.root_y, 1));
                }
                if pressed_bits & u16::from(KeyButMask::BUTTON2) != 0 {
                    pending.push(click_event("click", started_at, pointer.root_x, pointer.root_y, 2));
                }
                if pressed_bits & u16::from(KeyButMask::BUTTON3) != 0 {
                    pending.push(click_event("click", started_at, pointer.root_x, pointer.root_y, 3));
                }

                previous_mask = mask;

                if pending.len() >= 10 {
                    let events = std::mem::take(&mut pending);
                    uploader.upload_events(events).await?;
                }
            }
            _ = cancel_token.cancelled() => {
                if !pending.is_empty() {
                    uploader.upload_events(pending).await?;
                }
                break;
            }
        }
    }

    Ok(())
}

fn click_event(event_type: &str, started_at: Instant, x: i16, y: i16, button: u8) -> SessionEvent {
    SessionEvent {
        event_type: event_type.to_string(),
        timestamp_ms: started_at.elapsed().as_millis().min(i32::MAX as u128) as i32,
        x: x as i32,
        y: y as i32,
        target: "x11-root-window".to_string(),
        payload: json!({ "button": button }),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_click_event_shape() {
        let event = click_event("click", Instant::now(), 10, 20, 1);
        assert_eq!(event.event_type, "click");
        assert_eq!(event.x, 10);
        assert_eq!(event.y, 20);
    }
}
