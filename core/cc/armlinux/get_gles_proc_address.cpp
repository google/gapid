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

#include <string>
#include <unordered_map>

#define SYSTEM_LIB_PATH "/usr/lib/"

namespace {

void* ResolveSymbol(const char* name, bool bypassLocal) {
  using namespace core;
  typedef void* (*GPAPROC)(const char* name);

  if (bypassLocal) {
    static DlLoader libegl(SYSTEM_LIB_PATH "libEGL.so");
    if (void* proc = libegl.lookup(name)) {
      GAPID_DEBUG("GetGlesProcAddress(%s, %d) -> 0x%x (from libEGL dlsym)",
                  name, bypassLocal, proc);
      return proc;
    }

    static DlLoader libglesv2(SYSTEM_LIB_PATH "libGLESv2.so");
    if (void* proc = libglesv2.lookup(name)) {
      GAPID_DEBUG("GetGlesProcAddress(%s, %d) -> 0x%x (from libGLESv2 dlsym)",
                  name, bypassLocal, proc);
      return proc;
    }

    static DlLoader libglesv1(SYSTEM_LIB_PATH "libGLESv1_CM.so");
    if (void* proc = libglesv1.lookup(name)) {
      GAPID_DEBUG(
          "GetGlesProcAddress(%s, %d) -> 0x%x (from libGLESv1_CM dlsym)", name,
          bypassLocal, proc);
      return proc;
    }

    if (GPAPROC gpa =
            reinterpret_cast<GPAPROC>(libegl.lookup("eglGetProcAddress"))) {
      if (void* proc = gpa(name)) {
        static DlLoader local(nullptr);
        void* local_proc = local.lookup(name);
        if (local_proc == proc) {
          GAPID_WARNING(
              "libEGL eglGetProcAddress returned a local address %p for %s, "
              "this will be ignored",
              proc, name);
        } else {
          GAPID_DEBUG(
              "GetGlesProcAddress(%s, %d) -> 0x%x "
              "(via libEGL eglGetProcAddress)",
              name, bypassLocal, proc);
          return proc;
        }
      }
    }
  } else {
    static DlLoader local(nullptr);
    if (GPAPROC gpa =
            reinterpret_cast<GPAPROC>(local.lookup("eglGetProcAddress"))) {
      if (void* proc = gpa(name)) {
        GAPID_DEBUG(
            "GetGlesProcAddress(%s, %d) -> 0x%x (via local eglGetProcAddress)",
            name, bypassLocal, proc);
        return proc;
      }
    }
    if (void* proc = local.lookup(name)) {
      GAPID_DEBUG("GetGlesProcAddress(%s, %d) -> 0x%x (from local dlsym)", name,
                  bypassLocal, proc);
      return proc;
    }
  }

  GAPID_DEBUG("GetGlesProcAddress(%s, %d) -> not found", name, bypassLocal);
  return nullptr;
}

void* getGlesProcAddress(const char* name, bool bypassLocal) {
  static std::unordered_map<std::string, void*> cache;

  const std::string cacheKey =
      std::string(name) + (bypassLocal ? "/direct" : "/local");
  auto it = cache.find(cacheKey);
  if (it != cache.end()) {
    GAPID_DEBUG("GetGlesProcAddress(%s, %d) -> 0x%x (from cache)", name,
                bypassLocal, it->second);
    return it->second;
  }

  void* proc = ResolveSymbol(name, bypassLocal);
  cache[cacheKey] = proc;
  return proc;
}

}  // anonymous namespace

namespace core {
bool hasGLorGLES() {
  return DlLoader::can_load(SYSTEM_LIB_PATH "libEGL.so") ||
         DlLoader::can_load(SYSTEM_LIB_PATH "libGLESv2.so") ||
         DlLoader::can_load(SYSTEM_LIB_PATH SYSTEM_LIB_PATH "libGLESv1_CM.so");
}

GetGlesProcAddressFunc* GetGlesProcAddress = getGlesProcAddress;

}  // namespace core
