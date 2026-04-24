use chronoscope_sdk_linux::{CaptureConfig, LinuxCapture, detect_display_server};

#[tokio::test]
async fn test_detect_display_server() {
    // Save original env vars
    let wayland_display = std::env::var("WAYLAND_DISPLAY").ok();
    let display = std::env::var("DISPLAY").ok();

    // Clean env
    std::env::remove_var("WAYLAND_DISPLAY");
    std::env::remove_var("DISPLAY");

    // Should fail when neither is set
    assert!(detect_display_server().is_err());

    // Set WAYLAND_DISPLAY
    std::env::set_var("WAYLAND_DISPLAY", "wayland-1");
    let server = detect_display_server().unwrap();
    match server {
        chronoscope_sdk_linux::DisplayServer::Wayland => {},
        _ => panic!("Expected Wayland"),
    }

    // Clean and set DISPLAY
    std::env::remove_var("WAYLAND_DISPLAY");
    std::env::set_var("DISPLAY", ":0");
    let server = detect_display_server().unwrap();
    match server {
        chronoscope_sdk_linux::DisplayServer::X11 => {},
        _ => panic!("Expected X11"),
    }

    // Restore original env vars
    match wayland_display {
        Some(v) => std::env::set_var("WAYLAND_DISPLAY", v),
        None => std::env::remove_var("WAYLAND_DISPLAY"),
    }
    match display {
        Some(v) => std::env::set_var("DISPLAY", v),
        None => std::env::remove_var("DISPLAY"),
    }
}

#[tokio::test]
async fn test_circular_buffer() {
    use chronoscope_sdk_linux::buffer::CircularBuffer;

    let mut buf = CircularBuffer::new(10);
    assert!(buf.is_empty());

    buf.write(b"hello");
    assert_eq!(buf.len(), 5);

    let data = buf.read_chunk().unwrap();
    assert_eq!(data, b"hello");
    assert!(buf.is_empty());

    // Test wrap-around
    let mut buf = CircularBuffer::new(5);
    buf.write(b"hello");
    buf.write(b"world");
    let data = buf.read_chunk().unwrap();
    assert_eq!(data, b"world");
}
