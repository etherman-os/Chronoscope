use crate::detector::Detection;
use crate::RedactionMode;

pub fn apply(frame: &mut [u8], width: u32, _height: u32, detection: &Detection, mode: &RedactionMode) {
    // For frame redaction:
    //   Blackout: set pixel RGB to 0 (keep alpha)
    //   Blur: simple box blur on detected region
    //   Replace: same as Blackout for frames (text overlay not supported)

    match mode {
        RedactionMode::Blackout | RedactionMode::Replace(_) => {
            for i in (detection.start..detection.end).step_by(4) {
                if i + 3 < frame.len() {
                    frame[i] = 0;     // R
                    frame[i + 1] = 0; // G
                    frame[i + 2] = 0; // B
                    // keep alpha at i + 3
                }
            }
        }
        RedactionMode::Blur => {
            let stride = (width * 4) as usize;
            let source = frame.to_vec();
            for i in (detection.start..detection.end).step_by(4) {
                if i + 3 < frame.len() {
                    let mut r = 0u32;
                    let mut g = 0u32;
                    let mut b = 0u32;
                    let mut count = 0u32;

                    // Simple 3x3 blur approximation
                    for dy in -1..=1 {
                        for dx in -1..=1 {
                            let idx = i as isize + dy * stride as isize + dx * 4;
                            if idx >= 0 && (idx as usize) + 3 < source.len() {
                                let idx = idx as usize;
                                r += source[idx] as u32;
                                g += source[idx + 1] as u32;
                                b += source[idx + 2] as u32;
                                count += 1;
                            }
                        }
                    }

                    if count > 0 {
                        frame[i] = (r / count) as u8;
                        frame[i + 1] = (g / count) as u8;
                        frame[i + 2] = (b / count) as u8;
                    }
                }
            }
        }
    }
}
