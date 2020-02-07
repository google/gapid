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
#include "vk_lite.h"

#include "core/cc/assert.h"
#include "core/cc/dl_loader.h"
#include "core/cc/get_vulkan_proc_address.h"
#include "core/cc/target.h"

#include <functional>
#include <string>
#include <vector>

namespace query {

bool hasVulkanLoader() { return core::HasVulkanLoader(); }

#define MUST_SUCCESS(expr)                              \
  {                                                     \
    auto r = expr;                                      \
    if (VK_SUCCESS != r) {                              \
      GAPID_WARNING("Return: %d != VK_SUCCESS: " #expr  \
                    " for getting Vulkan Driver info.", \
                    r);                                 \
      return false;                                     \
    }                                                   \
  }

#define RETURN_IF_NOT_RESOLVED(FuncName)               \
  if (FuncName == nullptr) {                           \
    GAPID_WARNING("Failed at resolving: " #FuncName    \
                  " for getting Vulkan Driver info."); \
    return false;                                      \
  }

bool vkLayersAndExtensions(
    device::VulkanDriver* driver,
    std::function<void*(size_t, const char*)> get_inst_proc_addr) {
  if (!driver) {
    return false;
  }

// Resolve functions.
#define MUST_RESOLVE(FuncType, FuncName)                                    \
  FuncType FuncName = reinterpret_cast<FuncType>(                           \
      get_inst_proc_addr == nullptr ? core::GetVulkanProcAddress(#FuncName) \
                                    : get_inst_proc_addr(0, #FuncName));    \
  RETURN_IF_NOT_RESOLVED(FuncName)
  MUST_RESOLVE(PFNVKENUMERATEINSTANCELAYERPROPERTIES,
               vkEnumerateInstanceLayerProperties);
  MUST_RESOLVE(PFNVKENUMERATEINSTANCEEXTENSIONPROPERTIES,
               vkEnumerateInstanceExtensionProperties);
#undef MUST_RESOLVE
  // Layers and extensions supported by those layers.
  uint32_t layer_count = 0;
  MUST_SUCCESS(vkEnumerateInstanceLayerProperties(&layer_count, nullptr));
  std::vector<VkLayerProperties> inst_layer_props(layer_count,
                                                  VkLayerProperties{});
  MUST_SUCCESS(vkEnumerateInstanceLayerProperties(&layer_count,
                                                  inst_layer_props.data()));
  driver->clear_layers();
  for (size_t i = 0; i < inst_layer_props.size(); i++) {
    auto& l = inst_layer_props[i];
    uint32_t ext_count = 0;
    // Skip our layers.
    if (!strcmp(l.layerName, "GraphicsSpy") ||
        !strcmp(l.layerName, "VirtualSwapchain")) {
      continue;
    }
    MUST_SUCCESS(vkEnumerateInstanceExtensionProperties(l.layerName, &ext_count,
                                                        nullptr));
    std::vector<VkExtensionProperties> ext_props(ext_count,
                                                 VkExtensionProperties{});
    MUST_SUCCESS(vkEnumerateInstanceExtensionProperties(l.layerName, &ext_count,
                                                        ext_props.data()));
    driver->add_layers();
    driver->mutable_layers(driver->layers_size() - 1)->set_name(l.layerName);
    for (size_t j = 0; j < ext_props.size(); j++) {
      driver->mutable_layers(driver->layers_size() - 1)
          ->add_extensions(ext_props[j].extensionName);
    }
  }
  // For implicit layers and ICD extensions
  driver->clear_icd_and_implicit_layer_extensions();
  uint32_t ext_count = 0;
  MUST_SUCCESS(
      vkEnumerateInstanceExtensionProperties(nullptr, &ext_count, nullptr));
  std::vector<VkExtensionProperties> ext_props(ext_count,
                                               VkExtensionProperties{});
  MUST_SUCCESS(vkEnumerateInstanceExtensionProperties(nullptr, &ext_count,
                                                      ext_props.data()));
  for (size_t i = 0; i < ext_props.size(); i++) {
    driver->add_icd_and_implicit_layer_extensions(ext_props[i].extensionName);
  }
  return true;
}

bool vkPhysicalDevices(
    device::VulkanDriver* driver, size_t vk_inst_handle,
    std::function<void*(size_t, const char*)> get_inst_proc_addr,
    bool create_device) {
  if (!driver) {
    return false;
  }
  driver->clear_physical_devices();

// Resolve functions, create vkInstance handle if the given handle is NULL.
#define MUST_RESOLVE(FuncType, FuncName)                                  \
  FuncType FuncName = reinterpret_cast<FuncType>(                         \
      get_inst_proc_addr == nullptr                                       \
          ? core::GetVulkanInstanceProcAddress(vk_inst_handle, #FuncName) \
          : get_inst_proc_addr(vk_inst_handle, #FuncName));               \
  RETURN_IF_NOT_RESOLVED(FuncName)

  if (vk_inst_handle == 0) {
    MUST_RESOLVE(PFNVKCREATEINSTANCE, vkCreateInstance);
    VkInstanceCreateInfo inst_create_info{
        VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO,  // sType
        nullptr,                                 // pNext
        VkInstanceCreateFlags(0),                // flags
        nullptr,
        0,
        nullptr,
        0,
        nullptr};
    MUST_SUCCESS(vkCreateInstance(&inst_create_info, nullptr, &vk_inst_handle));
  }
  MUST_RESOLVE(PFNVKENUMERATEPHYSICALDEVICES, vkEnumeratePhysicalDevices);
  MUST_RESOLVE(PFNVKGETPHYSICALDEVICEPROPERTIES, vkGetPhysicalDeviceProperties);
  MUST_RESOLVE(PFNVKGETPHYSICALDEVICEQUEUEFAMILYPROPERTIES,
               vkGetPhysicalDeviceQueueFamilyProperties);
  MUST_RESOLVE(PFNVKCREATEDEVICE, vkCreateDevice);
#undef MUST_RESOLVE

  uint32_t phy_dev_count = 0;
  MUST_SUCCESS(
      vkEnumeratePhysicalDevices(vk_inst_handle, &phy_dev_count, nullptr));
  std::vector<VkPhysicalDevice> phy_devs(phy_dev_count, VkPhysicalDevice(0));
  MUST_SUCCESS(vkEnumeratePhysicalDevices(vk_inst_handle, &phy_dev_count,
                                          phy_devs.data()));

  for (size_t i = 0; i < phy_devs.size(); i++) {
    auto phy_dev = phy_devs[i];
    VkPhysicalDeviceProperties prop;
    vkGetPhysicalDeviceProperties(phy_dev, &prop);
    driver->add_physical_devices();
    driver->mutable_physical_devices(i)->set_api_version(prop.apiVersion);
    driver->mutable_physical_devices(i)->set_driver_version(prop.driverVersion);
    driver->mutable_physical_devices(i)->set_vendor_id(prop.vendorID);
    driver->mutable_physical_devices(i)->set_device_id(prop.deviceID);
    driver->mutable_physical_devices(i)->set_device_name(
        std::string(prop.deviceName));
    if (!create_device) {
      continue;
    }
    // Attempt to create VkDevice for every VkPhysicalDevice.
    uint32_t queue_family_count;
    vkGetPhysicalDeviceQueueFamilyProperties(phy_dev, &queue_family_count,
                                             nullptr);
    if (queue_family_count == 0) {
      continue;
    }
    std::vector<VkQueueFamilyProperties> queue_family_properties(
        queue_family_count, VkQueueFamilyProperties{});
    vkGetPhysicalDeviceQueueFamilyProperties(phy_dev, &queue_family_count,
                                             queue_family_properties.data());
    for (uint32_t j = 0; j < queue_family_count; ++j) {
      if ((queue_family_properties[j].queueFlags & VK_QUEUE_GRAPHICS_BIT) &&
          (queue_family_properties[j].queueFlags & VK_QUEUE_COMPUTE_BIT)) {
        float priority = 1.0f;
        VkDeviceQueueCreateInfo queue_create_info{
            VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
            nullptr,
            0,
            j,
            1,
            &priority,
        };
        VkDeviceCreateInfo create_info{
            VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO,
            nullptr,
            0,
            1,
            &queue_create_info,
            0,
            nullptr,
            0,
            nullptr,
            nullptr,
        };
        VkDevice device{};
        MUST_SUCCESS(vkCreateDevice(phy_dev, &create_info, nullptr, &device));
        break;
      }
    }
  }

  return true;
}

#undef RETURN_IF_NOT_RESOLVED
#undef MUST_SUCCESS
}  // namespace query
