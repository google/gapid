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

#include <city.h>
#include <iostream>

namespace {

inline bool isLittleEndian() {
  union {
    uint32_t i;
    char c[4];
  } u;
  u.i = 0x01020304;
  return u.c[0] == 4;
}

template <typename T>
device::DataTypeLayout* new_dt_layout() {
  struct AlignmentStruct {
    char c;
    T t;
  };
  auto out = new device::DataTypeLayout();
  out->set_size(sizeof(T));
  out->set_alignment(offsetof(AlignmentStruct, t));
  return out;
}

void deviceInstanceID(device::Instance* instance) {
  if (instance == nullptr) {
    return;
  }
  instance->clear_id();
  auto id = instance->mutable_id();

  // Serialize the instance so we can hash it.
  auto proto_size = instance->ByteSizeLong();
  auto proto_data = new uint8_t[proto_size];
  instance->SerializeToArray(proto_data, proto_size);

  // Generate the ID from the serialized instance.
  // auto id = new device::ID();
  auto hash128 =
      CityHash128(reinterpret_cast<const char*>(proto_data), proto_size);
  auto hash32 =
      CityHash32(reinterpret_cast<const char*>(proto_data), proto_size);
  char id_data[20];
  memcpy(&id_data[0], &hash128, 16);
  memcpy(&id_data[16], &hash32, 4);
  id->set_data(id_data, sizeof(id_data));
  // instance->set_allocated_id(id);

  // Free the memory for the hashed instance.
  delete[] proto_data;
}

std::string getVendorName(uint32_t vendorId) {
  // A tiny little PCI-ID database.
  // (https://pcisig.com/membership/member-companies)
  switch (vendorId) {
    case 0x1022:
      return "AMD";
    case 0x10DE:
      return "NVIDIA";
    case 0x13B5:
      return "ARM";
    case 0x1AE0:
      return "Google";
    case 0x144D:
      return "Samsung";
    case 0x14E4:
      return "Broadcom";
    case 0x1F96:
      return "Intel";
    case 0x5143:
      return "Qualcomm";
    default:
      return "";
  }
}

}  // anonymous namespace

namespace query {

device::Instance* getDeviceInstance(const Option& opt, std::string* error) {
  using namespace device;
  using namespace google::protobuf::io;

  PlatformInfo osInfo;
  CpuInfo cpuInfo;
  if (!query::queryPlatform(&osInfo, error) ||
      !query::queryCpu(&cpuInfo, error)) {
    return nullptr;
  }

  // OS
  auto os = new OS();
  os->set_kind(osInfo.osKind);
  os->set_name(osInfo.osName);
  os->set_build(osInfo.osBuild);
  os->set_major_version(osInfo.osMajor);
  os->set_minor_version(osInfo.osMinor);
  os->set_point_version(osInfo.osPoint);

  // Instance.Configuration.Drivers
  auto drivers = new Drivers();

  std::string gpuVendor = "";
  std::string gpuName = "";
  uint32_t gpuDriverVersion = 0u;

  // Checks if the device supports Vulkan (have Vulkan loader) first, then
  // populates the VulkanDriver message.
  if (query::hasVulkanLoader()) {
    auto vulkan_driver = new VulkanDriver();
    if (opt.vulkan.query_layers_and_extensions()) {
      query::vkLayersAndExtensions(vulkan_driver);
    }
    if (opt.vulkan.query_physical_devices()) {
      query::vkPhysicalDevices(vulkan_driver);
      if (vulkan_driver->physical_devices_size() > 0) {
        gpuVendor =
            getVendorName(vulkan_driver->physical_devices(0).vendor_id());
        gpuName = vulkan_driver->physical_devices(0).device_name();
        gpuDriverVersion = vulkan_driver->physical_devices(0).driver_version();
      }
    }
    drivers->set_allocated_vulkan(vulkan_driver);
  }

  // Instance.Configuration.Hardware.CPU
  auto cpu = new CPU();
  cpu->set_name(cpuInfo.name);
  cpu->set_vendor(cpuInfo.vendor);
  cpu->set_architecture(cpuInfo.architecture);
  cpu->set_cores(osInfo.numCpuCores);

  // Instance.Configuration.Hardware.GPU
  auto gpu = new GPU();
  gpu->set_name(gpuName);
  gpu->set_vendor(gpuVendor);
  gpu->set_version(gpuDriverVersion);

  // Instance.Configuration.Hardware
  auto hardware = new Hardware();
  hardware->set_name(osInfo.hardwareName);
  hardware->set_allocated_cpu(cpu);
  hardware->set_allocated_gpu(gpu);

  // Instance.Configuration
  auto configuration = new Configuration();
  configuration->set_allocated_os(os);
  configuration->set_allocated_hardware(hardware);
  configuration->set_allocated_drivers(drivers);
  *configuration->mutable_abis() = {osInfo.abis.begin(), osInfo.abis.end()};

  auto perfetto_config = new PerfettoCapability();
  auto vulkan_performance_layers = query::get_vulkan_profiling_layers();
  if (vulkan_performance_layers) {
    perfetto_config->set_allocated_vulkan_profile_layers(
        vulkan_performance_layers);
  }
  perfetto_config->set_can_specify_atrace_apps(query::hasAtrace());
  configuration->set_allocated_perfetto_capability(perfetto_config);

  // Instance
  auto instance = new Instance();
  instance->set_name(osInfo.name);
  instance->set_allocated_configuration(configuration);
  deviceInstanceID(instance);

  return instance;
}

bool updateVulkanDriver(
    device::Instance* inst, size_t vk_inst_handle,
    std::function<void*(size_t, const char*)> get_inst_proc_addr) {
  using namespace device;
  using namespace google::protobuf::io;

  if (inst == nullptr) {
    return false;
  }

  device::VulkanDriver* vk_driver = new device::VulkanDriver();
  if (!query::vkLayersAndExtensions(vk_driver)) {
    // Failed at getting Vulkan layers and extensions info, the device may not
    // support Vulkan, return without touching the device::Instance.
    return false;
  }
  if (!query::vkPhysicalDevices(vk_driver, vk_inst_handle, get_inst_proc_addr,
                                false)) {
    // Failed at getting Vulkan physical device info, the device may not
    // support Vulkan, return without touching the device::Instance.
    return false;
  }

  if (!inst->has_configuration()) {
    auto conf = new Configuration();
    inst->set_allocated_configuration(conf);
  }
  if (!inst->configuration().has_drivers()) {
    auto drivers = new Drivers();
    inst->mutable_configuration()->set_allocated_drivers(drivers);
  }
  inst->mutable_configuration()->mutable_drivers()->set_allocated_vulkan(
      vk_driver);

  // rehash the ID
  deviceInstanceID(inst);
  return true;
}

device::MemoryLayout* currentMemoryLayout() {
  auto out = new device::MemoryLayout();
  out->set_endian(isLittleEndian() ? device::LittleEndian : device::BigEndian);
  out->set_allocated_pointer(new_dt_layout<void*>());
  out->set_allocated_integer(new_dt_layout<int>());
  out->set_allocated_size(new_dt_layout<size_t>());
  out->set_allocated_char_(new_dt_layout<char>());
  out->set_allocated_i64(new_dt_layout<int64_t>());
  out->set_allocated_i32(new_dt_layout<int32_t>());
  out->set_allocated_i16(new_dt_layout<int16_t>());
  out->set_allocated_i8(new_dt_layout<int8_t>());
  out->set_allocated_f64(new_dt_layout<double>());
  out->set_allocated_f32(new_dt_layout<float>());
  out->set_allocated_f16(new_dt_layout<uint16_t>());
  return out;
}

}  // namespace query
