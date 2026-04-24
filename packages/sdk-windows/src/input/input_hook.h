#pragma once
#include <functional>
#include <windows.h>

namespace chronoscope {

using MouseCallback = std::function<void(int x, int y, int button)>;
using KeyboardCallback = std::function<void(int key_code, bool down)>;

class InputHook {
public:
    InputHook();
    ~InputHook();

    void SetMouseHook(MouseCallback callback);
    void SetKeyboardHook(KeyboardCallback callback);
    void RemoveHooks();

private:
    MouseCallback mouse_callback_;
    KeyboardCallback keyboard_callback_;
    HHOOK mouse_hook_ = nullptr;
    HHOOK keyboard_hook_ = nullptr;
};

} // namespace chronoscope
