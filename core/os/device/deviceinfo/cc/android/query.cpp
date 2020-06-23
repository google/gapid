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

#include "core/cc/assert.h"
#include "core/cc/log.h"

#include <cstring>
#include <sstream>

#include <android/log.h>
#include <sys/system_properties.h>

#define LOG_ERR(...) __android_log_print(ANDROID_LOG_ERROR, "AGI", __VA_ARGS__);

#define LOG_WARN(...) __android_log_print(ANDROID_LOG_WARN, "AGI", __VA_ARGS__);

namespace {

device::DataTypeLayout* new_dt_layout(int size, int alignment) {
  auto out = new device::DataTypeLayout();
  out->set_size(size);
  out->set_alignment(alignment);
  return out;
}

void abiByName(const std::string name, device::ABI* abi) {
  abi->set_name(name);
  abi->set_os(device::Android);

  if (name == "armeabi-v7a") {
    // http://infocenter.arm.com/help/topic/com.arm.doc.ihi0042f/IHI0042F_aapcs.pdf
    // 4 DATA TYPES AND ALIGNMENT
    auto memory_layout = new device::MemoryLayout();
    memory_layout->set_allocated_pointer(new_dt_layout(4, 4));
    memory_layout->set_allocated_integer(new_dt_layout(4, 4));
    memory_layout->set_allocated_size(new_dt_layout(4, 4));
    memory_layout->set_allocated_char_(new_dt_layout(1, 1));
    memory_layout->set_allocated_i64(new_dt_layout(8, 8));
    memory_layout->set_allocated_i32(new_dt_layout(4, 4));
    memory_layout->set_allocated_i16(new_dt_layout(2, 2));
    memory_layout->set_allocated_i8(new_dt_layout(1, 1));
    memory_layout->set_allocated_f64(new_dt_layout(8, 8));
    memory_layout->set_allocated_f32(new_dt_layout(4, 4));
    memory_layout->set_allocated_f16(new_dt_layout(2, 2));
    memory_layout->set_endian(device::LittleEndian);
    abi->set_allocated_memory_layout(memory_layout);
    abi->set_architecture(device::ARMv7a);
  } else if (name == "arm64-v8a") {
    // http://infocenter.arm.com/help/topic/com.arm.doc.ihi0055b/IHI0055B_aapcs64.pdf
    // 4 DATA TYPES AND ALIGNMENT
    auto memory_layout = new device::MemoryLayout();
    memory_layout->set_allocated_pointer(new_dt_layout(8, 8));
    memory_layout->set_allocated_integer(new_dt_layout(8, 8));
    memory_layout->set_allocated_size(new_dt_layout(8, 8));
    memory_layout->set_allocated_char_(new_dt_layout(1, 1));
    memory_layout->set_allocated_i64(new_dt_layout(8, 8));
    memory_layout->set_allocated_i32(new_dt_layout(4, 4));
    memory_layout->set_allocated_i16(new_dt_layout(2, 2));
    memory_layout->set_allocated_i8(new_dt_layout(1, 1));
    memory_layout->set_allocated_f64(new_dt_layout(8, 8));
    memory_layout->set_allocated_f32(new_dt_layout(4, 4));
    memory_layout->set_allocated_f16(new_dt_layout(2, 2));
    memory_layout->set_endian(device::LittleEndian);
    abi->set_allocated_memory_layout(memory_layout);
    abi->set_architecture(device::ARMv8a);
  } else if (name == "x86") {
    // https://en.wikipedia.org/wiki/Data_structure_alignment#Typical_alignment_of_C_structs_on_x86
    auto memory_layout = new device::MemoryLayout();
    memory_layout->set_allocated_pointer(new_dt_layout(4, 4));
    memory_layout->set_allocated_integer(new_dt_layout(4, 4));
    memory_layout->set_allocated_size(new_dt_layout(4, 4));
    memory_layout->set_allocated_char_(new_dt_layout(1, 1));
    memory_layout->set_allocated_i64(new_dt_layout(8, 4));
    memory_layout->set_allocated_i32(new_dt_layout(4, 4));
    memory_layout->set_allocated_i16(new_dt_layout(2, 2));
    memory_layout->set_allocated_i8(new_dt_layout(1, 1));
    memory_layout->set_allocated_f64(new_dt_layout(8, 4));
    memory_layout->set_allocated_f32(new_dt_layout(4, 4));
    memory_layout->set_allocated_f16(new_dt_layout(2, 2));
    memory_layout->set_endian(device::LittleEndian);
    abi->set_allocated_memory_layout(memory_layout);
    abi->set_architecture(device::X86);
  } else if (name == "x86_64") {
    auto memory_layout = new device::MemoryLayout();
    memory_layout->set_allocated_pointer(new_dt_layout(8, 8));
    memory_layout->set_allocated_integer(new_dt_layout(4, 4));
    memory_layout->set_allocated_size(new_dt_layout(8, 8));
    memory_layout->set_allocated_char_(new_dt_layout(1, 1));
    memory_layout->set_allocated_i64(new_dt_layout(8, 4));
    memory_layout->set_allocated_i32(new_dt_layout(4, 4));
    memory_layout->set_allocated_i16(new_dt_layout(2, 2));
    memory_layout->set_allocated_i8(new_dt_layout(1, 1));
    memory_layout->set_allocated_f64(new_dt_layout(8, 4));
    memory_layout->set_allocated_f32(new_dt_layout(4, 4));
    memory_layout->set_allocated_f16(new_dt_layout(2, 2));
    memory_layout->set_endian(device::LittleEndian);
    abi->set_allocated_memory_layout(memory_layout);
    abi->set_architecture(device::X86_64);
  } else {
    LOG_WARN("Unrecognised ABI: %s", name.c_str());
  }
}

typedef struct {
  int major, minor;
} AndroidVersion;

const AndroidVersion kVersionBySdk[] = {
    {0, 0}, {1, 0}, {1, 1}, {1, 5}, {1, 6},  /*  0- 4 */
    {2, 0}, {2, 0}, {2, 1}, {2, 2}, {2, 3},  /*  5- 9 */
    {2, 3}, {3, 0}, {3, 1}, {3, 2}, {4, 0},  /* 10-14 */
    {4, 0}, {4, 1}, {4, 2}, {4, 3}, {4, 4},  /* 15-19 */
    {4, 4}, {5, 0}, {5, 1}, {6, 0}, {7, 0},  /* 20-24 */
    {7, 1}, {8, 0}, {8, 1}, {9, 0}, {10, 0}, /* 25-29*/
    {11, 0}};
constexpr int kMaxKnownSdk = (sizeof(kVersionBySdk) / sizeof(AndroidVersion));

}  // anonymous namespace

namespace query {

bool queryPlatform(PlatformInfo* info, std::string* errorMsg) {
  info->numCpuCores = 0;

#define GET_PROP(name, trans)                       \
  do {                                              \
    char _v[PROP_VALUE_MAX] = {0};                  \
    if (__system_property_get(name, _v) == 0) {     \
      errorMsg->append("Failed reading property "); \
      errorMsg->append(name);                       \
      return false;                                 \
    }                                               \
    trans;                                          \
  } while (0)

#define GET_STRING_PROP(n, t) GET_PROP(n, t = _v)
#define GET_INT_PROP(n, t) GET_PROP(n, t = atoi(_v))
#define GET_STRING_LIST_PROP(n, t)      \
  do {                                  \
    std::string _l, _t;                 \
    GET_STRING_PROP(n, _l);             \
    std::istringstream _s(_l);          \
    while (std::getline(_s, _t, ',')) { \
      t.push_back(_t);                  \
    }                                   \
  } while (0)

  std::string manufacturer;
  std::string model;
  GET_STRING_PROP("ro.product.manufacturer", manufacturer);
  GET_STRING_PROP("ro.product.model", model);
  GET_STRING_PROP("ro.hardware", info->hardwareName);
  if (model != "") {
    if (manufacturer != "") {
      info->name = manufacturer + " " + model;
    } else {
      info->name = model;
    }
  } else {
    info->name = info->hardwareName;
  }

  std::vector<std::string> supportedABIs;
  GET_STRING_LIST_PROP("ro.product.cpu.abilist", supportedABIs);
  info->abis.resize(supportedABIs.size());
  for (size_t i = 0; i < supportedABIs.size(); i++) {
    abiByName(supportedABIs[i], &info->abis[i]);
  }

  info->numCpuCores = 0;

  info->osKind = device::Android;
  GET_STRING_PROP("ro.build.version.release", info->osName);
  GET_STRING_PROP("ro.build.display.id", info->osBuild);
  int sdkVersion = 0;
  GET_INT_PROP("ro.build.version.sdk", sdkVersion);
  // preview_sdk is used to determine the version for the next OS release
  // Until the official release, the new OS releases will use the same sdk
  // version as the previous OS while setting the preview_sdk
  int previewSdk = 0;
  GET_INT_PROP("ro.build.version.preview_sdk", previewSdk);
  sdkVersion += previewSdk;
  if (sdkVersion >= 0 && sdkVersion < kMaxKnownSdk) {
    info->osMajor = kVersionBySdk[sdkVersion].major;
    info->osMinor = kVersionBySdk[sdkVersion].minor;
  }

  return true;
}

device::ABI* currentABI() {
  device::ABI* out = new device::ABI();
#if defined(__arm__)
  abiByName("armeabi-v7a", out);
#elif defined(__aarch64__)
  abiByName("arm64-v8a", out);
#elif defined(__i686__)
  abiByName("x86", out);
#elif defined(__x86_64__)
  abiByName("x86_64", out);
#else
#error "Unknown ABI"
#endif
  return out;
}

device::VulkanProfilingLayers* get_vulkan_profiling_layers() {
  auto layers = new device::VulkanProfilingLayers();
  layers->set_cpu_timing(true);
  layers->set_memory_tracker(false);
  return layers;
}

bool hasAtrace() { return true; }

}  // namespace query
