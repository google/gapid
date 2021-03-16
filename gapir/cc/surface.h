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

#ifndef GAPIR_SURFACE_H
#define GAPIR_SURFACE_H

#include <cstdint>

#include "core/cc/target.h"

#if TARGET_OS == GAPID_OS_ANDROID
class ANativeWindow;
#elif TARGET_OS == GAPID_OS_LINUX
#include <xcb/xcb.h>
#elif TARGET_OS == GAPID_OS_WINDOWS
#include <Windows.h>
#endif

namespace gapir {

#if TARGET_OS == GAPID_OS_ANDROID
extern ANativeWindow* android_window;
#elif TARGET_OS == GAPID_OS_LINUX
struct XcbWindowInfo {
  xcb_connection_t* connection;
  xcb_window_t window;
};
#elif TARGET_OS == GAPID_OS_WINDOWS
struct Win32WindowInfo {
  HINSTANCE instance;
  HWND window;
};
#endif

enum SurfaceType { Unknown, Android, Win32, Xcb };

// Get the platform-specific data pointer to create the surface
const void* CreateSurface(uint32_t width, uint32_t height, SurfaceType& type);

void WaitForWindowClose();

}  // namespace gapir

#endif  // GAPIR_SURFACE_H
