#pragma once
#include <vector>
#include <cstdint>
#include <mutex>
#include <condition_variable>

namespace chronoscope {

class CircularBuffer {
public:
    explicit CircularBuffer(size_t capacity_bytes);
    ~CircularBuffer();

    bool Write(const std::vector<uint8_t>& data);
    std::vector<uint8_t> ReadChunk();
    void Clear();

private:
    std::vector<uint8_t> buffer_;
    size_t capacity_;
    size_t write_pos_ = 0;
    size_t read_pos_ = 0;
    size_t available_ = 0;
    std::mutex mutex_;
    std::condition_variable cv_;
};

} // namespace chronoscope
