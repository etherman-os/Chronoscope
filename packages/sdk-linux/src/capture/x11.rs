use anyhow::Result;
use tokio_util::sync::CancellationToken;
use x11rb::connection::Connection;
use x11rb::protocol::xproto::ConnectionExt;
use x11rb::protocol::xproto::*;
use x11rb::rust_connection::RustConnection;

pub async fn start_capture(
    frame_tx: tokio::sync::mpsc::Sender<Vec<u8>>,
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
                let jpeg = capture_root_jpeg(&conn, root, width, height)?;
                if frame_tx.send(jpeg).await.is_err() {
                    tracing::warn!("Frame receiver closed, stopping X11 capture");
                    break;
                }
            }
            _ = cancel_token.cancelled() => {
                tracing::info!("X11 capture cancelled, shutting down");
                break;
            }
        }
    }

    Ok(())
}

fn capture_root_jpeg(
    conn: &RustConnection,
    root: Window,
    width: u16,
    height: u16,
) -> Result<Vec<u8>> {
    let image = conn
        .get_image(ImageFormat::Z_PIXMAP, root, 0, 0, width, height, u32::MAX)?
        .reply()?;

    let mut rgb = image::RgbImage::new(width as u32, height as u32);
    for (idx, pixel) in image.data.chunks_exact(4).enumerate() {
        let x = (idx as u32) % width as u32;
        let y = (idx as u32) / width as u32;
        if y >= height as u32 {
            break;
        }

        // Most X11 true-color visuals expose root pixels as BGRX/BGRA on little-endian Linux.
        rgb.put_pixel(x, y, image::Rgb([pixel[2], pixel[1], pixel[0]]));
    }

    let max_width = 1280;
    let output = if rgb.width() > max_width {
        let scaled_height =
            ((rgb.height() as f32) * (max_width as f32 / rgb.width() as f32)) as u32;
        image::imageops::resize(
            &rgb,
            max_width,
            scaled_height.max(1),
            image::imageops::FilterType::Triangle,
        )
    } else {
        rgb
    };

    let mut encoded = Vec::new();
    let mut encoder = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut encoded, 72);
    encoder.encode_image(&output)?;
    Ok(encoded)
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio_util::sync::CancellationToken;

    #[tokio::test]
    async fn test_x11_capture_without_display() {
        let orig = std::env::var("DISPLAY").ok();
        std::env::remove_var("DISPLAY");
        let (tx, _rx) = tokio::sync::mpsc::channel(1);
        let token = CancellationToken::new();
        let result = start_capture(tx, 1, token).await;
        if let Some(v) = orig {
            std::env::set_var("DISPLAY", v);
        }
        assert!(result.is_err());
    }
}
