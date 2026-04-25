use crate::config::Config;
use anyhow::{Context, Result};
use ffmpeg::{codec, format, frame, Rational};
use ffmpeg::codec::traits::Encoder;
use ffmpeg_next as ffmpeg;
use image::GenericImageView;
use std::path::PathBuf;

pub async fn encode_h264(
    _config: &Config,
    session_id: &str,
    frames: Vec<PathBuf>,
) -> Result<PathBuf> {
    encode_h264_impl(session_id, frames).await
}

pub async fn encode_h264_impl(session_id: &str, frames: Vec<PathBuf>) -> Result<PathBuf> {
    let session_id = session_id.to_string();
    let output_path = tokio::task::spawn_blocking(move || -> Result<PathBuf> {
        ffmpeg::init()?;

        if frames.is_empty() {
            anyhow::bail!("no frames to encode");
        }

        let safe_id = session_id.replace(['/', '\\', '\0'], "_");
        let output_path = std::env::temp_dir().join(format!("{}.mp4", safe_id));
        let mut octx = format::output(&output_path)?;

        // Infer dimensions from the first frame
        let first_img = image::open(&frames[0]).context("open first frame")?;
        let (width, height) = (first_img.width() as i32, first_img.height() as i32);

        let global_header = octx.format().flags().contains(format::Flags::GLOBAL_HEADER);

        let codec = ffmpeg::encoder::find(codec::Id::H264).context("find h264 encoder")?;
        let enc = unsafe {
            let ptr = ffmpeg_next::ffi::avcodec_alloc_context3(
                codec.encoder().unwrap().as_ptr(),
            );
            if ptr.is_null() {
                anyhow::bail!("Failed to allocate codec context");
            }
            codec::context::Context::wrap(ptr, None)
        };
        let mut video = enc.encoder().video()?;

        video.set_width(width as u32);
        video.set_height(height as u32);
        video.set_time_base(Rational::new(1, 30)); // 30 fps
        video.set_format(ffmpeg::format::Pixel::YUV420P);
        if global_header {
            video.set_flags(codec::Flags::GLOBAL_HEADER);
        }

        let mut opts = ffmpeg::Dictionary::new();
        opts.set("preset", "ultrafast");
        let mut video = video.open_as_with(Some(codec), opts)?;

        let stream_index = {
            let mut ost = octx.add_stream(codec)?;
            ost.set_parameters(&video);
            ost.set_time_base(Rational::new(1, 30));
            ost.index()
        };

        octx.write_header()?;

        let mut scaler = ffmpeg::software::scaling::Context::get(
            ffmpeg::format::Pixel::RGB24,
            width as u32,
            height as u32,
            ffmpeg::format::Pixel::YUV420P,
            width as u32,
            height as u32,
            ffmpeg::software::scaling::Flags::BILINEAR,
        )?;

        let mut frame_idx = 0i64;
        for path in &frames {
            let img = match image::open(path) {
                Ok(i) => i.to_rgb8(),
                Err(e) => {
                    tracing::warn!("skip frame {}: {}", path.display(), e);
                    continue;
                }
            };

            let raw = img.into_raw();
            let mut rgb_frame =
                frame::Video::new(ffmpeg::format::Pixel::RGB24, width as u32, height as u32);
            if raw.len() != rgb_frame.data(0).len() {
                tracing::warn!(
                    "Skipping frame with mismatched dimensions: {}",
                    path.display()
                );
                continue;
            }
            rgb_frame.data_mut(0).copy_from_slice(&raw);

            let mut yuv_frame = frame::Video::empty();
            scaler.run(&rgb_frame, &mut yuv_frame)?;
            yuv_frame.set_pts(Some(frame_idx));
            frame_idx += 1;

            video.send_frame(&yuv_frame)?;
            receive_and_write_packets(&mut video, &mut octx, stream_index)?;
        }

        video.send_eof()?;
        receive_and_write_packets(&mut video, &mut octx, stream_index)?;

        octx.write_trailer()?;
        Ok(output_path)
    })
    .await?;

    output_path
}

fn receive_and_write_packets(
    video: &mut ffmpeg::encoder::Video,
    octx: &mut format::context::Output,
    stream_index: usize,
) -> Result<()> {
    let mut pkt = ffmpeg::Packet::empty();
    while video.receive_packet(&mut pkt).is_ok() {
        pkt.set_stream(stream_index);
        pkt.write_interleaved(octx)?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use image::{ImageBuffer, Rgb};

    #[tokio::test]
    async fn test_encode_h264_empty_frames() {
        let result = encode_h264_impl("test_empty", vec![]).await;
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("no frames to encode"));
    }

    #[tokio::test]
    async fn test_encode_h264_with_image() {
        let temp_dir = tempfile::tempdir().unwrap();
        let img_path = temp_dir.path().join("frame1.jpg");
        let img: ImageBuffer<Rgb<u8>, Vec<u8>> = ImageBuffer::from_pixel(64, 64, Rgb([255, 0, 0]));
        img.save(&img_path).unwrap();

        let result = encode_h264_impl("test_session", vec![img_path]).await;
        if let Err(ref e) = result {
            eprintln!("Encoder error: {}", e);
        }
        let path = result.unwrap();
        assert!(path.exists());
        let _ = std::fs::remove_file(&path);
    }

}
