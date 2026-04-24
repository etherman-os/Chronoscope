cmake_minimum_required(VERSION 3.20)
project(ChronoscopeWindowsSDK VERSION 0.1.0 LANGUAGES CXX)
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Windows SDK components
add_library(ChronoscopeSDK SHARED
    src/sdk.cpp
    src/capture/graphics_capture.cpp
    src/input/input_hook.cpp
    src/buffer/circular_buffer.cpp
    src/network/uploader.cpp
    src/privacy/privacy_filter.cpp
)

target_include_directories(ChronoscopeSDK PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(ChronoscopeSDK
    d3d11
    dxgi
    windowsapp
    winhttp
)

# Privacy Engine static library (built by Rust)
set(PRIVACY_ENGINE_LIB "${CMAKE_CURRENT_SOURCE_DIR}/../../services/privacy-engine/target/x86_64-pc-windows-msvc/release/chronoscope_privacy.lib")

# Check if library exists (for local dev, it won't on Linux)
if(EXISTS ${PRIVACY_ENGINE_LIB})
    target_link_libraries(ChronoscopeSDK ${PRIVACY_ENGINE_LIB})
else()
    message(WARNING "Privacy engine library not found at ${PRIVACY_ENGINE_LIB}. Build it first with: cargo build --release --target x86_64-pc-windows-msvc")
endif()

# Also add the include directory for the C header
target_include_directories(ChronoscopeSDK PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../../services/privacy-engine/include
)

target_compile_definitions(ChronoscopeSDK PRIVATE CHRONOSCOPE_EXPORTS)

# Tests
enable_testing()
find_package(GTest)
if(GTest_FOUND)
    add_executable(ChronoscopeTests tests/test_capture.cpp)
    target_link_libraries(ChronoscopeTests ChronoscopeSDK GTest::gtest_main)
    add_test(NAME CaptureTests COMMAND ChronoscopeTests)
endif()
