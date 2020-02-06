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

#ifndef DEVICEINFO_QUERY_H
#define DEVICEINFO_QUERY_H

#include <functional>

#include "core/os/device/device.pb.h"

namespace query {

// VulkanOption specifies whether Vulkan layers/extentions info or Vulkan
// physical devices info should be queried with query::getDeviceInstance().
// By default, layers/extensions info and physical devices info will not be
// queried.
class VulkanOption {
 public:
  // Default VulkanOption specifies NOT to query layers/extensions and physical
  // devices info.
  VulkanOption()
      : query_layers_and_extensions_(false), query_physical_devices_(false) {}
  // set_query_layers_and_extensions sets the flag to indicate whether layers
  // and extensions info should be queried.
  VulkanOption& set_query_layers_and_extensions(bool f) {
    this->query_layers_and_extensions_ = f;
    return *this;
  }
  // set_query_physical_devices sets the flag to indicate whether physical
  // devices info should be queried.
  VulkanOption& set_query_physical_devices(bool f) {
    this->query_physical_devices_ = f;
    return *this;
  }
  inline bool query_layers_and_extensions() const {
    return query_layers_and_extensions_;
  }
  inline bool query_physical_devices() const { return query_physical_devices_; }

 private:
  // Flags to indicate whether some info should be queried.
  bool query_layers_and_extensions_;
  bool query_physical_devices_;
};

// Option specifies how some optional device info to be queried with
// query::getDeviceInstance().
struct Option {
  VulkanOption vulkan;
};

// getDeviceInstance returns the device::Instance proto message for the
// current device. It must be freed with delete.
device::Instance* getDeviceInstance(const Option& opt);

// updateVulkanPhysicalDevices modifies the given device::Instance by adding
// device::VulkanPhysicalDevice to the device::Instance. If a
// vkGetInstanceProcAddress function is given, that function will be used to
// resolve Vulkan calls, otherwise, all the Vulkan calls will be resolved from
// Vulkan loader. If the given VkInstance handle is 0, a new VkInstance handle
// will be created with the VkCreateInstance resolved from either the given
// callback or the Vulkan loader.  The modified device::Instance will have its
// ID re-hashed with the new content. Returns true if Vulkan physical device
// info is fetched successfully and device::Instance updated, otherwise returns
// false and keeps the device::Instance unchanged. Caution: When called with
// GraphicsSpy layer loaded i.e. during tracing, the function pointer to a layer
// under GraphicsSpy must be passed in. Resolving the Vulkan function addresses
// from loader will cause a infinite calling stack and may deadlock in the
// loader.
bool updateVulkanDriver(
    device::Instance* inst, size_t vk_inst_handle = 0,
    std::function<void*(size_t, const char*)> get_inst_proc_addr = nullptr);

// vkLayersAndExtensions populates the layers and extension fields in the given
// device::VulkanDriver. Returns true if succeeded. If a
// VkGetInstanceProcAddress callback is given, that function will be used to
// resolve Vulkan layer/extension enumeration calls, otherwise Vulkan loader
// will be used for the resolvation.
bool vkLayersAndExtensions(
    device::VulkanDriver*,
    std::function<void*(size_t, const char*)> get_inst_proc_addr = nullptr);

// vkPhysicalDevices populates the VulkanPhysicalDevices fields in the
// given device::VulkanDriver. Returns true if succeeded. If a
// vkGetInstanceProcAddress function is given, that function will be used to
// resolve Vulkan API calls, otherwise Vulkan loader will be used.
bool vkPhysicalDevices(
    device::VulkanDriver*, size_t vk_inst = 0,
    std::function<void*(size_t, const char*)> get_inst_proc_addr = nullptr);

device::VulkanProfilingLayers* get_vulkan_profiling_layers();

// hasVulkanLoader returns true if Vulkan loader is found, otherwise returns
// false.
bool hasVulkanLoader();

// hasGLorGLES returns true if there is a OpenGL or OpenGL ES display driver
// that can be used.
bool hasGLorGLES();

// The functions below are used by getDeviceInstance(), and are implemented
// in the target-dependent sub-directories.

bool createContext();
const char* contextError();
void destroyContext();

// The functions below require a context to be created.

int numABIs();
void abi(int idx, device::ABI* abi);
device::ABI* currentABI();
device::MemoryLayout* currentMemoryLayout();

const char* hardwareName();

const char* cpuName();
const char* cpuVendor();
device::Architecture cpuArchitecture();
int cpuNumCores();

const char* gpuName();
const char* gpuVendor();

const char* instanceName();

// This queries the platform independent GL things.
void glDriver(device::OpenGLDriver*);
// This queries the platform depended GL things.
void glDriverPlatform(device::OpenGLDriver*);

device::OSKind osKind();
const char* osName();
const char* osBuild();
int osMajor();
int osMinor();
int osPoint();

bool hasAtrace();

}  // namespace query

#endif  // DEVICEINFO_QUERY_H
