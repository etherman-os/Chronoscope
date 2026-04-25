use std::process::{Command, Stdio};
use std::time::Duration;

fn start_container(name: &str, args: &[&str]) -> String {
    // Remove any existing container with the same name
    let _ = Command::new("docker")
        .args(["rm", "-f", name])
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status();

    let output = Command::new("docker")
        .args(["run", "-d", "--rm", "--name", name])
        .args(args)
        .output()
        .expect("docker run failed");

    if !output.status.success() {
        panic!(
            "Failed to start container {}: {}",
            name,
            String::from_utf8_lossy(&output.stderr)
        );
    }

    let id = String::from_utf8(output.stdout).unwrap().trim().to_string();
    id
}

fn stop_container(name: &str) {
    let _ = Command::new("docker")
        .args(["stop", "-t", "1", name])
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status();
}

fn wait_for_postgres(_port: u16) {
    for _ in 0..60 {
        if Command::new("docker")
            .args([
                "exec",
                "chronoscope-test-postgres",
                "sh",
                "-c",
                "pg_isready -h localhost -p 5432 -U postgres > /dev/null 2>&1 && echo ready",
            ])
            .output()
            .map(|o| String::from_utf8_lossy(&o.stdout).contains("ready"))
            .unwrap_or(false)
        {
            return;
        }
        std::thread::sleep(Duration::from_millis(500));
    }
    panic!("Postgres did not become ready in time");
}

fn wait_for_minio(port: u16) {
    for _ in 0..60 {
        if Command::new("curl")
            .args(["-sf", &format!("http://localhost:{}/minio/health/live", port)])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()
            .map(|s| s.success())
            .unwrap_or(false)
        {
            return;
        }
        std::thread::sleep(Duration::from_millis(500));
    }
    panic!("MinIO did not become ready in time");
}

fn wait_for_redis(_port: u16) {
    for _ in 0..60 {
        if Command::new("docker")
            .args([
                "exec",
                "chronoscope-test-redis",
                "sh",
                "-c",
                "redis-cli ping | grep PONG > /dev/null 2>&1 && echo ready",
            ])
            .output()
            .map(|o| String::from_utf8_lossy(&o.stdout).contains("ready"))
            .unwrap_or(false)
        {
            return;
        }
        std::thread::sleep(Duration::from_millis(500));
    }
    panic!("Redis did not become ready in time");
}

struct TestInfra {
    pg_port: u16,
    s3_port: u16,
    redis_port: u16,
}

impl TestInfra {
    fn start() -> Self {
        let pg_port = 15432u16;
        let s3_port = 19000u16;
        let redis_port = 16379u16;

        start_container(
            "chronoscope-test-postgres",
            &[
                "-p",
                &format!("{}:5432", pg_port),
                "-e",
                "POSTGRES_PASSWORD=password",
                "postgres:16-alpine",
            ],
        );
        wait_for_postgres(pg_port);

        start_container(
            "chronoscope-test-minio",
            &[
                "-p",
                &format!("{}:9000", s3_port),
                "-e",
                "MINIO_ROOT_USER=minioadmin",
                "-e",
                "MINIO_ROOT_PASSWORD=minioadmin",
                "minio/minio",
                "server",
                "/data",
            ],
        );
        wait_for_minio(s3_port);

        start_container(
            "chronoscope-test-redis",
            &["-p", &format!("{}:6379", redis_port), "redis:7-alpine"],
        );
        wait_for_redis(redis_port);

        TestInfra {
            pg_port,
            s3_port,
            redis_port,
        }
    }
}

impl Drop for TestInfra {
    fn drop(&mut self) {
        stop_container("chronoscope-test-postgres");
        stop_container("chronoscope-test-minio");
        stop_container("chronoscope-test-redis");
    }
}

async fn create_config(infra: &TestInfra) -> chronoscope_processor::config::Config {
    let db_url = format!(
        "postgres://postgres:password@localhost:{}/postgres",
        infra.pg_port
    );
    let mut pg_cfg = deadpool_postgres::Config::new();
    pg_cfg.url = Some(db_url);
    let db_pool = pg_cfg.create_pool(None, tokio_postgres::NoTls).unwrap();

    let aws_cfg = aws_config::defaults(aws_config::BehaviorVersion::latest())
        .region(aws_sdk_s3::config::Region::new("us-east-1"))
        .endpoint_url(format!("http://localhost:{}", infra.s3_port))
        .credentials_provider(aws_sdk_s3::config::Credentials::new(
            "minioadmin",
            "minioadmin",
            None,
            None,
            "test",
        ))
        .load()
        .await;
    let s3_config = aws_sdk_s3::config::Builder::from(&aws_cfg)
        .force_path_style(true)
        .build();
    let s3_client = aws_sdk_s3::Client::from_conf(s3_config);

    let redis_url = format!("redis://localhost:{}", infra.redis_port);
    let redis_client = redis::Client::open(redis_url)
        .unwrap()
        .get_multiplexed_tokio_connection()
        .await
        .unwrap();

    chronoscope_processor::config::Config {
        db_pool,
        s3_client,
        redis_client,
        bucket_name: "chunks".to_string(),
        processed_bucket_name: "processed".to_string(),
    }
}

async fn setup_db(pool: &deadpool_postgres::Pool) {
    let client = pool.get().await.unwrap();
    client
        .batch_execute(
            "CREATE TABLE IF NOT EXISTS sessions (
                id UUID PRIMARY KEY,
                status TEXT,
                processed_at TIMESTAMP,
                video_path TEXT,
                metadata JSONB
            );
            CREATE TABLE IF NOT EXISTS events (
                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                session_id UUID,
                event_type TEXT,
                timestamp_ms BIGINT,
                x INTEGER,
                y INTEGER
            );
            CREATE EXTENSION IF NOT EXISTS pgcrypto;",
        )
        .await
        .unwrap();
}

async fn setup_s3(config: &chronoscope_processor::config::Config) {
    for bucket in [&config.bucket_name, &config.processed_bucket_name] {
        let _ = config
            .s3_client
            .create_bucket()
            .bucket(bucket.to_string())
            .send()
            .await;
    }
}

#[tokio::test]
async fn test_full_pipeline() {
    let infra = TestInfra::start();
    let config = create_config(&infra).await;

    setup_db(&config.db_pool).await;
    setup_s3(&config).await;

    // 1. Test uploader and downloader
    let temp_dir = tempfile::tempdir().unwrap();
    let chunk_path = temp_dir.path().join("frame1.jpg");
    let img: image::ImageBuffer<image::Rgb<u8>, Vec<u8>> =
        image::ImageBuffer::from_pixel(64, 64, image::Rgb([255, 0, 0]));
    img.save(&chunk_path).unwrap();

    // Upload a chunk
    let body = aws_sdk_s3::primitives::ByteStream::from_path(&chunk_path).await.unwrap();
    config
        .s3_client
        .put_object()
        .bucket(&config.bucket_name)
        .key("test_session/frame1.jpg")
        .body(body)
        .send()
        .await
        .unwrap();

    // Download chunks
    let (_dir, chunks) = chronoscope_processor::downloader::download_chunks(&config, "test_session")
        .await
        .unwrap();
    assert_eq!(chunks.len(), 1);
    assert!(chunks[0].file_name().unwrap() == "frame1.jpg");

    // 2. Test deduplicator
    let unique = chronoscope_processor::deduplicator::deduplicate(chunks.clone())
        .await
        .unwrap();
    assert_eq!(unique.len(), 1);

    // 3. Test encoder
    let video_path = chronoscope_processor::encoder::encode_h264_impl("test_session", unique)
        .await
        .unwrap();
    assert!(video_path.exists());

    // 4. Test indexer
    let timeline = chronoscope_processor::sync::EventTimeline { events: vec![] };
    let index = chronoscope_processor::indexer::generate_index(&video_path, &timeline)
        .await
        .unwrap();
    assert!(index.video_url.contains("file://"));

    // 5. Test uploader
    chronoscope_processor::uploader::upload_video(&config, "test_session", &video_path)
        .await
        .unwrap();

    // 6. Test db::update_session_status
    let client = config.db_pool.get().await.unwrap();
    let session_id = "550e8400-e29b-41d4-a716-446655440000";
    client
        .execute(
            "INSERT INTO sessions (id, status) VALUES ($1, 'uploading')",
            &[&uuid::Uuid::parse_str(session_id).unwrap()],
        )
        .await
        .unwrap();

    chronoscope_processor::db::update_session_status(&config, session_id, "ready", &index)
        .await
        .unwrap();

    let row = client
        .query_one("SELECT status FROM sessions WHERE id = $1", &[&uuid::Uuid::parse_str(session_id).unwrap()])
        .await
        .unwrap();
    assert_eq!(row.get::<_, String>(0), "ready");

    // 7. Test sync::synchronize_events
    client
        .execute(
            "INSERT INTO events (session_id, event_type, timestamp_ms, x, y) VALUES ($1, 'click', 1000, 10, 20)",
            &[&uuid::Uuid::parse_str(session_id).unwrap()],
        )
        .await
        .unwrap();

    let events = chronoscope_processor::sync::synchronize_events(&config, session_id, &video_path)
        .await
        .unwrap();
    assert_eq!(events.events.len(), 1);
    assert_eq!(events.events[0].event_type, "click");

    // 8. Test queue::queue_listener
    let (tx, mut rx) = tokio::sync::mpsc::channel::<String>(10);
    let config_clone = config.clone();
    let listener_handle = tokio::spawn(async move {
        let _ = chronoscope_processor::queue::queue_listener(config_clone, tx).await;
    });

    // Push a message
    let mut con = config.redis_client.clone();
    redis::cmd("LPUSH")
        .arg("chronoscope:process_queue")
        .arg("queued_session_123")
        .query_async::<_, ()>(&mut con)
        .await
        .unwrap();

    let received = tokio::time::timeout(Duration::from_secs(5), rx.recv())
        .await
        .unwrap()
        .unwrap();
    assert_eq!(received, "queued_session_123");

    listener_handle.abort();

    // Cleanup
    let _ = tokio::fs::remove_file(&video_path).await;
}
