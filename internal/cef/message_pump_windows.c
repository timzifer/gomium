//go:build windows

#include <windows.h>

// cAppPumpWindowsMessages drains the Win32 message queue of the calling
// thread. Returns 1 when it stopped at |budget| with messages possibly left,
// 0 when the queue was empty or the drain handed control back to the host.
//
// This is the half of the message loop cef_do_message_loop_work does not do.
// With external_message_pump CEF installs MessagePumpExternal, whose Run()
// only drives Chromium's task queues - it never calls PeekMessage. cefclient
// keeps its own GetMessage/DispatchMessage loop on the UI thread for exactly
// that reason. gomium's host loop is a Go select, so without this nothing
// services the HWNDs CEF creates on the UI thread: a windowed browser gets
// "not responding" and never closes, and even an offscreen host leaves
// Chromium's hidden top-level windows deaf - any SendMessage broadcast from
// another application then blocks on us.
//
// Messages are only taken from the calling thread's queue, so this must run
// on the thread that called cef_initialize (which is where the pump worker
// runs anyway).
//
// Two kinds of message are not ours to consume:
//   - WM_QUIT belongs to the host's loop; it is re-posted and the drain ends.
//   - thread messages (hwnd == NULL) cannot be dispatched to a window
//     procedure; a host that pumps itself (GLFW, a Win32 loop) may expect
//     them. They are re-posted to the end of the queue and the drain ends,
//     so the host gets its turn and the queue still makes progress.
int cAppPumpWindowsMessages(int budget) {
    MSG msg;
    for (int i = 0; i < budget; i++) {
        if (!PeekMessageW(&msg, NULL, 0, 0, PM_REMOVE)) {
            return 0;
        }
        if (msg.message == WM_QUIT) {
            PostQuitMessage((int)msg.wParam);
            return 0;
        }
        if (msg.hwnd == NULL) {
            PostThreadMessageW(GetCurrentThreadId(), msg.message, msg.wParam, msg.lParam);
            return 0;
        }
        TranslateMessage(&msg);
        DispatchMessageW(&msg);
    }
    return 1;
}
