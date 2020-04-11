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

#include <windows.h>

namespace {

std::string getOsName(const OSVERSIONINFOEX& version) {
  bool isNTWorkstation = version.wProductType == VER_NT_WORKSTATION;
  int major = version.dwMajorVersion;
  int minor = version.dwMinorVersion;
  if (major == 10 && isNTWorkstation) {
    return "Windows 10";
  } else if (major == 10 && !isNTWorkstation) {
    return "Windows Server 2016 Technical Preview";
  } else if (major == 6 && minor == 3 && isNTWorkstation) {
    return "Windows 8.1";
  } else if (major == 6 && minor == 3 && !isNTWorkstation) {
    return "Windows Server 2012 R2";
  } else if (major == 6 && minor == 2 && isNTWorkstation) {
    return "Windows 8";
  } else if (major == 6 && minor == 2 && !isNTWorkstation) {
    return "Windows Server 2012";
  } else if (major == 6 && minor == 1 && isNTWorkstation) {
    return "Windows 7";
  } else if (major == 6 && minor == 1 && !isNTWorkstation) {
    return "Windows Server 2008 R2";
  } else if (major == 6 && minor == 0 && isNTWorkstation) {
    return "Windows Vista";
  } else if (major == 6 && minor == 0 && !isNTWorkstation) {
    return "Windows Server 2008";
  } else if (major == 5 && minor == 1) {
    return "Windows XP";
  } else if (major == 5 && minor == 0) {
    return "Windows 2000";
  } else {
    return "";
  }
}

device::ABI* abi(device::ABI* abi) {
  abi->set_name("x86_64");
  abi->set_os(device::Windows);
  abi->set_architecture(device::X86_64);
  abi->set_allocated_memory_layout(query::currentMemoryLayout());
  return abi;
}

}  // namespace

namespace query {

bool queryPlatform(PlatformInfo* info, std::string* errorMsg) {
  DWORD size = MAX_COMPUTERNAME_LENGTH + 1;
  WCHAR host_wide[MAX_COMPUTERNAME_LENGTH + 1];
  if (!GetComputerNameW(host_wide, &size)) {
    errorMsg->append("Couldn't get host name: " +
                     std::to_string(GetLastError()));
    return false;
  }
  char hostName[MAX_COMPUTERNAME_LENGTH * 4 + 1];  // Stored as UTF-8
  WideCharToMultiByte(CP_UTF8,                     // CodePage
                      0,                           // dwFlags
                      host_wide,                   // lpWideCharStr
                      -1,                          // cchWideChar
                      hostName,                    // lpMultiByteStr
                      sizeof(hostName),            // cbMultiByte
                      nullptr,                     // lpDefaultChar
                      nullptr                      // lpUsedDefaultChar
  );

  info->name = hostName;
  info->abis.resize(1);
  abi(&info->abis[0]);

  SYSTEM_INFO sysInfo;
  GetSystemInfo(&sysInfo);
  info->numCpuCores = sysInfo.dwNumberOfProcessors;

  info->osKind = device::Windows;
  OSVERSIONINFOEX osVersion;
  osVersion.dwOSVersionInfoSize = sizeof(osVersion);
  GetVersionEx((OSVERSIONINFO*)(&osVersion));
  info->osName = getOsName(osVersion);
  info->osMajor = osVersion.dwMajorVersion;
  info->osMinor = osVersion.dwMinorVersion;
  info->osPoint = osVersion.dwBuildNumber;

  return true;
}

device::ABI* currentABI() { return abi(new device::ABI()); }

device::VulkanProfilingLayers* get_vulkan_profiling_layers() { return nullptr; }

bool hasAtrace() { return false; }

}  // namespace query
