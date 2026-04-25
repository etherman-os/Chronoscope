use anyhow::Result;
use ffmpeg_next as ffmpeg;
use std::path::Path;

#[derive(Debug, serde::Serialize, serde::Deserialize)]
pub struct VideoIndex {
    pub video_url: String,
    pub duration_ms: u64,
    pub keyframes: Vec<Keyframe>,
    pub event_timeline: Vec<TimelineEvent>,
}

#[derive(Debug, serde::Serialize, serde::Deserialize)]
pub struct Keyframe {
    pub timestamp_ms: u64,
    pub byte_offset: u64,
}

#[derive(Debug, serde::Serialize, serde::Deserialize, Clone)]
pub struct TimelineEvent {
    pub event_type: String,
    pub timestamp_ms: u64,
    pub x: i32,
    pub y: i32,
}

pub async fn generate_index(
    video_path: &Path,
    timeline: &crate::sync::EventTimeline,
) -> Result<VideoIndex> {
    let video_path = video_path.to_path_buf();
    let timeline_events = timeline.events.clone();

    let index = tokio::task::spawn_blocking(move || -> Result<VideoIndex> {
        ffmpeg::init()?;
        let mut ictx = ffmpeg::format::input(&video_path)?;
        let stream = ictx
            .streams()
            .best(ffmpeg::media::Type::Video)
            .ok_or_else(|| anyhow::anyhow!("no video stream found"))?;

        let duration_ms = if stream.duration() > 0 {
            (stream.duration() as f64 * f64::from(stream.time_base()) * 1000.0) as u64
        } else {
            0
        };

        let video_stream_index = stream.index();
        let mut keyframes: Vec<Keyframe> = Vec::new();

        for (s, packet) in ictx.packets() {
            if s.index() == video_stream_index {
                let is_key = packet.is_key();
                if is_key {
                    let ts = packet.pts().or_else(|| packet.dts()).unwrap_or(0);
                    let timestamp_ms = (ts as f64 * f64::from(s.time_base()) * 1000.0) as u64;
                    let byte_offset = 0;
                    keyframes.push(Keyframe {
                        timestamp_ms,
                        byte_offset,
                    });
                }
            }
        }

        // Derive a placeholder video URL; caller can override if needed
        let video_url = format!("file://{}", video_path.display());

        Ok(VideoIndex {
            video_url,
            duration_ms,
            keyframes,
            event_timeline: timeline_events,
        })
    })
    .await?;

    index
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sync::EventTimeline;
    use image::{ImageBuffer, Rgb};

    #[test]
    fn test_generate_index_with_video() {
        let rt = tokio::runtime::Runtime::new().unwrap();
        let temp_dir = tempfile::tempdir().unwrap();
        let img_path = temp_dir.path().join("frame1.jpg");
        let img: ImageBuffer<Rgb<u8>, Vec<u8>> = ImageBuffer::from_pixel(64, 64, Rgb([255, 0, 0]));
        img.save(&img_path).unwrap();

        let video_path = rt.block_on(async {
            crate::encoder::encode_h264_impl("index_test", vec![img_path])
                .await
                .unwrap()
        });

        let timeline = EventTimeline { events: vec![] };
        let index = rt.block_on(async {
            generate_index(&video_path, &timeline).await.unwrap()
        });

        assert!(index.video_url.contains("file://"));
        let _ = std::fs::remove_file(&video_path);
    }
}


