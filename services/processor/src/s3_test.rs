#[cfg(test)]
mod tests {
    #[test]
    fn test_s3_client_creation() {
        let cfg = aws_sdk_s3::config::Builder::new().behavior_version_latest().build();
        let _client = aws_sdk_s3::Client::from_conf(cfg);
        println!("S3 client created");
    }
}
