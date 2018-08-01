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

namespace gapir {

#if TARGET_OS == GAPID_OS_ANDROID
ANativeWindow* android_window;
#elif TARGET_OS == GAPID_OS_LINUX
static XcbWindowInfo window_info;

bool createXcbWindow(uint32_t width, uint32_t height) {
  window_info.connection = xcb_connect(nullptr, nullptr);
  if (!window_info.connection) {
    // Signal failure
    return false;
  }

  xcb_screen_t* screen =
      xcb_setup_roots_iterator(xcb_get_setup(window_info.connection)).data;
  if (!screen) {
    return false;
  }

  window_info.window = xcb_generate_id(window_info.connection);

  xcb_create_window(window_info.connection, XCB_COPY_FROM_PARENT,
                    window_info.window, screen->root, 0, 0, width, height, 1,
                    XCB_WINDOW_CLASS_INPUT_OUTPUT, screen->root_visual, 0,
                    nullptr);
  xcb_map_window(window_info.connection, window_info.window);
  xcb_flush(window_info.connection);

  return true;
}
#endif

void* CreateSurface(uint32_t width, uint32_t height) {
#if TARGET_OS == GAPID_OS_ANDROID
  return (void*)android_window;
#elif TARGET_OS == GAPID_OS_LINUX
  // Create window
  if (createXcbWindow(width, height)) {
    return (void*)&window_info;
  }
#endif
  return nullptr;
}

}  // namespace gapir
