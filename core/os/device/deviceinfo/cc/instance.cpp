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
#include "core/cc/log.h"
#include "query.h"

namespace {

std::string error;

}  // anonymous namespace

extern "C" {

device_instance get_device_instance() {
  device_instance out = {};

  query::Option query_opt;
  query_opt.vulkan.set_query_layers_and_extensions(true)
      .set_query_physical_devices(true);
  error.clear();
  auto instance = query::getDeviceInstance(query_opt, &error);
  if (!instance) {
    GAPID_ERROR("Failed to query device info: %s", error.c_str());
    return out;
  }

  // Reserialize the instance with the ID field.
  out.size = instance->ByteSizeLong();
  out.data = new uint8_t[out.size];
  instance->SerializeToArray(out.data, out.size);

  delete instance;

  return out;
}

const char* get_device_instance_error() { return error.c_str(); }

void free_device_instance(device_instance di) { delete[] di.data; }

}  // extern "C"
