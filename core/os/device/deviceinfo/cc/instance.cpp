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

#include "instance.h"
#include "query.h"

#include <city.h>

extern "C" {

device_instance get_device_instance(void* platform_data) {
    using namespace device;
    using namespace google::protobuf::io;

    if (!query::createContext(platform_data)) {
        return device_instance{nullptr, 0};
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
    instance->set_serial(query::instanceSerial());
    instance->set_name(query::instanceName());
    instance->set_allocated_configuration(configuration);

    // Serialize the instance so we can hash it.
    device_instance out;
    out.size = instance->ByteSize();
    out.data = new uint8_t[out.size];
    instance->SerializeToArray(out.data, out.size);

    // Generate the ID from the serialized instance.
    auto id = new ID();
    auto hash128 = CityHash128(reinterpret_cast<const char*>(out.data), out.size);
    auto hash32 = CityHash32(reinterpret_cast<const char*>(out.data), out.size);
    char id_data[20];
    memcpy(&id_data[0], &hash128, 16);
    memcpy(&id_data[16], &hash32, 4);
    id->set_data(id_data, sizeof(id_data));
    instance->set_allocated_id(id);

    // Free the memory for the hashed instance.
    delete [] out.data;

    // Reserialize the instance with the ID field.
    out.size = instance->ByteSize();
    out.data = new uint8_t[out.size];
    instance->SerializeToArray(out.data, out.size);

    query::destroyContext();

    return out;
}

const char* get_device_instance_error() {
    return query::contextError();
}

void free_device_instance(device_instance di) {
    delete[] di.data;
}

} // extern "C"
