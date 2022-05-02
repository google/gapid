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
#include <fstream>
#include <string>
#include <vector>

#include "base_caller.h"
#include "command_deserializer.h"
#include "command_inline_fixer.h"
#include "decoder.h"
#include "handle_fixer.h"
#include "layer_helper.h"
#include "layerer.h"
#include "minimal_state_tracker.h"
#include "null_caller.h"
#include "transform_base.h"

namespace gapid2 {

class replayer : public command_deserializer {
  using super = command_deserializer;

 public:
  transform_base* call_through;
  handle_fixer* fixer;

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

    VkInstance raw_instance = fixer->VkInstance_map[instance];

    auto original_physical_device_count = *pPhysicalDeviceCount;
    memcpy(pPhysicalDeviceCount, tmp_pPhysicalDeviceCount,
           sizeof(pPhysicalDeviceCount[0]) * 1);  // setting inout properly
    VkResult current_return_ = decoder_->decode<VkResult>();
    // -------- Call ------
    if (!pPhysicalDevices) {
      call_through->vkEnumeratePhysicalDevices(raw_instance, pPhysicalDeviceCount,
                                               pPhysicalDevices);
      return;
    }
    std::vector<VkPhysicalDevice> actual_physical_devices;
    uint32_t actual_physical_device_count = 0;
    call_through->vkEnumeratePhysicalDevices(raw_instance, &actual_physical_device_count,
                                             nullptr);
    VkPhysicalDevice fake_handle =
        reinterpret_cast<VkPhysicalDevice>(static_cast<uintptr_t>(-1));
    actual_physical_devices.resize(actual_physical_device_count);
    pPhysicalDeviceCount[0] = actual_physical_device_count;
    call_through->vkEnumeratePhysicalDevices(raw_instance, pPhysicalDeviceCount,
                                             actual_physical_devices.data());

    std::vector<VkPhysicalDeviceProperties> props;
    // Get the properties for all the CURRENT devices
    for (size_t i = 0; i < pPhysicalDeviceCount[0]; ++i) {
      props.push_back({});
      call_through->vkGetPhysicalDeviceProperties(actual_physical_devices[i],
                                                  &props.back());
    }
    std::vector<size_t> actual_devices(original_physical_device_count);

    const uint64_t data_left = decoder_->data_left();
    if (data_left < sizeof(uint64_t)) {
      return;
    }
    if (data_left - sizeof(uint64_t) < decoder_->decode<uint64_t>()) {
      return;
    }

    for (size_t i = 0; i < original_physical_device_count; ++i) {
      uint32_t device_id = decoder_->decode<uint32_t>();
      uint32_t vendor_id = decoder_->decode<uint32_t>();
      uint32_t driver_version = decoder_->decode<uint32_t>();
      bool pushed = false;
      for (size_t j = 0; j < props.size(); ++j) {
        if (props[j].vendorID == vendor_id && props[j].deviceID == device_id &&
            props[j].driverVersion == driver_version) {
          fixer->VkPhysicalDevice_map[pPhysicalDevices[i]] = actual_physical_devices[j];
          props[j].vendorID = 0xFFFFFFFF;
          props[j].deviceID = 0xFFFFFFFF;
          props[j].driverVersion = 0xFFFFFFFF;
          pushed = true;
          break;
        }
        if (props[j].vendorID == vendor_id && props[j].deviceID == device_id) {
          fixer->VkPhysicalDevice_map[pPhysicalDevices[i]] = actual_physical_devices[j];
          props[j].vendorID = 0xFFFFFFFF;
          props[j].deviceID = 0xFFFFFFFF;
          props[j].driverVersion = 0xFFFFFFFF;
          std::string err =
              "Driver version mismatch, replay may be incorrect for device: ";
          err += props[j].deviceName;
          err += "\n";
          GAPID2_WARNING(err.c_str());
          pushed = true;
          break;
        }
        if (props[j].vendorID == vendor_id) {
          fixer->VkPhysicalDevice_map[pPhysicalDevices[i]] = actual_physical_devices[j];
          props[j].vendorID = 0xFFFFFFFF;
          props[j].deviceID = 0xFFFFFFFF;
          props[j].driverVersion = 0xFFFFFFFF;
          pushed = true;
          std::string err =
              "DeviceID mismatch, trying and hoping for the best with "
              "device: ";
          err += props[j].deviceName;
          err += "\n";
          GAPID2_WARNING(err.c_str());
          pushed = true;
          break;
        }
      }
      if (!pushed) {
        std::string err = "Cannot find device matching deviceID: ";
        err += std::to_string(device_id);
        err += ", and vendorID: ";
        err += std::to_string(vendor_id);
        err += "\n";
        GAPID2_WARNING(err.c_str());
        actual_devices.push_back(0xFFFFFFFF);
        fixer->VkPhysicalDevice_map[pPhysicalDevices[i]] = reinterpret_cast<VkPhysicalDevice>(static_cast<uintptr_t>(0xFFFFFFFF - i));
      }
    }
  }

  void call_vkWaitForFences(decoder* decoder_) override {
    // -------- Args ------
    VkDevice device;
    uint32_t fenceCount;
    VkFence* pFences;  // length fenceCount
    VkBool32 waitAll;
    uint64_t timeout;
    // -------- Serialized Params ------
    device = reinterpret_cast<VkDevice>(
        static_cast<uintptr_t>(decoder_->decode<uint64_t>()));
    fenceCount = decoder_->decode<uint32_t>();
    pFences = decoder_->get_typed_memory<VkFence>(fenceCount);
    for (size_t i_4 = 0; i_4 < fenceCount; ++i_4) {
      pFences[i_4] = reinterpret_cast<VkFence>(decoder_->decode<uint64_t>());
    }
    waitAll = decoder_->decode<uint32_t>();
    timeout = decoder_->decode<uint64_t>();
    // -------- Out Params ------
    // -------- FixUp Params ------
    VkResult current_return_ = decoder_->decode<VkResult>();

    std::vector<VkFence> success_fences;
    success_fences.reserve(fenceCount);
    if (fenceCount == 1 && current_return_ == VK_SUCCESS) {
      success_fences.push_back(pFences[0]);
    } else {
      for (size_t i = 0; i < fenceCount; ++i) {
        char fence = decoder_->decode<char>();
        if (fence) {
          success_fences.push_back(pFences[i]);
        }
      }
    }
    if (success_fences.size() > 0) {
      super::vkWaitForFences(device, success_fences.size(),
                             success_fences.data(), VK_TRUE,
                             ~static_cast<uint64_t>(0));
    }
  }

  void call_vkGetFenceStatus(decoder* decoder_) override {
    // -------- Args ------
    VkDevice device;
    VkFence fence;
    // -------- Serialized Params ------
    device = reinterpret_cast<VkDevice>(
        static_cast<uintptr_t>(decoder_->decode<uint64_t>()));
    fence = reinterpret_cast<VkFence>(decoder_->decode<uint64_t>());
    // -------- Out Params ------
    // -------- FixUp Params ------
    VkResult current_return_ = decoder_->decode<VkResult>();
    // -------- Call ------
    if (current_return_ == VK_SUCCESS) {
      super::vkWaitForFences(device, 1, &fence, VK_TRUE,
                             ~static_cast<uint64_t>(0));
    }
  }

  void* get_memory_write_location(VkDeviceMemory memory,
                                  VkDeviceSize offset,
                                  VkDeviceSize size) override {
    if (dummy_runner) {
      return nullptr;
    }
    if (!memory) {
      return nullptr;
    }
    auto mem = state_block_->get(memory);
    auto retval = mem->_mapped_location;
    GAPID2_ASSERT(retval != nullptr, "Expected memory to be mapped");
    retval += offset;
    GAPID2_ASSERT(offset + size <= mem->_mapped_size,
                  "Writing over the end of mapped memory");
    return static_cast<void*>(retval);
  }

  bool dummy_runner = false;
};
}  // namespace gapid2

int main(int argc, const char** argv) {
  auto begin = std::chrono::high_resolution_clock::now();
  if (argc < 2) {
    GAPID2_ERROR("Expected the file as an argument");
  }
  bool dummy = false;
  for (size_t i = 1; i < argc - 1; ++i) {
    if (!strcmp(argv[i], "--dummy")) {
      dummy = true;
    }
  }

  HANDLE file = CreateFileA(argv[argc - 1], GENERIC_READ, 0, nullptr, OPEN_ALWAYS,
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

  gapid2::transform<gapid2::replayer>
      replayer(nullptr);
  gapid2::transform<gapid2::base_caller> base_caller(dummy ? nullptr : &replayer);
  gapid2::transform<gapid2::null_caller> null_caller(dummy ? &replayer : nullptr);
  gapid2::transform<gapid2::command_inline_fixer> inline_fixer(&replayer);
  gapid2::transform<gapid2::state_block> state_block_(&replayer);
  gapid2::transform<gapid2::minimal_state_tracker> minimal_state_tracker_(&replayer);
  gapid2::transform<gapid2::layerer> layerer_(&replayer);

  layerer_.initializeLayers(gapid2::get_layers());

  if (!dummy) {
    auto vk = LoadLibraryA("vulkan-1.dll");
    auto gipa = reinterpret_cast<PFN_vkGetInstanceProcAddr>(
        GetProcAddress(vk, "vkGetInstanceProcAddr"));
    auto vkci = reinterpret_cast<PFN_vkCreateInstance>(
        gipa(nullptr, "vkCreateInstance"));
    base_caller.vkCreateInstance_ = vkci;
    base_caller.vkGetInstanceProcAddr_ = gipa;
  }
  layerer_.fixer = &inline_fixer.fix_;
  replayer.fixer = &inline_fixer.fix_;
  replayer.call_through = dummy ? static_cast<gapid2::transform_base*>(&null_caller) : static_cast<gapid2::transform_base*>(&base_caller);
  replayer.dummy_runner = dummy;

  std::vector<block>
      b({block{static_cast<uint64_t>(fileSize.QuadPart),
               reinterpret_cast<char*>(loc), 0}});
  gapid2::decoder dec(std::move(b));
  auto res = std::chrono::high_resolution_clock::now();
  replayer.DeserializeStream(&dec);
  auto end = std::chrono::high_resolution_clock::now();
  auto elapsed = end - res;
  OutputDebugString(("Initializing time:: " +
                     std::to_string(std::chrono::duration<float>(res - begin).count()) +
                     "\n")
                        .c_str());
  auto str = "Run time:: " +
             std::to_string(std::chrono::duration<float>(elapsed).count()) +
             "\n";
  OutputDebugString(str.c_str());
  return 0;
}
