/*
 * Copyright (C) 2018 Google Inc.
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

#include <cstdint>
#include <cstring>
#include <iostream>
#include <string>

#if _WIN32
#include <fcntl.h>
#include <stdio.h>
#endif

#include <google/protobuf/util/json_util.h>
#include "core/cc/log.h"
#include "core/os/device/device.pb.h"
#include "core/os/device/deviceinfo/cc/instance.h"

void print_help() {
  std::cout << "Usage: device-info [--binary]" << std::endl;
  std::cout << "Output information about the current device." << std::endl;
  std::cout << " --binary         Output a binary protobuf instead of json"
            << std::endl;
}

int main(int argc, char const* argv[]) {
  bool output_binary = false;
  for (int i = 1; i < argc; ++i) {
    if (strcmp(argv[i], "--help") == 0 || strcmp(argv[i], "-help") == 0 ||
        strcmp(argv[i], "-h") == 0) {
      print_help();
      return 0;
    } else if (strcmp(argv[i], "--binary") == 0) {
      output_binary = true;
    } else {
      print_help();
      return -1;
    }
  }

  device_instance instance = get_device_instance();
  if (output_binary) {
#if _WIN32
    _setmode(_fileno(stdout), _O_BINARY);
#endif
    std::cout << std::string(reinterpret_cast<char*>(instance.data),
                             instance.size);
  } else {
    device::Instance device_inst;
    if (!device_inst.ParseFromArray(instance.data, instance.size)) {
      GAPID_ERROR("Internal error");
      free_device_instance(instance);
      return -1;
    }
    std::string output;
    google::protobuf::util::JsonPrintOptions options;
    options.add_whitespace = true;
    if (!google::protobuf::util::MessageToJsonString(device_inst, &output,
                                                     options)
             .ok()) {
      GAPID_ERROR("Internal error: Could not convert to json");
      free_device_instance(instance);
      return -1;
    }
    std::cout << output;
  }

  free_device_instance(instance);

  return 0;
}
