// evdev/libinput integration for Linux input events
// MVP: stub, return Ok(())
// TODO: Implement actual input capture (see issue #input-capture)
#[allow(dead_code)]
pub async fn start_input_capture() -> anyhow::Result<()> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_start_input_capture() {
        let result = start_input_capture().await;
        assert!(result.is_ok());
    }
}
