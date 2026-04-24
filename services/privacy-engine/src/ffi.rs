use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use crate::{PrivacyEngine, PrivacyConfig};

/// Initialize privacy engine from JSON config string.
/// Returns opaque pointer. Must be freed with chronoscope_privacy_free.
#[no_mangle]
pub extern "C" fn chronoscope_privacy_init(config_json: *const c_char) -> *mut PrivacyEngine {
    // SAFETY: We check for null before dereferencing. The caller must pass a
    // valid null-terminated C string; if they pass null we return null_mut().
    if config_json.is_null() {
        return std::ptr::null_mut();
    }
    let config_str = unsafe { CStr::from_ptr(config_json).to_string_lossy() };
    let config: PrivacyConfig = match serde_json::from_str(&config_str) {
        Ok(c) => c,
        Err(_) => return std::ptr::null_mut(),
    };
    Box::into_raw(Box::new(PrivacyEngine::new(config)))
}

/// Process a frame in-place.
/// frame_data: raw RGBA bytes, len = height * stride
#[no_mangle]
pub extern "C" fn chronoscope_privacy_process_frame(
    engine: *mut PrivacyEngine,
    frame_data: *mut u8,
    width: u32,
    height: u32,
    stride: u32,
) {
    // SAFETY: We check for null before dereferencing engine or frame_data.
    // The caller must pass a valid engine pointer returned by chronoscope_privacy_init
    // and a valid frame_data buffer with length >= height * stride.
    if engine.is_null() || frame_data.is_null() {
        return;
    }
    const MAX_DIMENSION: u32 = 16384;
    if width == 0 || height == 0 || stride == 0 || width > MAX_DIMENSION || height > MAX_DIMENSION {
        return;
    }
    let len = usize::from(height)
        .checked_mul(usize::from(stride))
        .expect("frame dimension overflow");
    if len == 0 {
        return;
    }
    let engine = unsafe { &mut *engine };
    let frame = unsafe { std::slice::from_raw_parts_mut(frame_data, len) };
    engine.process_frame(frame, width, height);
}

/// Process text and return redacted version.
/// Caller must free the returned string with chronoscope_privacy_free_string.
#[no_mangle]
pub extern "C" fn chronoscope_privacy_process_text(
    engine: *mut PrivacyEngine,
    text: *const c_char,
) -> *mut c_char {
    // SAFETY: We check for null before dereferencing engine or text.
    // The caller must pass a valid engine pointer and a valid null-terminated C string.
    if engine.is_null() || text.is_null() {
        return std::ptr::null_mut();
    }
    let engine = unsafe { &mut *engine };
    let text_str = unsafe { CStr::from_ptr(text).to_string_lossy() };
    let result = engine.process_text(&text_str);
    // SAFETY: CString::new returns Err only if result contains interior null bytes.
    // We map the error to null_mut() so the C caller can check for failure.
    CString::new(result).map(|s| s.into_raw()).unwrap_or(std::ptr::null_mut())
}

/// Free a string returned by chronoscope_privacy_process_text.
#[no_mangle]
pub extern "C" fn chronoscope_privacy_free_string(s: *mut c_char) {
    // SAFETY: We check for null before reconstructing the CString.
    // The caller must pass a pointer previously returned by chronoscope_privacy_process_text.
    if !s.is_null() {
        unsafe { let _ = CString::from_raw(s); }
    }
}

/// Free the privacy engine.
#[no_mangle]
pub extern "C" fn chronoscope_privacy_free(engine: *mut PrivacyEngine) {
    // SAFETY: We check for null before reconstructing the Box.
    // The caller must pass a pointer previously returned by chronoscope_privacy_init.
    if !engine.is_null() {
        unsafe { let _ = Box::from_raw(engine); }
    }
}
