#pragma once
#include "config.h"
#include "types.h"
#include <memory>
#include <vector>
#include <functional>

#ifdef CHRONOSCOPE_EXPORTS
#define CHRONOSCOPE_API __declspec(dllexport)
#else
#define CHRONOSCOPE_API __declspec(dllimport)
#endif

namespace chronoscope {

using FrameCallback = std::function<void(const std::vector<uint8_t>&)>;

class CHRONOSCOPE_API Session {
public:
    explicit Session(const CaptureConfig& config);
    ~Session();
    void Start(HWND hwnd, FrameCallback callback);
    void Stop();
    void Pause();
    void SetPrivacyFilter(const std::vector<std::string>& excluded_window_titles);
private:
    class Impl;
    std::unique_ptr<Impl> pImpl;
};

class CHRONOSCOPE_API Chronoscope {
public:
    static Chronoscope& Instance();
    std::shared_ptr<Session> StartSession(const CaptureConfig& config, HWND hwnd, FrameCallback callback);
    void StopAllSessions();
private:
    Chronoscope() = default;
    std::vector<std::shared_ptr<Session>> sessions_;
};

} // namespace chronoscope
