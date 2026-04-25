use anyhow::{Context, Result};
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
        let db_url = std::env::var("DATABASE_URL").context("DATABASE_URL must be set")?;

        let mut pg_cfg = deadpool_postgres::Config::new();
        pg_cfg.url = Some(db_url);
        let db_pool = pg_cfg.create_pool(None, tokio_postgres::NoTls)?;

        let aws_endpoint = std::env::var("AWS_ENDPOINT_URL")
            .unwrap_or_else(|_| "http://localhost:9000".to_string());
        let access_key =
            std::env::var("AWS_ACCESS_KEY_ID").context("AWS_ACCESS_KEY_ID must be set")?;
        let secret_key =
            std::env::var("AWS_SECRET_ACCESS_KEY").context("AWS_SECRET_ACCESS_KEY must be set")?;

        let aws_cfg = aws_config::defaults(aws_config::BehaviorVersion::latest())
            .endpoint_url(aws_endpoint)
            .credentials_provider(aws_sdk_s3::config::Credentials::new(
                access_key, secret_key, None, None, "env",
            ))
            .load()
            .await;

        let s3_config = aws_sdk_s3::config::Builder::from(&aws_cfg)
            .force_path_style(true)
            .build();
        let s3_client = S3Client::from_conf(s3_config);

        let redis_url =
            std::env::var("REDIS_URL").unwrap_or_else(|_| "redis://localhost:6379".to_string());
        let redis_client = {
            let client = redis::Client::open(redis_url)?;
            client.get_multiplexed_tokio_connection().await?
        };

        let bucket_name = std::env::var("S3_BUCKET").context("S3_BUCKET must be set")?;
        let processed_bucket_name =
            std::env::var("S3_PROCESSED_BUCKET").context("S3_PROCESSED_BUCKET must be set")?;

        Ok(Config {
            db_pool,
            s3_client,
            redis_client,
            bucket_name,
            processed_bucket_name,
        })
    }
}

#[cfg(test)]
mod tests {
    #[test]
    fn test_pool_creation_fake_url() {
        let mut pg_cfg = deadpool_postgres::Config::new();
        pg_cfg.url = Some("postgres://fake:5432/db".to_string());
        let res = pg_cfg.create_pool(None, tokio_postgres::NoTls);
        println!("Pool result: {:?}", res.is_ok());
    }
}
