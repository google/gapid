/*
 * Copyright (C) 2018 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "core/cc/target.h"

#include "surface.h"

#if TARGET_OS == GAPID_OS_WINDOWS
#include <Windows.h>

#include <condition_variable>
#include <mutex>
#include <thread>
#endif

namespace gapir {

#if TARGET_OS == GAPID_OS_ANDROID
ANativeWindow* android_window;
#elif TARGET_OS == GAPID_OS_LINUX
static XcbWindowInfo window_info;

void* createXcbWindow(uint32_t width, uint32_t height) {
  window_info.connection = xcb_connect(nullptr, nullptr);
  if (!window_info.connection) {
    return nullptr;
  }

  xcb_screen_t* screen =
      xcb_setup_roots_iterator(xcb_get_setup(window_info.connection)).data;
  if (!screen) {
    return nullptr;
  }

  window_info.window = xcb_generate_id(window_info.connection);

  xcb_create_window(window_info.connection, XCB_COPY_FROM_PARENT,
                    window_info.window, screen->root, 0, 0, width, height, 1,
                    XCB_WINDOW_CLASS_INPUT_OUTPUT, screen->root_visual, 0,
                    nullptr);
  xcb_map_window(window_info.connection, window_info.window);
  xcb_flush(window_info.connection);

  return (void*)&window_info;
}
#elif TARGET_OS == GAPID_OS_WINDOWS
static Win32WindowInfo window_info;

static HANDLE window_create_sem;
static std::mutex quit_lock;
static std::condition_variable quit_condition;
static bool quit;
static HANDLE thread;

LRESULT windowProc(HWND hwnd, UINT uMsg, WPARAM wParam, LPARAM lParam) {
  switch (uMsg) {
    case WM_CLOSE:
      PostQuitMessage(0);
      return 0;
      break;
  }
  return DefWindowProc(hwnd, uMsg, wParam, lParam);
}

bool createWindow(uint32_t width, uint32_t height) {
  window_info.instance = GetModuleHandle(nullptr);

  WNDCLASS wndclass = {
      CS_HREDRAW | CS_VREDRAW,             // style
      windowProc,                          // lpfnWindowProc
      0,                                   // cbClsExtra
      0,                                   // cbWndExtra
      window_info.instance,                // hInstance
      LoadIcon(nullptr, IDI_APPLICATION),  // hIcon
      LoadCursor(nullptr, IDC_ARROW),      // hCursor
      (HBRUSH)(COLOR_BACKGROUND + 1),      // hbrBackground
      "",                                  // lpszMenuName
      "GAPID Replay",                      // lpszClassName
  };
  ATOM cls = RegisterClass(&wndclass);
  if (cls == 0) {
    // Class registration failed
    return false;
  }

  window_info.window = CreateWindow(
      MAKEINTATOM(cls), "GAPID Replay",
      WS_BORDER | WS_CAPTION | WS_GROUP | WS_OVERLAPPED | WS_POPUP |
          WS_SYSMENU | WS_TILED | WS_VISIBLE,
      0, 0, width, height, nullptr, nullptr, window_info.instance, nullptr);
  return (bool)window_info.window;
}

DWORD handleWindow(void* data) {
  auto extent = (const uint32_t*)data;
  bool res = createWindow(extent[0], extent[1]);
  ReleaseSemaphore(window_create_sem, 1, nullptr);
  if (!res) {
    return 1;
  }

  MSG msg;
  while (GetMessage(&msg, window_info.window, 0, 0)) {
    TranslateMessage(&msg);
    DispatchMessage(&msg);
  }
  {
    std::unique_lock<std::mutex> guard(quit_lock);
    quit = true;
    quit_condition.notify_all();
  }
  return 0;
}

void* createWin32Window(uint32_t width, uint32_t height) {
  window_create_sem = CreateSemaphore(NULL, 0, 1, NULL);

  uint32_t extent[] = {width, height};
  thread = CreateThread(NULL, 0, handleWindow, (void*)extent, 0, nullptr);
  WaitForSingleObject(window_create_sem, INFINITE);
  return window_info.window ? (void*)&window_info : nullptr;
}
#endif

void* CreateSurface(uint32_t width, uint32_t height) {
#if TARGET_OS == GAPID_OS_ANDROID
  return (void*)android_window;
#elif TARGET_OS == GAPID_OS_LINUX
  return createXcbWindow(width, height);
#elif TARGET_OS == GAPID_OS_WINDOWS
  return createWin32Window(width, height);
#endif
  return nullptr;
}

void WaitForWindowClose() {
#if TARGET_OS == GAPID_OS_WINDOWS
  {
    std::unique_lock<std::mutex> guard(quit_lock);
    while (!quit) {
      quit_condition.wait(guard);
    }
  }
#endif
}

}  // namespace gapir
