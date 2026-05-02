use anyhow::Result;
use chronoscope_sdk_linux::{
    upload::{ChunkUploader, SessionEvent},
    CaptureConfig, CaptureMode,
};
use clap::Parser;
use image::{ImageBuffer, ImageOutputFormat, Rgb};
use serde_json::json;
use std::io::Cursor;

#[derive(Parser, Debug)]
#[command(name = "chronoscope-demo-session")]
#[command(about = "Upload a synthetic replay session to a self-hosted Chronoscope instance")]
struct Args {
    #[arg(long, default_value = "http://localhost:8080")]
    endpoint: String,

    #[arg(long, default_value = "local-dev-key")]
    api_key: String,

    #[arg(long, default_value = "demo-user")]
    user_id: String,

    #[arg(long, default_value_t = 45)]
    frames: u32,

    #[arg(long, default_value_t = 5)]
    fps: u32,

    #[arg(long, default_value_t = 960)]
    width: u32,

    #[arg(long, default_value_t = 540)]
    height: u32,
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt::init();
    let args = Args::parse();

    let frame_count = args.frames.clamp(1, 300);
    let fps = args.fps.clamp(1, 30);
    let width = args.width.clamp(320, 1920);
    let height = args.height.clamp(180, 1080);

    let mut config = CaptureConfig::new(args.api_key, args.endpoint).with_user_id(args.user_id);
    config.capture_mode = CaptureMode::Hybrid;
    config.frame_rate = fps;

    let mut uploader = ChunkUploader::new(&config)?;
    let session_id = uploader
        .initialize_session(&config.user_id, &config.capture_mode)
        .await?;

    println!("Chronoscope demo session started: {}", session_id);

    for index in 0..frame_count {
        let frame = render_demo_frame(index, frame_count, width, height)?;
        uploader.upload_chunk(frame, index).await?;
    }

    let duration_ms = ((frame_count as f64 / fps as f64) * 1000.0).round() as i32;
    uploader.upload_events(demo_events(duration_ms, width, height)).await?;
    uploader
        .finalize_with_duration(Some(duration_ms.max(1) as u128))
        .await?;

    println!(
        "Chronoscope demo session completed: {} ({} frames, {} ms)",
        session_id, frame_count, duration_ms
    );
    Ok(())
}

fn render_demo_frame(index: u32, total: u32, width: u32, height: u32) -> Result<Vec<u8>> {
    let progress = if total <= 1 {
        0.0
    } else {
        index as f32 / (total - 1) as f32
    };

    let mut img = ImageBuffer::from_fn(width, height, |x, y| {
        let horizontal = x as f32 / width.max(1) as f32;
        let vertical = y as f32 / height.max(1) as f32;
        let base = (22.0 + horizontal * 42.0 + vertical * 30.0) as u8;
        Rgb([base, base.saturating_add(18), base.saturating_add(38)])
    });

    draw_rect(&mut img, 0, 0, width, 72, Rgb([16, 24, 39]));
    draw_rect(&mut img, 32, 22, 210, 30, Rgb([81, 162, 255]));
    draw_rect(
        &mut img,
        265,
        26,
        (width as i32 - 300).max(120) as u32,
        20,
        Rgb([52, 73, 94]),
    );

    let panel_w = (width / 3).max(220);
    draw_rect(&mut img, 42, 112, panel_w, height - 168, Rgb([245, 247, 250]));
    draw_rect(&mut img, 74, 148, panel_w - 64, 28, Rgb([32, 43, 54]));
    draw_rect(&mut img, 74, 208, panel_w - 84, 18, Rgb([134, 148, 164]));
    draw_rect(&mut img, 74, 248, panel_w - 120, 18, Rgb([134, 148, 164]));
    draw_rect(&mut img, 74, 304, 150, 44, Rgb([81, 162, 255]));

    let replay_x = panel_w + 86;
    let replay_w = width.saturating_sub(replay_x + 42);
    let replay_h = height.saturating_sub(168);
    draw_rect(&mut img, replay_x, 112, replay_w, replay_h, Rgb([12, 18, 28]));

    let card_w = (replay_w / 3).max(110);
    let moving_x = replay_x + 34 + ((replay_w.saturating_sub(card_w + 68)) as f32 * progress) as u32;
    let moving_y = 172 + ((progress * std::f32::consts::TAU).sin().abs() * 96.0) as u32;
    draw_rect(&mut img, moving_x, moving_y, card_w, 96, Rgb([255, 255, 255]));
    draw_rect(&mut img, moving_x + 22, moving_y + 20, card_w - 44, 16, Rgb([39, 50, 63]));
    draw_rect(&mut img, moving_x + 22, moving_y + 52, card_w - 80, 12, Rgb([102, 117, 132]));

    let cursor_x = replay_x + 42 + ((replay_w.saturating_sub(84)) as f32 * progress) as u32;
    let cursor_y = 148 + ((replay_h.saturating_sub(96)) as f32 * (1.0 - progress)) as u32;
    draw_cursor(&mut img, cursor_x, cursor_y);

    let timeline_y = height.saturating_sub(42);
    draw_rect(&mut img, replay_x, timeline_y, replay_w, 8, Rgb([71, 85, 105]));
    draw_rect(
        &mut img,
        replay_x,
        timeline_y,
        (replay_w as f32 * progress) as u32,
        8,
        Rgb([81, 162, 255]),
    );

    let mut bytes = Vec::new();
    image::DynamicImage::ImageRgb8(img).write_to(
        &mut Cursor::new(&mut bytes),
        ImageOutputFormat::Jpeg(86),
    )?;
    Ok(bytes)
}

fn draw_rect(
    img: &mut ImageBuffer<Rgb<u8>, Vec<u8>>,
    x: u32,
    y: u32,
    width: u32,
    height: u32,
    color: Rgb<u8>,
) {
    let max_x = (x + width).min(img.width());
    let max_y = (y + height).min(img.height());
    for yy in y.min(img.height())..max_y {
        for xx in x.min(img.width())..max_x {
            img.put_pixel(xx, yy, color);
        }
    }
}

fn draw_cursor(img: &mut ImageBuffer<Rgb<u8>, Vec<u8>>, x: u32, y: u32) {
    for offset in 0..28 {
        draw_rect(img, x + offset / 2, y + offset, 3, 3, Rgb([255, 255, 255]));
    }
    draw_rect(img, x + 10, y + 20, 16, 5, Rgb([255, 255, 255]));
    draw_rect(img, x + 18, y + 24, 6, 16, Rgb([255, 255, 255]));
    draw_rect(img, x + 2, y + 2, 2, 22, Rgb([15, 23, 42]));
    draw_rect(img, x + 10, y + 24, 16, 2, Rgb([15, 23, 42]));
}

fn demo_events(duration_ms: i32, width: u32, height: u32) -> Vec<SessionEvent> {
    vec![
        SessionEvent {
            event_type: "click".to_string(),
            timestamp_ms: duration_ms / 4,
            x: (width / 4) as i32,
            y: (height / 2) as i32,
            target: "demo.primary_button".to_string(),
            payload: json!({"label": "Start import"}),
        },
        SessionEvent {
            event_type: "scroll".to_string(),
            timestamp_ms: duration_ms / 2,
            x: (width / 2) as i32,
            y: (height / 2) as i32,
            target: "demo.replay_panel".to_string(),
            payload: json!({"delta_y": 420}),
        },
        SessionEvent {
            event_type: "click".to_string(),
            timestamp_ms: duration_ms * 3 / 4,
            x: (width * 3 / 4) as i32,
            y: (height / 3) as i32,
            target: "demo.export_menu".to_string(),
            payload: json!({"label": "Export"}),
        },
    ]
}
