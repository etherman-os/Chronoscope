use anyhow::Result;
use aws_sdk_s3::Client as S3Client;
use deadpool_postgres::Pool;
use redis::aio::MultiplexedConnection;

#[derive(Clone)]
pub struct Config {
    pub db_pool: Pool,
    pub s3_client: S3Client,
    pub redis_client: MultiplexedConnection,
    pub bucket_name: String,
    pub processed_bucket_name: String,
}

impl Config {
    pub async fn from_env() -> Result<Self> {
        let db_url = std::env::var("DATABASE_URL")
            .unwrap_or_else(|_| "postgres://chronoscope:chronoscope@localhost:5432/chronoscope".to_string());

        let mut pg_cfg = deadpool_postgres::Config::new();
        pg_cfg.url = Some(db_url);
        let db_pool = pg_cfg.create_pool(None, tokio_postgres::NoTls)?;

        let aws_endpoint = std::env::var("AWS_ENDPOINT_URL")
            .unwrap_or_else(|_| "http://localhost:9000".to_string());
        let access_key = std::env::var("AWS_ACCESS_KEY_ID")
            .unwrap_or_else(|_| "chronoscope".to_string());
        let secret_key = std::env::var("AWS_SECRET_ACCESS_KEY")
            .unwrap_or_else(|_| "chronoscope123".to_string());

        let aws_cfg = aws_config::from_env()
            .endpoint_url(aws_endpoint)
            .credentials_provider(aws_sdk_s3::config::Credentials::new(
                access_key,
                secret_key,
                None,
                None,
                "env",
            ))
            .load()
            .await;

        let s3_config = aws_sdk_s3::config::Builder::from(&aws_cfg)
            .force_path_style(true)
            .build();
        let s3_client = S3Client::from_conf(s3_config);

        let redis_url = std::env::var("REDIS_URL")
            .unwrap_or_else(|_| "redis://localhost:6379".to_string());
        let redis_client = {
            let client = redis::Client::open(redis_url)?;
            client.get_multiplexed_tokio_connection().await?
        };

        let bucket_name = std::env::var("S3_BUCKET")
            .unwrap_or_else(|_| "chronoscope-sessions".to_string());
        let processed_bucket_name = std::env::var("S3_PROCESSED_BUCKET")
            .unwrap_or_else(|_| "chronoscope-processed".to_string());

        Ok(Config {
            db_pool,
            s3_client,
            redis_client,
            bucket_name,
            processed_bucket_name,
        })
    }
}
