/*
 * Copyright (C) 2019 Google Inc.
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

#include <unistd.h>
#include <fstream>
#include <streambuf>
#include <string>

namespace core {
static const size_t MAX_PATH = 4096;

std::string get_process_name() {
  std::ifstream t("/proc/self/cmdline");
  std::string name;
  // Watch out: the string returned by reading /proc/self/cmdline contains '\0'
  // characters to delimit the command line arguments (see man proc), make sure
  // to stop at the first '\0' to only extract the process name.
  auto it = std::istreambuf_iterator<char>(t);
  while (it != std::istreambuf_iterator<char>() && *it != '\0') {
    name += *it++;
  }
  return name;
}

uint64_t get_process_id() { return static_cast<uint64_t>(getpid()); }

}  // namespace core
