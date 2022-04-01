/*
 * Copyright (C) 2022 Google Inc.
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

#include "core/os/device/deviceinfo/cc/query.h"

#include <sys/utsname.h>
#include <unistd.h>

#include "core/cc/dl_loader.h"

#define STR_OR_EMPTY(x) ((x != nullptr) ? x : "")

namespace {

device::ABI* abi(device::ABI* abi) {
  // TODO: need aarch64 abi differentiation ?
  abi->set_name("ARMv8a");
  abi->set_os(device::Fuchsia);
  abi->set_architecture(device::ARMv8a);
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

  info->osKind = device::Fuchsia;
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
