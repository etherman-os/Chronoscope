#pragma once
#include <functional>
#include <vector>
#include <windows.h>

namespace chronoscope {

class GraphicsCapture {
public:
    using FrameCallback = std::function<void(const std::vector<uint8_t>&)>;
    explicit GraphicsCapture(int frame_rate);
    ~GraphicsCapture();
    void Start(HWND hwnd, FrameCallback callback);
    void Stop();
private:
    class Impl;
    std::unique_ptr<Impl> pImpl;
};

} // namespace chronoscope
