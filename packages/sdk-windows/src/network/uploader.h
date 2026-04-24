#pragma once
#include <vector>
#include <cstdint>
#include <string>

namespace chronoscope {

class ChunkUploader {
public:
    ChunkUploader(const std::string& endpoint, const std::string& api_key);
    ~ChunkUploader();

    bool UploadChunk(const std::vector<uint8_t>& data, int index, const std::string& session_id);
    bool Finalize(const std::string& session_id);

private:
    std::string endpoint_;
    std::string api_key_;
};

} // namespace chronoscope
