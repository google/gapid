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

#include <dlfcn.h>

namespace {

void* getGlesProcAddress(const char* name, bool bypassLocal) {
  using namespace core;
  typedef void* (*GPAPROC)(const char* name);

  // The mesa driver does bad things with LLVM. Since we also use llvm,
  // we can't have the mesa driver do bad things to our code.
  // Therefore we should preload any versions of llvm that may be required
  // into the start of our address space.
  // See: https://github.com/google/gapid/issues/1707 for more information
  static void* _dummy0 = dlopen("libLLVM-3.0.so.1", RTLD_LAZY | RTLD_DEEPBIND);
  static void* _dummy1 = dlopen("libLLVM-4.0.so.1", RTLD_LAZY | RTLD_DEEPBIND);
  static void* _dummy2 = dlopen("libLLVM-5.0.so.1", RTLD_LAZY | RTLD_DEEPBIND);
  (void)_dummy0;
  (void)_dummy1;
  (void)_dummy2;

  if (bypassLocal) {
    // Why .1 ?
    // See: https://bugs.launchpad.net/ubuntu/+source/python-qt4/+bug/941826
    static DlLoader libgl("libGL.so.1");
    if (GPAPROC gpa =
            reinterpret_cast<GPAPROC>(libgl.lookup("glXGetProcAddress"))) {
      if (void* proc = gpa(name)) {
        GAPID_VERBOSE(
            "GetGlesProcAddress(%s, %d) -> 0x%x (via libGL glXGetProcAddress)",
            name, bypassLocal, proc);
        return proc;
      }
    }
    if (void* proc = libgl.lookup(name)) {
      GAPID_VERBOSE("GetGlesProcAddress(%s, %d) -> 0x%x (from libGL dlsym)",
                    name, bypassLocal, proc);
      return proc;
    }
  } else {
    static DlLoader local(nullptr);
    if (GPAPROC gpa =
            reinterpret_cast<GPAPROC>(local.lookup("glXGetProcAddress"))) {
      if (void* proc = gpa(name)) {
        GAPID_VERBOSE(
            "GetGlesProcAddress(%s, %d) -> 0x%x (via local glXGetProcAddress)",
            name, bypassLocal, proc);
        return proc;
      }
    }
    if (void* proc = local.lookup(name)) {
      GAPID_VERBOSE("GetGlesProcAddress(%s, %d) -> 0x%x (from local dlsym)",
                    name, bypassLocal, proc);
      return proc;
    }
  }

  GAPID_DEBUG("GetGlesProcAddress(%s, %d) -> not found", name, bypassLocal);
  return nullptr;
}

}  // anonymous namespace

namespace core {

GetGlesProcAddressFunc* GetGlesProcAddress = getGlesProcAddress;
bool hasGLorGLES() { return DlLoader::can_load("libGL.so.1"); }

}  // namespace core
