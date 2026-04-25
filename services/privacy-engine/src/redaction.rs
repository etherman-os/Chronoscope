use crate::detector::Detection;
use crate::RedactionMode;

pub fn apply(
    frame: &mut [u8],
    width: u32,
    height: u32,
    detection: &Detection,
    mode: &RedactionMode,
) {
    // For frame redaction:
    //   Blackout: set pixel RGB to 0 (keep alpha)
    //   Blur: simple box blur on detected region
    //   Replace: same as Blackout for frames (text overlay not supported)

    let max_len = (width as usize)
        .checked_mul(height as usize)
        .and_then(|v| v.checked_mul(4))
        .unwrap_or(frame.len())
        .min(frame.len());
    if detection.start >= max_len {
        return;
    }

    match mode {
        RedactionMode::Blackout | RedactionMode::Replace(_) => {
            for i in (detection.start..detection.end).step_by(4) {
                if i + 3 < frame.len() {
                    frame[i] = 0; // R
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

                    if let Some(div_r) = r.checked_div(count) {
                        frame[i] = div_r as u8;
                        frame[i + 1] = g.checked_div(count).unwrap_or(0) as u8;
                        frame[i + 2] = b.checked_div(count).unwrap_or(0) as u8;
                    }
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::detector::DetectionType;

    #[test]
    fn test_redaction_blackout() {
        let mut frame = vec![255, 255, 255, 255, 255, 255, 255, 255];
        let detection = Detection {
            start: 0,
            end: 8,
            detection_type: DetectionType::Email,
            confidence: 1.0,
        };
        apply(&mut frame, 2, 1, &detection, &RedactionMode::Blackout);
        assert_eq!(frame, vec![0, 0, 0, 255, 0, 0, 0, 255]);
    }

    #[test]
    fn test_redaction_bounds_check() {
        let mut frame = vec![255, 255, 255, 255];
        let detection = Detection {
            start: 8, // out of bounds for a 1x1 frame
            end: 16,
            detection_type: DetectionType::Email,
            confidence: 1.0,
        };
        apply(&mut frame, 1, 1, &detection, &RedactionMode::Blackout);
        // Frame should remain unchanged
        assert_eq!(frame, vec![255, 255, 255, 255]);
    }

    #[test]
    fn test_redaction_blur_does_not_panic() {
        let mut frame = vec![
            10, 20, 30, 255, 40, 50, 60, 255, 70, 80, 90, 255, 100, 110, 120, 255,
        ];
        let detection = Detection {
            start: 0,
            end: 16,
            detection_type: DetectionType::CreditCard,
            confidence: 1.0,
        };
        apply(&mut frame, 2, 2, &detection, &RedactionMode::Blur);
        // Just ensure it doesn't panic and alpha is preserved
        assert_eq!(frame[3], 255);
        assert_eq!(frame[7], 255);
    }
}
