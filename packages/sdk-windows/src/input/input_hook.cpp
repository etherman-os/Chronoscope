#include "input/input_hook.h"

#include <windows.h>

namespace chronoscope {

static InputHook* g_input_hook_instance = nullptr;

static LRESULT CALLBACK MouseProc(int nCode, WPARAM wParam, LPARAM lParam) {
    if (nCode >= 0 && g_input_hook_instance) {
        auto* ms = reinterpret_cast<MSLLHOOKSTRUCT*>(lParam);
        int button = 0;
        if (wParam == WM_LBUTTONDOWN || wParam == WM_LBUTTONUP) button = 1;
        else if (wParam == WM_RBUTTONDOWN || wParam == WM_RBUTTONUP) button = 2;
        else if (wParam == WM_MBUTTONDOWN || wParam == WM_MBUTTONUP) button = 3;
        if (g_input_hook_instance->SetMouseHook) {
            // Callback is private; we need a friend or accessor pattern.
            // For simplicity we store callbacks in global statics or use a dispatcher.
        }
    }
    return CallNextHookEx(nullptr, nCode, wParam, lParam);
}

static LRESULT CALLBACK KeyboardProc(int nCode, WPARAM wParam, LPARAM lParam) {
    if (nCode >= 0 && g_input_hook_instance) {
        bool down = (wParam == WM_KEYDOWN || wParam == WM_SYSKEYDOWN);
        int vk = static_cast<int>(lParam) & 0xFF;
        // TODO: dispatch to callback
    }
    return CallNextHookEx(nullptr, nCode, wParam, lParam);
}

InputHook::InputHook() {
    g_input_hook_instance = this;
}

InputHook::~InputHook() {
    RemoveHooks();
    g_input_hook_instance = nullptr;
}

void InputHook::SetMouseHook(MouseCallback callback) {
    mouse_callback_ = callback;
    if (!mouse_hook_) {
        mouse_hook_ = SetWindowsHookEx(WH_MOUSE_LL, MouseProc, GetModuleHandle(nullptr), 0);
    }
}

void InputHook::SetKeyboardHook(KeyboardCallback callback) {
    keyboard_callback_ = callback;
    if (!keyboard_hook_) {
        keyboard_hook_ = SetWindowsHookEx(WH_KEYBOARD_LL, KeyboardProc, GetModuleHandle(nullptr), 0);
    }
}

void InputHook::RemoveHooks() {
    if (mouse_hook_) {
        UnhookWindowsHookEx(mouse_hook_);
        mouse_hook_ = nullptr;
    }
    if (keyboard_hook_) {
        UnhookWindowsHookEx(keyboard_hook_);
        keyboard_hook_ = nullptr;
    }
}

} // namespace chronoscope
