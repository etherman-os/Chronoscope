#[cfg(test)]
mod tests {
    use deadpool_postgres::Config as PgConfig;

    #[test]
    fn test_create_pool_fake() {
        let mut cfg = PgConfig::new();
        cfg.url = Some("postgres://fake:5432/db".to_string());
        let res = cfg.create_pool(None, tokio_postgres::NoTls);
        println!("{:?}", res);
    }
}
