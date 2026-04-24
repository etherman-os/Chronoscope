#pragma once
#include <memory>
#include <string>
#include <vector>
#include <cstdint>

namespace chronoscope {

class PrivacyFilter {
public:
    PrivacyFilter();
    ~PrivacyFilter();
    
    void Initialize(const std::string& config_json);
    void ProcessFrame(uint8_t* frame_data, uint32_t width, uint32_t height, uint32_t stride);
    std::string ProcessText(const std::string& text);
    void AddExcludedWindow(const std::string& title);
    bool ShouldRedact(const std::string& window_title) const;
    
private:
    class Impl;
    std::unique_ptr<Impl> pImpl;
};

} // namespace chronoscope
