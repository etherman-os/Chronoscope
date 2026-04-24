#include "buffer/circular_buffer.h"

#include <algorithm>
#include <cstring>

namespace chronoscope {

CircularBuffer::CircularBuffer(size_t capacity_bytes)
    : capacity_(capacity_bytes) {
    buffer_.resize(capacity_bytes);
}

CircularBuffer::~CircularBuffer() = default;

bool CircularBuffer::Write(const std::vector<uint8_t>& data) {
    if (data.empty()) return false;

    std::lock_guard<std::mutex> lock(mutex_);
    size_t len = data.size();
    if (len > capacity_) return false; // Too large to ever fit

    // If not enough contiguous space, wrap around (overwrite old data)
    for (size_t i = 0; i < len; ++i) {
        buffer_[write_pos_] = data[i];
        write_pos_ = (write_pos_ + 1) % capacity_;
    }

    available_ = std::min(available_ + len, capacity_);
    cv_.notify_one();
    return true;
}

std::vector<uint8_t> CircularBuffer::ReadChunk() {
    std::lock_guard<std::mutex> lock(mutex_);
    if (available_ == 0) return {};

    size_t to_read = std::min(available_, static_cast<size_t>(65536)); // 64KB chunks
    std::vector<uint8_t> result;
    result.reserve(to_read);

    for (size_t i = 0; i < to_read; ++i) {
        result.push_back(buffer_[read_pos_]);
        read_pos_ = (read_pos_ + 1) % capacity_;
    }

    available_ -= to_read;
    return result;
}

void CircularBuffer::Clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    write_pos_ = 0;
    read_pos_ = 0;
    available_ = 0;
}

} // namespace chronoscope
