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

// The mesa driver does bad things with LLVM. Since we also use llvm,
// we can't have the mesa driver do bad things to our code.
// Therefore we should preload any versions of llvm that may be required
// into the start of our address space.
// See: https://github.com/google/gapid/issues/1707 for more information
struct MesaLLVMOpener {
  MesaLLVMOpener() {
    char name[512];
    for (int i = 3; i <= 9; i++) {
      snprintf(name, sizeof(name), "libLLVM-%d.0.so.1", i);
      dlopen(name, RTLD_LAZY | RTLD_DEEPBIND);
      snprintf(name, sizeof(name), "libLLVM-%d.so.1", i);
      dlopen(name, RTLD_LAZY | RTLD_DEEPBIND);
    }
  }
};

void* getGlesProcAddress(const char* name) {
  using namespace core;
  typedef void* (*GPAPROC)(const char* name);
  static MesaLLVMOpener _dummy;
  (void)_dummy;

  // Why .1 ?
  // See: https://bugs.launchpad.net/ubuntu/+source/python-qt4/+bug/941826
  static DlLoader libgl("libGL.so.1");
  if (GPAPROC gpa =
          reinterpret_cast<GPAPROC>(libgl.lookup("glXGetProcAddress"))) {
    if (void* proc = gpa(name)) {
      GAPID_VERBOSE(
          "GetGlesProcAddress(%s) -> 0x%x (via libGL glXGetProcAddress)", name,
          proc);
      return proc;
    }
  }
  if (void* proc = libgl.lookup(name)) {
    GAPID_VERBOSE("GetGlesProcAddress(%s) -> 0x%x (from libGL dlsym)", name,
                  proc);
    return proc;
  }

  GAPID_DEBUG("GetGlesProcAddress(%s) -> not found", name);
  return nullptr;
}

}  // anonymous namespace

namespace core {

GetGlesProcAddressFunc* GetGlesProcAddress = getGlesProcAddress;
bool hasGLorGLES() { return DlLoader::can_load("libGL.so.1"); }

}  // namespace core
