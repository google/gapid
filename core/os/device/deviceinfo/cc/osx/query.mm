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

#import <AppKit/AppKit.h>

#include <sys/sysctl.h>
#include <sys/types.h>
#include <unistd.h>
#include <cstring>

#define STR_OR_EMPTY(x) ((x != nullptr) ? x : "")

namespace {

device::ABI* abi(device::ABI* abi) {
  abi->set_name("x86_64");
  abi->set_os(device::OSX);
  abi->set_architecture(device::X86_64);
  abi->set_allocated_memory_layout(query::currentMemoryLayout());
  return abi;
}

}  // namespace

namespace query {

bool queryPlatform(PlatformInfo* info, std::string* errorMsg) {
  char hostname[256];
  if (gethostname(hostname, sizeof(hostname)) != 0) {
    errorMsg->append("gethostname returned error: " + std::to_string(errno));
    return false;
  }
  info->name = hostname;
  info->abis.resize(1);
  abi(&info->abis[0]);

  size_t len = 0;
  int mib[2] = {CTL_HW, HW_MODEL};
  if (sysctl(mib, 2, nullptr, &len, nullptr, 0) != 0) {
    errorMsg->append("sysctl {CTL_HW, HW_MODEL} returned error: " + std::to_string(errno));
    return false;
  }
  char* hwModel = new char[len];
  if (sysctl(mib, 2, hwModel, &len, nullptr, 0) != 0) {
    errorMsg->append("sysctl {CTL_HW, HW_MODEL} returned error: " + std::to_string(errno));
    delete[] hwModel;
    return false;
  }
  info->hardwareName = hwModel;
  delete[] hwModel;

  len = sizeof(info->numCpuCores);
  if (sysctlbyname("hw.logicalcpu_max", &info->numCpuCores, &len, nullptr, 0) != 0) {
    errorMsg->append("sysctlbyname 'hw.logicalcpu_max' returned error: " + std::to_string(errno));
    return false;
  }

  NSOperatingSystemVersion version = [[NSProcessInfo processInfo] operatingSystemVersion];
  info->osKind = device::OSX;
  info->osName = "OSX";
  info->osMajor = version.majorVersion;
  info->osMinor = version.minorVersion;
  info->osPoint = version.patchVersion;

  return true;
}

device::ABI* currentABI() { return abi(new device::ABI()); }

device::VulkanProfilingLayers* get_vulkan_profiling_layers() { return nullptr; }

bool hasAtrace() { return false; }

}  // namespace query
