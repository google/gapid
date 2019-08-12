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

#include <libgen.h>
#include <unistd.h>
#include <cstring>
#include <string>

namespace core {
static const size_t MAX_PATH = 4096;

std::string get_process_name() {
  char mp[MAX_PATH + 1];
  memset(mp, 0, MAX_PATH + 1);
  ssize_t len = readlink("/proc/self/exe", mp, MAX_PATH);
  if (len <= 0) {
    return "";
  }
  return basename(mp);
}

}  // namespace core
