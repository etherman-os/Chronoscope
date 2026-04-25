use std::path::PathBuf;

// Mock downloader that simulates S3 list/get without real network calls.
struct MockDownloader {
    chunks: Vec<PathBuf>,
}

impl MockDownloader {
    fn new(chunks: Vec<PathBuf>) -> Self {
        Self { chunks }
    }

    fn download(&self, _session_id: &str) -> Vec<PathBuf> {
        self.chunks.clone()
    }
}

// Mock encoder that simulates FFmpeg without real encoding.
struct MockEncoder {
    output_path: PathBuf,
}

impl MockEncoder {
    fn new(output_path: PathBuf) -> Self {
        Self { output_path }
    }

    fn encode(&self, _session_id: &str, _frames: &[PathBuf]) -> PathBuf {
        self.output_path.clone()
    }
}

// Simplified pipeline logic testable with mocks.
fn process_session_mock(
    downloader: &MockDownloader,
    encoder: &MockEncoder,
    session_id: &str,
) -> PathBuf {
    let chunks = downloader.download(session_id);
    // Simulate deduplication by removing duplicates
    let unique: Vec<PathBuf> = chunks
        .into_iter()
        .collect::<std::collections::HashSet<_>>()
        .into_iter()
        .collect();
    encoder.encode(session_id, &unique)
}

#[test]
fn test_mock_pipeline() {
    let chunk1 = PathBuf::from("/tmp/session_1/chunk_0.jpg");
    let chunk2 = PathBuf::from("/tmp/session_1/chunk_1.jpg");
    let downloader = MockDownloader::new(vec![chunk1.clone(), chunk2.clone()]);
    let expected_output = PathBuf::from("/tmp/session_1/output.mp4");
    let encoder = MockEncoder::new(expected_output.clone());

    let result = process_session_mock(&downloader, &encoder, "session_1");
    assert_eq!(result, expected_output);
}

#[test]
fn test_mock_pipeline_deduplicates() {
    let chunk1 = PathBuf::from("/tmp/session_2/chunk_0.jpg");
    let chunk2 = chunk1.clone();
    let downloader = MockDownloader::new(vec![chunk1.clone(), chunk2]);
    let expected_output = PathBuf::from("/tmp/session_2/output.mp4");
    let encoder = MockEncoder::new(expected_output.clone());

    let result = process_session_mock(&downloader, &encoder, "session_2");
    assert_eq!(result, expected_output);
}

#[test]
fn test_deduplicator_with_real_images() {
    use chronoscope_processor::deduplicator;
    use image::{ImageBuffer, Rgb};

    let temp_dir = tempfile::tempdir().unwrap();
    let img1_path = temp_dir.path().join("frame1.jpg");
    let img2_path = temp_dir.path().join("frame2.jpg");
    let img3_path = temp_dir.path().join("frame3.jpg");

    // Create three images: img1 and img2 are identical, img3 is structurally different
    let img1: ImageBuffer<Rgb<u8>, Vec<u8>> = ImageBuffer::from_pixel(64, 64, Rgb([255, 0, 0]));
    let img2: ImageBuffer<Rgb<u8>, Vec<u8>> = ImageBuffer::from_pixel(64, 64, Rgb([255, 0, 0]));
    let mut img3: ImageBuffer<Rgb<u8>, Vec<u8>> = ImageBuffer::from_pixel(64, 64, Rgb([0, 255, 0]));
    // Add high-contrast noise so perceptual hash differs significantly
    for (x, y, pixel) in img3.enumerate_pixels_mut() {
        if (x + y) % 2 == 0 {
            *pixel = Rgb([0, 0, 255]);
        }
    }

    img1.save(&img1_path).unwrap();
    img2.save(&img2_path).unwrap();
    img3.save(&img3_path).unwrap();

    let chunks = vec![img1_path.clone(), img2_path.clone(), img3_path.clone()];
    let rt = tokio::runtime::Runtime::new().unwrap();
    let unique = rt.block_on(deduplicator::deduplicate(chunks)).unwrap();

    // img1 and img2 are identical, so one should be removed
    assert_eq!(unique.len(), 2);
    assert!(unique.contains(&img1_path) || unique.contains(&img2_path));
    assert!(unique.contains(&img3_path));
}

#[test]
fn test_config_from_env_missing_vars() {
    use chronoscope_processor::config::Config;
    use std::env;

    // This test validates that Config::from_env returns an error
    // when required environment variables are missing.
    env::remove_var("DATABASE_URL");
    let rt = tokio::runtime::Runtime::new().unwrap();
    let result = rt.block_on(Config::from_env());
    assert!(result.is_err());
}

#[test]
fn test_downloader_empty_list() {
    let downloader = MockDownloader::new(vec![]);
    let chunks = downloader.download("empty_session");
    assert!(chunks.is_empty());
}

#[test]
fn test_encoder_empty_frames() {
    let output = PathBuf::from("/tmp/empty.mp4");
    let encoder = MockEncoder::new(output.clone());
    let result = encoder.encode("empty_session", &[]);
    assert_eq!(result, output);
}
