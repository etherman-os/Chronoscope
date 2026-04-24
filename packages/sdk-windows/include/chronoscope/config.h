#pragma once
#include <string>

namespace chronoscope {

enum class CaptureMode { Video, Events, Hybrid };
enum class CaptureQuality { Low, Medium, High };

struct CaptureConfig {
    std::string api_key;
    std::string endpoint;
    CaptureMode mode = CaptureMode::Hybrid;
    CaptureQuality quality = CaptureQuality::Medium;
    int frame_rate = 10;
    int buffer_size_mb = 100;
};

} // namespace chronoscope
