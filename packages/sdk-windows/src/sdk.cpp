#include "chronoscope/sdk.h"
#include "capture/graphics_capture.h"
#include "buffer/circular_buffer.h"
#include "network/uploader.h"
#include "privacy/privacy_filter.h"
#include "input/input_hook.h"

#include <thread>
#include <mutex>
#include <atomic>

namespace chronoscope {

// =============================================================================
// Session::Impl
// =============================================================================
class Session::Impl {
public:
    explicit Impl(const CaptureConfig& config)
        : config_(config)
        , capture_(config.frame_rate)
        , buffer_(config.buffer_size_mb * 1024 * 1024)
        , uploader_(config.endpoint, config.api_key)
        , running_(false)
        , paused_(false) {
    }

    ~Impl() {
        Stop();
    }

    void Start(HWND hwnd, FrameCallback callback) {
        if (running_.exchange(true)) return;

        hwnd_ = hwnd;
        user_callback_ = callback;

        // Initialize privacy filter with config JSON
        std::string config_json = "{";
        config_json += "\"api_key\":\"" + config_.api_key + "\",";
        config_json += "\"endpoint\":\"" + config_.endpoint + "\",";
        config_json += "\"frame_rate\":" + std::to_string(config_.frame_rate);
        config_json += "}";
        privacy_.Initialize(config_json);

        capture_.Start(hwnd, [this](const std::vector<uint8_t>& frame) {
            if (paused_.load()) return;

            // Get window title for privacy check
            char title[256] = {};
            std::string window_title;
            if (hwnd_) {
                GetWindowTextA(hwnd_, title, sizeof(title));
                window_title = title;
            }

            if (!frame.empty()) {
                // Process frame through privacy engine
                RECT rect = {};
                GetClientRect(hwnd_, &rect);
                uint32_t width = static_cast<uint32_t>(rect.right - rect.left);
                uint32_t height = static_cast<uint32_t>(rect.bottom - rect.top);
                uint32_t stride = width * 4; // Assume BGRA8

                std::vector<uint8_t> processed_frame = frame;
                privacy_.ProcessFrame(processed_frame.data(), width, height, stride);

                if (privacy_.ShouldRedact(window_title)) {
                    // TODO: redact frame by drawing black rect
                }

                buffer_.Write(processed_frame);
                if (user_callback_) {
                    user_callback_(processed_frame);
                }
            }
        });

        upload_thread_ = std::thread([this]() {
            int chunk_index = 0;
            while (running_.load()) {
                auto chunk = buffer_.ReadChunk();
                if (!chunk.empty()) {
                    uploader_.UploadChunk(chunk, chunk_index++, session_id_);
                } else {
                    std::this_thread::sleep_for(std::chrono::milliseconds(16));
                }
            }
            uploader_.Finalize(session_id_);
        });
    }

    void Stop() {
        if (!running_.exchange(false)) return;
        capture_.Stop();
        if (upload_thread_.joinable()) {
            upload_thread_.join();
        }
    }

    void Pause() {
        paused_.store(true);
    }

    void SetPrivacyFilter(const std::vector<std::string>& excluded_window_titles) {
        for (const auto& title : excluded_window_titles) {
            privacy_.AddExcludedWindow(title);
        }
    }

private:
    CaptureConfig config_;
    GraphicsCapture capture_;
    CircularBuffer buffer_;
    ChunkUploader uploader_;
    PrivacyFilter privacy_;
    std::atomic<bool> running_;
    std::atomic<bool> paused_;
    HWND hwnd_ = nullptr;
    FrameCallback user_callback_;
    std::thread upload_thread_;
    std::string session_id_ = "session_0"; // TODO: generate UUID
};

// =============================================================================
// Session
// =============================================================================
Session::Session(const CaptureConfig& config)
    : pImpl(std::make_unique<Impl>(config)) {
}

Session::~Session() = default;

void Session::Start(HWND hwnd, FrameCallback callback) {
    pImpl->Start(hwnd, callback);
}

void Session::Stop() {
    pImpl->Stop();
}

void Session::Pause() {
    pImpl->Pause();
}

void Session::SetPrivacyFilter(const std::vector<std::string>& excluded_window_titles) {
    pImpl->SetPrivacyFilter(excluded_window_titles);
}

// =============================================================================
// Chronoscope
// =============================================================================
Chronoscope& Chronoscope::Instance() {
    static Chronoscope instance;
    return instance;
}

std::shared_ptr<Session> Chronoscope::StartSession(
    const CaptureConfig& config,
    HWND hwnd,
    FrameCallback callback) {

    auto session = std::make_shared<Session>(config);
    session->Start(hwnd, callback);
    sessions_.push_back(session);
    return session;
}

void Chronoscope::StopAllSessions() {
    for (auto& s : sessions_) {
        if (s) s->Stop();
    }
    sessions_.clear();
}

} // namespace chronoscope
