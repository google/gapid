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

namespace {

device::ABI* abi(device::ABI* abi) {
  abi->set_name("x86_64");
  abi->set_os(device::Linux);
  abi->set_architecture(device::X86_64);
  abi->set_allocated_memory_layout(query::currentMemoryLayout());
  return abi;
}

}  // namespace

namespace query {

bool queryPlatform(PlatformInfo* info, std::string* errorMsg) {
  char hostname[HOST_NAME_MAX + 1];
  if (gethostname(hostname, sizeof(hostname)) != 0) {
    errorMsg->append("gethostname returned error: " + std::to_string(errno));
    return false;
  }
  info->name = hostname;
  info->abis.resize(1);
  abi(&info->abis[0]);

  utsname ubuf;
  if (uname(&ubuf) != 0) {
    errorMsg->append("uname returned error: " + std::to_string(errno));
    return false;
  }
  info->hardwareName = STR_OR_EMPTY(ubuf.machine);

  info->numCpuCores = sysconf(_SC_NPROCESSORS_CONF);

  info->osKind = device::Linux;
  info->osName = STR_OR_EMPTY(ubuf.release);
  info->osBuild = STR_OR_EMPTY(ubuf.version);

  return true;
}

device::ABI* currentABI() { return abi(new device::ABI()); }

device::VulkanProfilingLayers* get_vulkan_profiling_layers() {
  auto layers = new device::VulkanProfilingLayers();
  layers->set_cpu_timing(true);
  layers->set_memory_tracker(true);
  return layers;
}

bool hasAtrace() { return false; }

}  // namespace query
