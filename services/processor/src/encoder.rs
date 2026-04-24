use crate::config::Config;
use anyhow::{Context, Result};
use ffmpeg_next as ffmpeg;
use ffmpeg::{codec, format, frame, Rational};
use std::path::PathBuf;

pub async fn encode_h264(
    _config: &Config,
    session_id: &str,
    frames: Vec<PathBuf>,
) -> Result<PathBuf> {
    let session_id = session_id.to_string();
    let output_path = tokio::task::spawn_blocking(move || -> Result<PathBuf> {
        ffmpeg::init()?;

        if frames.is_empty() {
            anyhow::bail!("no frames to encode");
        }

        let output_path = std::env::temp_dir().join(format!("{}.mp4", session_id));
        let mut octx = format::output(&output_path)?;

        // Infer dimensions from the first frame
        let first_img = image::open(&frames[0]).context("open first frame")?;
        let (width, height) = (first_img.width() as i32, first_img.height() as i32);

        let global_header = octx.format().flags().contains(format::Flags::GLOBAL_HEADER);

        let codec = ffmpeg::encoder::find(codec::Id::H264).context("find h264 encoder")?;
        let mut ost = octx.add_stream(codec)?;
        let mut enc = codec::context::Context::new_with_codec(codec);
        let mut video = enc.encoder().video()?;

        video.set_width(width as u32);
        video.set_height(height as u32);
        video.set_time_base(Rational::new(1, 30)); // 30 fps
        video.set_format(ffmpeg::format::Pixel::YUV420P);
        if global_header {
            video.set_flags(codec::Flags::GLOBAL_HEADER);
        }

        let mut video = video.open_as(Some(&codec))?;
        ost.set_parameters(&video);
        ost.set_time_base(Rational::new(1, 30));

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
            let mut rgb_frame = frame::Video::new(ffmpeg::format::Pixel::RGB24, width as u32, height as u32);
            rgb_frame.data_mut(0).copy_from_slice(&raw);

            let mut yuv_frame = frame::Video::empty();
            scaler.run(&rgb_frame, &mut yuv_frame)?;
            yuv_frame.set_pts(Some(frame_idx));
            frame_idx += 1;

            video.send_frame(&yuv_frame)?;
            receive_and_write_packets(&mut video, &mut octx, ost.index())?;
        }

        video.send_eof()?;
        receive_and_write_packets(&mut video, &mut octx, ost.index())?;

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
