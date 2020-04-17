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

#include "query.h"

#include "core/cc/target.h"

#if (defined(__x86_64) || defined(__i386))
#include <cpuid.h>

bool query::queryCpu(CpuInfo* info, std::string* error) {
  static union {
    uint32_t reg[12];
    char str[49];
  };
  if (__get_cpuid(0x80000002, &reg[0], &reg[1], &reg[2], &reg[3]) &&
      __get_cpuid(0x80000003, &reg[4], &reg[5], &reg[6], &reg[7]) &&
      __get_cpuid(0x80000004, &reg[8], &reg[9], &reg[10], &reg[11])) {
    info->name = str;
  } else {
    error->append("Failed to query CPUID");
    return false;
  }

  str[12] = 0;  // In case the below 12 byte vendor name uses exactly 12 chars.
  uint32_t eax = 0;
  if (__get_cpuid(0, &eax, &reg[0], &reg[2], &reg[1])) {
    info->vendor = str;
  } else {
    error->append("Failed to query CPUID");
    return false;
  }

  info->architecture = device::X86_64;
  return true;
}

#elif ((defined(__arm__) || defined(__aarch64__)) && \
       TARGET_OS == GAPID_OS_ANDROID)
#include <sys/system_properties.h>
#include <fstream>

bool query::queryCpu(CpuInfo* info, std::string* error) {
  std::fstream proc("/proc/cpuinfo", std::ios_base::in);
  if (proc.is_open()) {
    std::string line, processor, hardware;
    while (std::getline(proc, line)) {
      size_t colon = line.rfind(": ");
      if (colon == std::string::npos) {
      } else if (line.rfind("Hardware") == 0) {
        hardware = line.substr(colon + 2);
      } else if (line.rfind("Processor") == 0) {
        processor = line.substr(colon + 2);
      }
    }
    proc.close();

    if (hardware != "") {
      info->name = hardware;
    } else if (processor != "") {
      info->name = processor;
    }
  }

  if (info->name == "") {
    static const char* cpuProps[] = {
        "ro.boot.hardware.platform",
        "ro.hardware.chipname",
        "ro.boot.hardware",
        "ro.hardware",
        "ro.arch",
    };
    char str[PROP_VALUE_MAX];
    for (const char* prop : cpuProps) {
      if (__system_property_get(prop, str) != 0) {
        info->name = str;
        break;
      }
    }
  }

  info->vendor = "ARM";  // TODO: get the implementer?
#ifdef __arm__
  info->architecture = device::ARMv7a;
#else
  info->architecture = device::ARMv8a;
#endif
  return true;
}

#else
#error Unsupported target architecture.
#endif
