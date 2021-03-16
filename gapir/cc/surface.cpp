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

#include <condition_variable>
#include <mutex>
#include <thread>

#include "core/cc/dl_loader.h"
#include "core/cc/target.h"

#include "surface.h"

#if TARGET_OS == GAPID_OS_WINDOWS
#include <Windows.h>
#endif

namespace gapir {

namespace {
class Flag {
 public:
  Flag() : set_(false) {}

  void Set() {
    std::unique_lock<std::mutex> guard(mutex_);
    set_ = true;
    condition_.notify_all();
  }

  void Wait() {
    std::unique_lock<std::mutex> guard(mutex_);
    while (!set_) {
      condition_.wait(guard);
    }
  }

 private:
  bool set_;
  std::mutex mutex_;
  std::condition_variable condition_;
};
}  // namespace

#if TARGET_OS == GAPID_OS_ANDROID
ANativeWindow* android_window;
#elif TARGET_OS == GAPID_OS_LINUX
typedef xcb_connection_t* (*pfn_xcb_connect)(const char*, int*);
typedef xcb_screen_iterator_t (*pfn_xcb_setup_roots_iterator)(
    const xcb_setup_t*);
typedef const struct xcb_setup_t* (*pfn_xcb_get_setup)(xcb_connection_t*);
typedef uint32_t (*pfn_xcb_generate_id)(xcb_connection_t*);
typedef xcb_void_cookie_t (*pfn_xcb_create_window)(xcb_connection_t*, uint8_t,
                                                   xcb_window_t, xcb_window_t,
                                                   int16_t, int16_t, uint16_t,
                                                   uint16_t, uint16_t, uint16_t,
                                                   xcb_visualid_t, uint32_t,
                                                   const uint32_t*);
typedef xcb_void_cookie_t (*pfn_xcb_map_window)(xcb_connection_t*,
                                                xcb_window_t);
typedef int (*pfn_xcb_flush)(xcb_connection_t* c);

static XcbWindowInfo window_info;

static Flag window_create_flag;
static std::thread window_thread;

bool createWindow(uint32_t width, uint32_t height) {
  const char* lib_xcb = "libxcb.so.1";
  if (!core::DlLoader::can_load(lib_xcb)) {
    lib_xcb = "libxcb.so";
    if (!core::DlLoader::can_load(lib_xcb)) {
      return false;
    }
  }
  core::DlLoader xcb(lib_xcb);
  pfn_xcb_connect _connect = (pfn_xcb_connect)xcb.lookup("xcb_connect");

  window_info.connection = _connect(nullptr, nullptr);
  if (!window_info.connection) {
    return false;
  }

  pfn_xcb_setup_roots_iterator _setup_roots_iterator =
      (pfn_xcb_setup_roots_iterator)xcb.lookup("xcb_setup_roots_iterator");
  pfn_xcb_get_setup _get_setup = (pfn_xcb_get_setup)xcb.lookup("xcb_get_setup");

  xcb_screen_t* screen =
      _setup_roots_iterator(_get_setup(window_info.connection)).data;
  if (!screen) {
    return false;
  }

  pfn_xcb_generate_id _generate_id =
      (pfn_xcb_generate_id)xcb.lookup("xcb_generate_id");
  window_info.window = _generate_id(window_info.connection);

  pfn_xcb_create_window _create_window =
      (pfn_xcb_create_window)xcb.lookup("xcb_create_window");
  _create_window(window_info.connection, XCB_COPY_FROM_PARENT,
                 window_info.window, screen->root, 0, 0, width, height, 1,
                 XCB_WINDOW_CLASS_INPUT_OUTPUT, screen->root_visual, 0,
                 nullptr);

  pfn_xcb_map_window _map_window =
      (pfn_xcb_map_window)xcb.lookup("xcb_map_window");
  _map_window(window_info.connection, window_info.window);
  pfn_xcb_flush _flush = (pfn_xcb_flush)xcb.lookup("xcb_flush");
  _flush(window_info.connection);

  return true;
}

typedef xcb_intern_atom_cookie_t (*pfn_xcb_intern_atom)(xcb_connection_t*,
                                                        uint8_t, uint16_t,
                                                        const char*);
typedef xcb_intern_atom_reply_t* (*pfn_xcb_intern_atom_reply)(
    xcb_connection_t*, xcb_intern_atom_cookie_t, xcb_generic_error_t**);
typedef xcb_generic_event_t* (*pfn_xcb_wait_for_event)(xcb_connection_t*);

void handleWindow(uint32_t width, uint32_t height) {
  const char* lib_xcb = "libxcb.so.1";
  if (!core::DlLoader::can_load(lib_xcb)) {
    lib_xcb = "libxcb.so";
    if (!core::DlLoader::can_load(lib_xcb)) {
      return;
    }
  }
  core::DlLoader xcb(lib_xcb);

  bool res = createWindow(width, height);
  window_create_flag.Set();
  if (!res) {
    return;
  }

  pfn_xcb_intern_atom _intern_atom =
      (pfn_xcb_intern_atom)xcb.lookup("xcb_intern_atom");
  xcb_intern_atom_cookie_t delete_cookie =
      _intern_atom(window_info.connection, 0, 16, "WM_DELETE_WINDOW");
  pfn_xcb_intern_atom_reply _intern_atom_reply =
      (pfn_xcb_intern_atom_reply)xcb.lookup("xcb_intern_atom_reply");
  xcb_intern_atom_reply_t* delete_reply =
      _intern_atom_reply(window_info.connection, delete_cookie, 0);

  pfn_xcb_wait_for_event _wait_for_event =
      (pfn_xcb_wait_for_event)xcb.lookup("xcb_wait_for_event");
  xcb_generic_event_t* event;
  while ((event = _wait_for_event(window_info.connection))) {
    if ((event->response_type & 0x7f) == XCB_CLIENT_MESSAGE) {
      auto message = (xcb_client_message_event_t*)event;
      if (message->data.data32[0] == delete_reply->atom) {
        break;
      }
    }
  }
}

void* createXcbWindow(uint32_t width, uint32_t height) {
  window_thread = std::thread(handleWindow, width, height);
  window_create_flag.Wait();
  return window_info.window ? (void*)&window_info : nullptr;
}

static const int32_t stream_index = 0;
#elif TARGET_OS == GAPID_OS_WINDOWS
static Win32WindowInfo window_info;

static Flag window_create_flag;
static HANDLE window_thread;

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
  window_create_flag.Set();
  if (!res) {
    return 1;
  }

  MSG msg;
  while (GetMessage(&msg, window_info.window, 0, 0)) {
    TranslateMessage(&msg);
    DispatchMessage(&msg);
  }
  return 0;
}

void* createWin32Window(uint32_t width, uint32_t height) {
  uint32_t extent[] = {width, height};
  window_thread =
      CreateThread(NULL, 0, handleWindow, (void*)extent, 0, nullptr);
  window_create_flag.Wait();
  return window_info.window ? (void*)&window_info : nullptr;
}
#endif

const void* CreateSurface(uint32_t width, uint32_t height, SurfaceType& type) {
  switch (type) {
#if TARGET_OS == GAPID_OS_ANDROID
    case SurfaceType::Android:
    case SurfaceType::Unknown:
      type = SurfaceType::Android;
      return (void*)android_window;
#elif TARGET_OS == GAPID_OS_LINUX
    case SurfaceType::Xcb:
    case SurfaceType::Unknown:
      type = SurfaceType::Xcb;
      return createXcbWindow(width, height);
#elif TARGET_OS == GAPID_OS_WINDOWS
    case SurfaceType::Win32:
    case SurfaceType::Unknown:
      type = SurfaceType::Win32;
      return createWin32Window(width, height);
#endif
    default:
      return nullptr;
  }
}

void WaitForWindowClose() {
#if TARGET_OS == GAPID_OS_WINDOWS
  if (window_thread) {
    WaitForSingleObject(window_thread, INFINITE);
  }
#endif
#if TARGET_OS == GAPID_OS_LINUX
  if (window_thread.joinable()) {
    window_thread.join();
  }
#endif
}

}  // namespace gapir
