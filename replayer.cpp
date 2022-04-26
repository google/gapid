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

#include "command_caller.h"
#include "command_deserializer.h"
#include "decoder.h"
#include "handle_runner.h"
#include "layer_helper.h"
#include "layerer.h"
#include "minimal_state_tracker.h"
#include "physical_device.h"

namespace gapid2 {
VkResult VKAPI_PTR layer_vkSetInstanceLoaderData(VkInstance instance,
                                                 void* object) {
  return VK_SUCCESS;
}
VkResult VKAPI_PTR layer_vkSetDeviceLoaderData(VkDevice device, void* object) {
  return VK_SUCCESS;
}

class Replayer : public gapid2::CommandDeserializer<gapid2::Layerer<
                     gapid2::MinimalStateTracker<
                         gapid2::CommandCaller<gapid2::HandleRunner<true>>>,
                     gapid2::HandleRunner<true>>> {
  using super = gapid2::CommandDeserializer<
      gapid2::Layerer<gapid2::MinimalStateTracker<
                          gapid2::CommandCaller<gapid2::HandleRunner<true>>>,
                      gapid2::HandleRunner<true>>>;
  using caller = gapid2::CommandCaller<gapid2::HandleRunner<true>>;

 public:
  Replayer() {}
  void vkCreateInstance(decoder* decoder_) override {
    super::vkCreateInstance(decoder_);
    for (auto& el : updater_.VkInstances_out_) {
      if (!el.second->_functions) {
        el.second->set_instance_data(gipa, &layer_vkSetInstanceLoaderData);
      }
    }
  }
  void vkCreateDevice(decoder* decoder_) override {
    super::vkCreateDevice(decoder_);
    for (auto& el : updater_.VkDevices_out_) {
      if (!el.second->_functions) {
        el.second->set_device_loader_data(&layer_vkSetDeviceLoaderData);
        el.second->_functions =
            std::make_unique<gapid2::DeviceFunctions>(el.second->_handle, gdpa);
      }
    }
  }

  // Custom vkEnumeratePhysicalDevices to handle the case where a vendor
  // or system may re-order physical devices based on certain
  // paramters of the application.
  // We have stored the VendorID/DeviceID in the trace just after the
  // call so look there.
  virtual void vkEnumeratePhysicalDevices(decoder* decoder_) override {
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
    // -------- FixUp Params ------
    updater_.register_handle(pPhysicalDevices, pPhysicalDeviceCount);

    auto original_physical_device_count = *pPhysicalDeviceCount;
    memcpy(pPhysicalDeviceCount, tmp_pPhysicalDeviceCount,
           sizeof(pPhysicalDeviceCount[0]) * 1);  // setting inout properly
    VkResult current_return_ = decoder_->decode<VkResult>();
    // -------- Call ------
    if (!pPhysicalDevices) {
      caller::vkEnumeratePhysicalDevices(instance, pPhysicalDeviceCount,
                                         pPhysicalDevices);
      return;
    }
    std::vector<VkPhysicalDevice> actual_physical_devices;
    uint32_t actual_physical_device_count;
    caller::vkEnumeratePhysicalDevices(instance, &actual_physical_device_count,
                                       nullptr);
    VkPhysicalDevice fake_handle =
        reinterpret_cast<VkPhysicalDevice>(static_cast<uintptr_t>(-1));
    for (uint32_t i = original_physical_device_count;
         i < actual_physical_device_count; ++i) {
      updater_.register_handle(&fake_handle, 1);
    }
    actual_physical_devices.resize(actual_physical_device_count);
    pPhysicalDeviceCount[0] = actual_physical_device_count;
    caller::vkEnumeratePhysicalDevices(instance, pPhysicalDeviceCount,
                                       actual_physical_devices.data());

    std::vector<gapid2::VkPhysicalDeviceWrapper<gapid2::HandleRunner<true>>*>
        physical_devices(pPhysicalDeviceCount[0]);
    for (size_t i = 0; i < pPhysicalDeviceCount[0]; ++i) {
      physical_devices[i] = updater_.cast_from_vk(actual_physical_devices[i]);
    }

    std::vector<VkPhysicalDeviceProperties> props;
    // Get the properties for all the CURRENT devices
    for (size_t i = 0; i < pPhysicalDeviceCount[0]; ++i) {
      props.push_back({});
      caller::vkGetPhysicalDeviceProperties(actual_physical_devices[i],
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
          updater_.VkPhysicalDevices_out_[pPhysicalDevices[i]] =
              physical_devices[j];
          props[j].vendorID = 0xFFFFFFFF;
          props[j].deviceID = 0xFFFFFFFF;
          props[j].driverVersion = 0xFFFFFFFF;
          pushed = true;
          break;
        }
        if (props[j].vendorID == vendor_id && props[j].deviceID == device_id) {
          updater_.VkPhysicalDevices_out_[pPhysicalDevices[i]] =
              physical_devices[j];
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
          updater_.VkPhysicalDevices_out_[pPhysicalDevices[i]] =
              physical_devices[j];
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
        updater_.tbd_handles.pop_front();
      }
    }

    GAPID2_ASSERT(updater_.tbd_handles.empty(), "Unprocessed handles");
  }

  void vkWaitForFences(decoder* decoder_) override {
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
      caller::vkWaitForFences(device, success_fences.size(),
                              success_fences.data(), VK_TRUE,
                              ~static_cast<uint64_t>(0));
    }
    GAPID2_ASSERT(updater_.tbd_handles.empty(), "Unprocessed handles");
  }

  void vkGetFenceStatus(decoder* decoder_) override {
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
      caller::vkWaitForFences(device, 1, &fence, VK_TRUE,
                              ~static_cast<uint64_t>(0));
    }
    GAPID2_ASSERT(updater_.tbd_handles.empty(), "Unprocessed handles");
  }

  void* get_memory_write_location(VkDeviceMemory memory,
                                  VkDeviceSize offset,
                                  VkDeviceSize size) override {
    auto mem = updater_.cast_from_vk(memory);
    auto retval = mem->_mapped_location;
    GAPID2_ASSERT(retval != nullptr, "Expected memory to be mapped");
    retval += offset;
    GAPID2_ASSERT(offset + size <= mem->_mapped_size,
                  "Writing over the end of mapped memory");
    return static_cast<void*>(retval);
  }

  PFN_vkGetInstanceProcAddr gipa;
  PFN_vkGetDeviceProcAddr gdpa;
  temporary_allocator allocator;
};
}  // namespace gapid2

int main(int argc, const char** argv) {
  if (argc < 2) {
    GAPID2_ERROR("Expected the file as an argument");
  }

  auto vk = LoadLibraryA("vulkan-1.dll");
  auto ci = GetProcAddress(vk, "vkCreateInstance");

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

  gapid2::Replayer replayer;
  replayer._vkCreateInstance = reinterpret_cast<PFN_vkCreateInstance>(ci);
  replayer.gipa = reinterpret_cast<PFN_vkGetInstanceProcAddr>(
      GetProcAddress(vk, "vkGetInstanceProcAddr"));
  replayer.gdpa = reinterpret_cast<PFN_vkGetDeviceProcAddr>(
      GetProcAddress(vk, "vkGetDeviceProcAddr"));
  replayer.initializeLayers(gapid2::get_layers());
  std::vector<block> b({block{static_cast<uint64_t>(fileSize.QuadPart),
                              reinterpret_cast<char*>(loc), 0}});
  gapid2::decoder dec(std::move(b));
  auto res = std::chrono::high_resolution_clock::now();
  replayer.DeserializeStream(&dec);
  auto end = std::chrono::high_resolution_clock::now();
  auto elapsed = end - res;
  auto str = "Elapsed time:: " +
             std::to_string(std::chrono::duration<float>(elapsed).count()) +
             "\\n";
  OutputDebugString(str.c_str());
  return 0;
}
