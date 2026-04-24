# Chronoscope Processor

Rust-based video processing pipeline for the Chronoscope platform.

## Architecture

The processor consumes session IDs from a Redis queue, downloads raw frame chunks from MinIO (S3-compatible), deduplicates frames using perceptual hashing, encodes the result into H.264 MP4, synchronizes events with the video timeline, generates a keyframe index, uploads the processed video back to MinIO, and updates the session status in PostgreSQL.

## Dependencies

- Rust 1.75+
- FFmpeg (with development libraries for build)
- PostgreSQL
- Redis
- MinIO (S3-compatible storage)

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://chronoscope:chronoscope@localhost:5432/chronoscope` |
| `REDIS_URL` | Redis connection string | `redis://localhost:6379` |
| `AWS_ENDPOINT_URL` | MinIO/S3 endpoint | `http://localhost:9000` |
| `AWS_ACCESS_KEY_ID` | S3 access key | `chronoscope` |
| `AWS_SECRET_ACCESS_KEY` | S3 secret key | `chronoscope123` |
| `S3_BUCKET` | Raw session chunks bucket | `chronoscope-sessions` |
| `S3_PROCESSED_BUCKET` | Processed video output bucket | `chronoscope-processed` |

## Running Locally

1. Install system dependencies:
   ```bash
   sudo apt-get update
   sudo apt-get install ffmpeg libavcodec-dev libavformat-dev libavutil-dev libswscale-dev pkg-config
   ```

2. Copy environment variables:
   ```bash
   cp .env.example .env
   # Edit .env to match your local setup
   ```

3. Build and run:
   ```bash
   cargo build --release
   ./target/release/chronoscope-processor
   ```

## Running with Docker

```bash
docker build -t chronoscope-processor .
docker run --env-file .env chronoscope-processor
```

## Pipeline Steps

1. **Queue Listener** — Blocks on Redis `BRPOP` from `chronoscope:process_queue`
2. **Downloader** — Lists and downloads chunk objects from S3 (prefix: `{session_id}/`)
3. **Deduplicator** — Computes perceptual hashes on each frame; skips duplicates within a Hamming distance of 5
4. **Encoder** — Encodes unique frames into H.264 MP4 using FFmpeg (`libx264`)
5. **Sync** — Queries PostgreSQL events table and builds a timeline mapped to video timestamps
6. **Indexer** — Probes the MP4 with FFmpeg to extract keyframe positions and durations
7. **Uploader** — Uploads the final MP4 to the processed S3 bucket
8. **DB Update** — Sets session `status = 'ready'` and stores metadata

## Logging

Structured logs are emitted via `tracing`. Set `RUST_LOG=info` (or `debug`) to control verbosity.
