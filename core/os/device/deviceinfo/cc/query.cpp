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

#include <thread>

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

void buildDeviceInstance(const query::Option& opt, device::Instance** out) {
  using namespace device;
  using namespace google::protobuf::io;

  if (!query::createContext()) {
    return;
  }

  // OS
  auto os = new OS();
  os->set_kind(query::osKind());
  os->set_name(query::osName());
  os->set_build(query::osBuild());
  os->set_major_version(query::osMajor());
  os->set_minor_version(query::osMinor());
  os->set_point_version(query::osPoint());

  // Instance.Configuration.Drivers
  auto drivers = new Drivers();

  const char* backupVendor = "";
  const char* backupName = "";
  if (query::hasGLorGLES()) {
    // Instance.Configuration.Drivers.OpenGLDriver
    auto opengl_driver = new OpenGLDriver();
    query::glDriver(opengl_driver);
    drivers->set_allocated_opengl(opengl_driver);
    backupVendor = opengl_driver->vendor().c_str();
    backupName = opengl_driver->renderer().c_str();
  }

  // Checks if the device supports Vulkan (have Vulkan loader) first, then
  // populates the VulkanDriver message.
  if (query::hasVulkanLoader()) {
    auto vulkan_driver = new VulkanDriver();
    if (opt.vulkan.query_layers_and_extensions()) {
      query::vkLayersAndExtensions(vulkan_driver);
    }
    if (opt.vulkan.query_physical_devices()) {
      query::vkPhysicalDevices(vulkan_driver);
      if (strlen(backupName) == 0 &&
          vulkan_driver->physical_devices_size() > 0) {
        backupName = vulkan_driver->physical_devices(0).device_name().c_str();
      }
    }
    drivers->set_allocated_vulkan(vulkan_driver);
  }

  // Instance.Configuration.Hardware.CPU
  auto cpu = new CPU();
  cpu->set_name(query::cpuName());
  cpu->set_vendor(query::cpuVendor());
  cpu->set_architecture(query::cpuArchitecture());
  cpu->set_cores(query::cpuNumCores());

  // Instance.Configuration.Hardware.GPU
  auto gpu = new GPU();
  const char* gpuName = query::gpuName();
  const char* gpuVendor = query::gpuVendor();
  if (strlen(gpuName) == 0) {
    gpuName = backupName;
  }
  if (strlen(gpuVendor) == 0) {
    gpuVendor = backupVendor;
  }
  gpu->set_name(gpuName);
  gpu->set_vendor(gpuVendor);

  // Instance.Configuration.Hardware
  auto hardware = new Hardware();
  hardware->set_name(query::hardwareName());
  hardware->set_allocated_cpu(cpu);
  hardware->set_allocated_gpu(gpu);

  // Instance.Configuration
  auto configuration = new Configuration();
  configuration->set_allocated_os(os);
  configuration->set_allocated_hardware(hardware);
  configuration->set_allocated_drivers(drivers);
  for (int i = 0, c = query::numABIs(); i < c; i++) {
    query::abi(i, configuration->add_abis());
  }

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
  instance->set_name(query::instanceName());
  instance->set_allocated_configuration(configuration);
  deviceInstanceID(instance);

  // Blacklist of OS/Hardware version that means we cannot safely
  // destroy the context.
  // https://github.com/google/gapid/issues/1867
  bool blacklist = false;
  if (std::string(gpuName).find("Vega") != std::string::npos &&
      std::string(query::osName()).find("Windows 10") != std::string::npos) {
    blacklist = true;
  }

  if (!blacklist) {
    query::destroyContext();
  }

  *out = instance;
}

}  // anonymous namespace

namespace query {

device::Instance* getDeviceInstance(const Option& opt) {
  device::Instance* instance = nullptr;

  // buildDeviceInstance on a separate thread to avoid EGL screwing with the
  // currently bound context.
  std::thread thread(buildDeviceInstance, opt, &instance);
  thread.join();
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
