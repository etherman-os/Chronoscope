use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use crate::{PrivacyEngine, PrivacyConfig};

/// Initialize privacy engine from JSON config string.
/// Returns opaque pointer. Must be freed with chronoscope_privacy_free.
#[no_mangle]
pub extern "C" fn chronoscope_privacy_init(config_json: *const c_char) -> *mut PrivacyEngine {
    // SAFETY: caller must pass valid null-terminated string
    let config_str = unsafe { CStr::from_ptr(config_json).to_string_lossy() };
    let config: PrivacyConfig = serde_json::from_str(&config_str).unwrap_or_default();
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
    if engine.is_null() || frame_data.is_null() { return; }
    let engine = unsafe { &mut *engine };
    let len = (height * stride) as usize;
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
    if engine.is_null() || text.is_null() {
        return std::ptr::null_mut();
    }
    let engine = unsafe { &mut *engine };
    let text_str = unsafe { CStr::from_ptr(text).to_string_lossy() };
    let result = engine.process_text(&text_str);
    CString::new(result).map(|s| s.into_raw()).unwrap_or(std::ptr::null_mut())
}

/// Free a string returned by chronoscope_privacy_process_text.
#[no_mangle]
pub extern "C" fn chronoscope_privacy_free_string(s: *mut c_char) {
    if !s.is_null() {
        unsafe { let _ = CString::from_raw(s); }
    }
}

/// Free the privacy engine.
#[no_mangle]
pub extern "C" fn chronoscope_privacy_free(engine: *mut PrivacyEngine) {
    if !engine.is_null() {
        unsafe { let _ = Box::from_raw(engine); }
    }
}
