/*
 * Copyright (C) 2022 Google Inc.
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

#include "utils.h"

#include <string>

#include "zircon/process.h"
#include "zircon/syscalls.h"
#include "zircon/syscalls/object.h"
#include "zircon/threads.h"

namespace core {

std::string get_process_name() {
  char name[ZX_MAX_NAME_LEN];
  if (zx_object_get_property(zx_process_self(), ZX_PROP_NAME, name,
                             sizeof(name)) == ZX_OK) {
    return std::string(name);
  }
  fprintf(stderr, "ERROR: Unable to get process name.\n");
  return std::string();
}

uint64_t get_process_id() {
  uint64_t koid = 0;
  if (KoidFromHandle(zx_process_self(), &koid)) {
    return koid;
  }
  fprintf(stderr, "ERROR: Unable to get process id.\n");
  return 0;
}

}  // namespace core
