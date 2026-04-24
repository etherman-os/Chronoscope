#include <gtest/gtest.h>
#include "chronoscope/config.h"
#include "buffer/circular_buffer.h"

TEST(ConfigTest, Defaults) {
    chronoscope::CaptureConfig config;
    config.api_key = "test";
    config.endpoint = "http://localhost:8080";
    EXPECT_EQ(config.mode, chronoscope::CaptureMode::Hybrid);
    EXPECT_EQ(config.frame_rate, 10);
}

TEST(BufferTest, BasicWriteRead) {
    chronoscope::CircularBuffer buffer(1024);
    std::vector<uint8_t> data = {1, 2, 3, 4, 5};
    EXPECT_TRUE(buffer.Write(data));
    auto chunk = buffer.ReadChunk();
    EXPECT_EQ(chunk.size(), 5);
    EXPECT_EQ(chunk[0], 1);
    EXPECT_EQ(chunk[4], 5);
}

TEST(BufferTest, WrapAround) {
    chronoscope::CircularBuffer buffer(10);
    std::vector<uint8_t> data = {1, 2, 3, 4, 5, 6};
    EXPECT_TRUE(buffer.Write(data));
    auto chunk1 = buffer.ReadChunk();
    EXPECT_EQ(chunk1.size(), 6);
    std::vector<uint8_t> data2 = {7, 8, 9, 10, 11};
    EXPECT_TRUE(buffer.Write(data2));
    auto chunk2 = buffer.ReadChunk();
    EXPECT_EQ(chunk2.size(), 5);
}

int main(int argc, char** argv) {
    ::testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
