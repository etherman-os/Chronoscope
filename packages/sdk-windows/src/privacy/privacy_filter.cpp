#include "privacy_filter.h"
#include "../../services/privacy-engine/include/chronoscope_privacy.h"
#include <memory>
#include <string>
#include <vector>

namespace chronoscope {

class PrivacyFilter::Impl {
public:
    Impl() : engine_(nullptr) {}
    
    ~Impl() {
        if (engine_) {
            chronoscope_privacy_free(engine_);
        }
    }
    
    void Initialize(const std::string& config_json) {
        if (engine_) {
            chronoscope_privacy_free(engine_);
        }
        engine_ = chronoscope_privacy_init(config_json.c_str());
    }
    
    void ProcessFrame(uint8_t* frame_data, uint32_t width, uint32_t height, uint32_t stride) {
        if (!engine_) return;
        chronoscope_privacy_process_frame(engine_, frame_data, width, height, stride);
    }
    
    std::string ProcessText(const std::string& text) {
        if (!engine_) return text;
        char* result = chronoscope_privacy_process_text(engine_, text.c_str());
        if (!result) return text;
        std::string output(result);
        chronoscope_privacy_free_string(result);
        return output;
    }
    
    void AddExcludedWindow(const std::string& title) {
        excluded_windows_.push_back(title);
    }
    
    bool ShouldRedact(const std::string& window_title) const {
        for (const auto& excluded : excluded_windows_) {
            if (window_title.find(excluded) != std::string::npos) {
                return true;
            }
        }
        return false;
    }
    
private:
    ChronoscopePrivacyEngine* engine_;
    std::vector<std::string> excluded_windows_;
};

PrivacyFilter::PrivacyFilter() : pImpl_(std::make_unique<Impl>()) {}
PrivacyFilter::~PrivacyFilter() = default;

void PrivacyFilter::Initialize(const std::string& config_json) {
    pImpl_->Initialize(config_json);
}

void PrivacyFilter::ProcessFrame(uint8_t* frame_data, uint32_t width, uint32_t height, uint32_t stride) {
    pImpl_->ProcessFrame(frame_data, width, height, stride);
}

std::string PrivacyFilter::ProcessText(const std::string& text) {
    return pImpl_->ProcessText(text);
}

void PrivacyFilter::AddExcludedWindow(const std::string& title) {
    pImpl_->AddExcludedWindow(title);
}

bool PrivacyFilter::ShouldRedact(const std::string& window_title) const {
    return pImpl_->ShouldRedact(window_title);
}

} // namespace chronoscope
