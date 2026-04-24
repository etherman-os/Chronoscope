#pragma once
#include <cstdint>
#include <vector>
#include <string>
#include <functional>

namespace chronoscope {

struct FrameData {
    std::vector<uint8_t> pixels;
    uint32_t width = 0;
    uint32_t height = 0;
    uint64_t timestamp_us = 0;
};

enum class EventType {
    MouseMove,
    MouseClick,
    KeyPress,
    KeyRelease
};

struct InputEvent {
    EventType type;
    int x = 0;
    int y = 0;
    int key_code = 0;
    uint64_t timestamp_us = 0;
};

using EventCallback = std::function<void(const InputEvent&)>;

} // namespace chronoscope
