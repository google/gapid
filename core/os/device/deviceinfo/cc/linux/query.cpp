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

#include "../query.h"

#include "core/cc/dl_loader.h"

#include <X11/Xresource.h>
#include <string.h>
#include <cstring>

#include <sys/utsname.h>
#include <unistd.h>

#if defined(__LP64__)
#define SYSTEM_LIB_PATH "/system/lib64/"
#else
#define SYSTEM_LIB_PATH "/system/lib/"
#endif

#define STR_OR_EMPTY(x) ((x != nullptr) ? x : "")

namespace query {

struct Context {
  int mNumCores;
  utsname mUbuf;
  char mHostName[512];
};

static Context gContext;
static int gContextRefCount = 0;

void destroyContext() {
  if (--gContextRefCount > 0) {
    return;
  }
}

bool createContext(std::string* errorMsg) {
  if (gContextRefCount++ > 0) {
    return true;
  }

  memset(&gContext, 0, sizeof(gContext));

  if (uname(&gContext.mUbuf) != 0) {
    errorMsg->append("uname returned error: " + std::to_string(errno));
    destroyContext();
    return false;
  }

  gContext.mNumCores = sysconf(_SC_NPROCESSORS_CONF);

  if (gethostname(gContext.mHostName, sizeof(gContext.mHostName)) != 0) {
    errorMsg->append("gethostname returned error: " + std::to_string(errno));
    destroyContext();
    return false;
  }

  return true;
}

int numABIs() { return 1; }

void abi(int idx, device::ABI* abi) {
  abi->set_name("x86_64");
  abi->set_os(device::Linux);
  abi->set_architecture(device::X86_64);
  abi->set_allocated_memory_layout(currentMemoryLayout());
}

device::ABI* currentABI() {
  auto out = new device::ABI();
  abi(0, out);
  return out;
}

int cpuNumCores() { return gContext.mNumCores; }

const char* gpuName() { return ""; }

const char* gpuVendor() { return ""; }

const char* instanceName() { return gContext.mHostName; }

const char* hardwareName() { return STR_OR_EMPTY(gContext.mUbuf.machine); }

device::OSKind osKind() { return device::Linux; }

const char* osName() { return STR_OR_EMPTY(gContext.mUbuf.release); }

const char* osBuild() { return STR_OR_EMPTY(gContext.mUbuf.version); }

int osMajor() { return 0; }

int osMinor() { return 0; }

int osPoint() { return 0; }

device::VulkanProfilingLayers* get_vulkan_profiling_layers() {
  auto layers = new device::VulkanProfilingLayers();
  layers->set_cpu_timing(true);
  layers->set_memory_tracker(true);
  return layers;
}

bool hasAtrace() { return false; }

}  // namespace query
