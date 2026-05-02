use anyhow::Result;
use chronoscope_sdk_linux::{CaptureConfig, CaptureMode, LinuxCapture};
use clap::{Parser, ValueEnum};
use std::time::Duration;

#[derive(Parser, Debug)]
#[command(name = "chronoscope-linux-record")]
#[command(about = "Record a Linux desktop session into a self-hosted Chronoscope instance")]
struct Args {
    #[arg(long, default_value = "http://localhost:8080")]
    endpoint: String,

    #[arg(long)]
    api_key: String,

    #[arg(long, default_value = "linux-user")]
    user_id: String,

    #[arg(long, default_value_t = 5)]
    fps: u32,

    #[arg(long)]
    duration: Option<u64>,

    #[arg(long, value_enum, default_value_t = ModeArg::Hybrid)]
    mode: ModeArg,
}

#[derive(Copy, Clone, Debug, ValueEnum)]
enum ModeArg {
    Video,
    Events,
    Hybrid,
}

impl From<ModeArg> for CaptureMode {
    fn from(value: ModeArg) -> Self {
        match value {
            ModeArg::Video => CaptureMode::Video,
            ModeArg::Events => CaptureMode::Events,
            ModeArg::Hybrid => CaptureMode::Hybrid,
        }
    }
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt::init();
    let args = Args::parse();

    let mut config = CaptureConfig::new(args.api_key, args.endpoint).with_user_id(args.user_id);
    config.frame_rate = args.fps.max(1);
    config.capture_mode = args.mode.into();

    let mut capture = LinuxCapture::new(config)?;
    capture.start().await?;

    if let Some(session_id) = capture.session_id() {
        println!("Chronoscope session started: {}", session_id);
    }

    if let Some(seconds) = args.duration {
        tokio::time::sleep(Duration::from_secs(seconds)).await;
    } else {
        tokio::signal::ctrl_c().await?;
    }

    capture.stop().await?;
    println!("Chronoscope session completed");
    Ok(())
}
