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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <chrono>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

#include "command_deserializer.h"
#include "command_printer.h"
#include "decoder.h"
#include "json_printer.h"
#include "minimal_state_tracker.h"
#include "null_caller.h"
#include "transform.h"

namespace gapid2 {
class trace_printer : public command_deserializer {
  using super = command_deserializer;

 public:
  // Custom vkEnumeratePhysicalDevices to handle the case where a vendor
  // or system may re-order physical devices based on certain
  // paramters of the application.
  // We have stored the VendorID/DeviceID in the trace just after the
  // call so look there.
  virtual void call_vkEnumeratePhysicalDevices(decoder* decoder_) override {
    // -------- Args ------
    VkInstance instance;
    uint32_t tmp_pPhysicalDeviceCount[1];
    VkPhysicalDevice* pPhysicalDevices;  // optional
    // -------- Serialized Params ------
    instance = reinterpret_cast<VkInstance>(
        static_cast<uintptr_t>(decoder_->decode<uint64_t>()));
    tmp_pPhysicalDeviceCount[0] = decoder_->decode<uint32_t>();
    // -------- Out Params ------
    uint32_t pPhysicalDeviceCount[1];  // inout
    pPhysicalDeviceCount[0] = decoder_->decode<uint32_t>();
    if (decoder_->decode<char>()) {
      pPhysicalDevices =
          decoder_->get_typed_memory<VkPhysicalDevice>(*pPhysicalDeviceCount);
      for (size_t i_6 = 0; i_6 < *pPhysicalDeviceCount; ++i_6) {
        pPhysicalDevices[i_6] = reinterpret_cast<VkPhysicalDevice>(
            static_cast<uintptr_t>(decoder_->decode<uint64_t>()));
      }
    } else {
      pPhysicalDevices = nullptr;
    }

    VkResult current_return_ = decoder_->decode<VkResult>();

    call_through->vkEnumeratePhysicalDevices(instance, pPhysicalDeviceCount,
                                             pPhysicalDevices);
    if (!pPhysicalDevices) {
      return;
    }

    const uint64_t data_left = decoder_->data_left();
    if (data_left < sizeof(uint64_t)) {
      return;
    }
    if (data_left - sizeof(uint64_t) < decoder_->decode<uint64_t>()) {
      return;
    }
    for (size_t i = 0; i < pPhysicalDeviceCount[0]; ++i) {
      uint32_t device_id = decoder_->decode<uint32_t>();
      uint32_t vendor_id = decoder_->decode<uint32_t>();
      uint32_t driver_version = decoder_->decode<uint32_t>();
      (void)device_id;
      (void)vendor_id;
      (void)driver_version;
    }
  }

 public:
  transform_base* call_through;
  trace_printer() {}
  temporary_allocator allocator;
};
}  // namespace gapid2

const std::string version_string = "1";

int main(int argc, const char** argv) {
  HANDLE file = CreateFileA(argv[1], GENERIC_READ, 0, nullptr, OPEN_ALWAYS,
                            FILE_ATTRIBUTE_NORMAL, NULL);

  if (!file) {
    OutputDebugStringA("Error could not open file");
    return -1;
  }

  LARGE_INTEGER fileSize;
  if (!GetFileSizeEx(file, &fileSize)) {
    OutputDebugStringA("Error could not determine file size");
    return -1;
  }

  HANDLE mapping =
      CreateFileMappingA(file, nullptr, PAGE_READONLY, 0, 0, nullptr);
  if (!mapping) {
    OutputDebugStringA("Error could not create file mapping");
    return -1;
  }

  PVOID loc = MapViewOfFile(mapping, FILE_MAP_READ, 0, 0, 0);
  if (!loc) {
    OutputDebugStringA("Could not map view of file");
    return -1;
  }

  gapid2::transform<gapid2::trace_printer> printer(nullptr);
  gapid2::transform<gapid2::null_caller> null_caller(&printer);
  gapid2::transform<gapid2::state_block> state_block_(&printer);
  gapid2::transform<gapid2::minimal_state_tracker> minimal_state_tracker_(&printer);
  gapid2::transform<gapid2::command_printer> command_printer(&printer);
  printer.call_through = &minimal_state_tracker_;
  gapid2::json_printer jp;
  if (argc > 2) {
    jp.set_file(argv[2]);
  }

  command_printer.printer_ = &jp;

  std::vector<block> b({block{static_cast<uint64_t>(fileSize.QuadPart),
                              reinterpret_cast<char*>(loc), 0}});
  gapid2::decoder dec(std::move(b));
  auto res = std::chrono::high_resolution_clock::now();

  jp.begin_array("");
  printer.DeserializeStream(&dec);
  jp.end_array();

  auto end = std::chrono::high_resolution_clock::now();
  auto elapsed = end - res;
  auto str = "Elapsed time:: " +
             std::to_string(std::chrono::duration<float>(elapsed).count()) +
             "\\n";
  OutputDebugString(str.c_str());
  return 0;
}