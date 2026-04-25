# Builder stage
FROM rust:1.75-bookworm AS builder

RUN apt-get update && apt-get install -y \
    ffmpeg \
    libavcodec-dev \
    libavformat-dev \
    libavutil-dev \
    libswscale-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY Cargo.toml ./
COPY src ./src
RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ffmpeg \
    wget \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r appuser && useradd -r -u 1000 -g appuser appuser
RUN mkdir -p /app && chown appuser:appuser /app
USER appuser
WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/target/release/chronoscope-processor /usr/local/bin/chronoscope-processor

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/healthz || exit 1

CMD ["chronoscope-processor"]
