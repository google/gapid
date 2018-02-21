/*
 * Copyright (C) 2017 Google Inc.
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

#include "../get_gles_proc_address.h"
#include "../dl_loader.h"
#include "../log.h"

namespace {

#define FRAMEWORK_ROOT "/System/Library/Frameworks/OpenGL.framework/"
#define CORE_GRAPHICS \
  "/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics"

void* getGlesProcAddress(const char* name) {
  using namespace core;
  static DlLoader opengl(FRAMEWORK_ROOT "OpenGL");
  if (void* proc = opengl.lookup(name)) {
    GAPID_DEBUG("GetGlesProcAddress(%s) -> 0x%x (from OpenGL dlsym)", name,
                proc);
    return proc;
  }

  static DlLoader libgl(FRAMEWORK_ROOT "Libraries/libGL.dylib");
  if (void* proc = libgl.lookup(name)) {
    GAPID_DEBUG("GetGlesProcAddress(%s) -> 0x%x (from libGL dlsym)", name,
                proc);
    return proc;
  }

  static DlLoader libglu(FRAMEWORK_ROOT "Libraries/libGLU.dylib");
  if (void* proc = libglu.lookup(name)) {
    GAPID_DEBUG("GetGlesProcAddress(%s) -> 0x%x (from libGLU dlsym)", name,
                proc);
    return proc;
  }

  static DlLoader coregraphics(CORE_GRAPHICS);
  if (void* proc = coregraphics.lookup(name)) {
    GAPID_DEBUG("GetGlesProcAddress(%s) -> 0x%x (from CoreGraphics dlsym)",
                name, proc);
    return proc;
  }

  return nullptr;
}

}  // anonymous namespace

namespace core {

GetGlesProcAddressFunc* GetGlesProcAddress = getGlesProcAddress;
bool hasGLorGLES() {
  return DlLoader::can_load(FRAMEWORK_ROOT "OpenGL") ||
         DlLoader::can_load(FRAMEWORK_ROOT "Libraries/libGL.dylib") ||
         DlLoader::can_load(FRAMEWORK_ROOT "Libraries/libGLU.dylib") ||
         DlLoader::can_load(CORE_GRAPHICS);
}

}  // namespace core
