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

void buildDeviceInstance(void* platform_data, device::Instance** out) {
    using namespace device;
    using namespace google::protobuf::io;

    if (!query::createContext(platform_data)) {
        return;
    }

    // OS
    auto os = new OS();
    os->set_kind(query::osKind());
	os->set_name(query::osName());
	os->set_build(query::osBuild());
	os->set_major(query::osMajor());
	os->set_minor(query::osMinor());
	os->set_point(query::osPoint());

    // Instance.Configuration.Drivers.OpenGLDriver
    auto opengl_driver = new OpenGLDriver();
    query::glDriver(opengl_driver);

    // Instance.Configuration.Drivers.VulkanDriver
    auto vulkan_driver = new VulkanDriver();

    // Instance.Configuration.Drivers
    auto drivers = new Drivers();
    drivers->set_allocated_opengl(opengl_driver);
    drivers->set_allocated_vulkan(vulkan_driver);

    // Instance.Configuration.Hardware.CPU
    auto cpu = new CPU();
    cpu->set_name(query::cpuName());
    cpu->set_vendor(query::cpuVendor());
    cpu->set_architecture(query::cpuArchitecture());
    cpu->set_cores(query::cpuNumCores());

    // Instance.Configuration.Hardware.GPU
    auto gpu = new GPU();
    gpu->set_name(query::gpuName());
    gpu->set_vendor(query::gpuVendor());

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

    // Instance
    auto instance = new Instance();
    instance->set_name(query::instanceName());
    instance->set_allocated_configuration(configuration);

    // Serialize the instance so we can hash it.
    auto proto_size = instance->ByteSize();
    auto proto_data = new uint8_t[proto_size];
    instance->SerializeToArray(proto_data, proto_size);

    // Generate the ID from the serialized instance.
    auto id = new ID();
    auto hash128 = CityHash128(reinterpret_cast<const char*>(proto_data), proto_size);
    auto hash32 = CityHash32(reinterpret_cast<const char*>(proto_data), proto_size);
    char id_data[20];
    memcpy(&id_data[0], &hash128, 16);
    memcpy(&id_data[16], &hash32, 4);
    id->set_data(id_data, sizeof(id_data));
    instance->set_allocated_id(id);

    // Free the memory for the hashed instance.
    delete [] proto_data;

    query::destroyContext();

    *out = instance;
}

}  // anonymous namespace

namespace query {

device::Instance* getDeviceInstance(void* platform_data) {
    device::Instance* instance = nullptr;

    // buildDeviceInstance on a seperate thread to avoid EGL screwing with the
    // currently bound context.
    std::thread thread(buildDeviceInstance, platform_data, &instance);
    thread.join();

    return instance;
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
