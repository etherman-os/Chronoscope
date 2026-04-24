#include "capture/graphics_capture.h"

#include <windows.h>
#include <d3d11.h>
#include <dxgi1_2.h>
#include <winrt/Windows.Graphics.Capture.h>
#include <winrt/Windows.Graphics.DirectX.h>
#include <winrt/Windows.Graphics.DirectX.Direct3D11.h>

#include <memory>
#include <vector>
#include <functional>
#include <mutex>

namespace chronoscope {

using namespace winrt::Windows::Graphics::Capture;
using namespace winrt::Windows::Graphics::DirectX;
using namespace winrt::Windows::Graphics::DirectX::Direct3D11;

// =============================================================================
// GraphicsCapture::Impl
// =============================================================================
class GraphicsCapture::Impl {
public:
    explicit Impl(int frame_rate) : frame_rate_(frame_rate), running_(false) {}
    ~Impl() { Stop(); }

    void Start(HWND hwnd, FrameCallback callback) {
        if (running_.exchange(true)) return;

        hwnd_ = hwnd;
        callback_ = callback;

        // TODO: Initialize D3D11 device
        // ID3D11Device* d3d_device = nullptr;
        // D3D11CreateDevice(..., &d3d_device, ...);
        // IDirect3DDevice* device = CreateDirect3DDevice(d3d_device);

        // TODO: Create capture item from HWND via IGraphicsCaptureItemInterop
        // auto interop = get_activation_factory<GraphicsCaptureItem, IGraphicsCaptureItemInterop>();
        // GraphicsCaptureItem item{ nullptr };
        // interop->CreateForWindow(hwnd_, winrt::guid_of<ABI::Windows::Graphics::Capture::IGraphicsCaptureItem>(),
        //     reinterpret_cast<void**>(winrt::put_abi(item)));

        // TODO: Create frame pool
        // auto frame_pool = Direct3D11CaptureFramePool::Create(
        //     device, DirectXPixelFormat::B8G8R8A8UIntNormalized, 2, item.Size());

        // TODO: Register frame arrived callback
        // frame_arrived_ = frame_pool.FrameArrived([this](auto&, auto&) {
        //     auto frame = frame_pool.TryGetNextFrame();
        //     // Convert D3D texture to bitmap and encode to JPEG
        //     // std::vector<uint8_t> jpeg = EncodeFrameToJpeg(frame);
        //     // if (callback_) callback_(jpeg);
        // });

        // TODO: Create and start session
        // session_ = frame_pool.CreateCaptureSession(item);
        // session_.IsCursorCaptureEnabled(false);
        // session_.StartCapture();
    }

    void Stop() {
        if (!running_.exchange(false)) return;
        // TODO: Close session and frame pool
        // session_.Close();
        // frame_pool_.Close();
    }

private:
    int frame_rate_;
    std::atomic<bool> running_;
    HWND hwnd_ = nullptr;
    FrameCallback callback_;

    // TODO: Store WinRT objects as members
    // GraphicsCaptureSession session_{ nullptr };
    // Direct3D11CaptureFramePool frame_pool_{ nullptr };
    // winrt::event_token frame_arrived_;
};

// =============================================================================
// GraphicsCapture
// =============================================================================
GraphicsCapture::GraphicsCapture(int frame_rate)
    : pImpl(std::make_unique<Impl>(frame_rate)) {
}

GraphicsCapture::~GraphicsCapture() = default;

void GraphicsCapture::Start(HWND hwnd, FrameCallback callback) {
    pImpl->Start(hwnd, callback);
}

void GraphicsCapture::Stop() {
    pImpl->Stop();
}

} // namespace chronoscope
