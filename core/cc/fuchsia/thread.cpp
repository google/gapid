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

#include "core/cc/thread.h"

#include "zircon/syscalls.h"
#include "zircon/syscalls/object.h"
#include "zircon/threads.h"

#include "utils.h"
namespace core {

Thread Thread::current() {
  uint64_t koid = 0;
  if (KoidFromHandle(thrd_get_zx_handle(thrd_current()), &koid)) {
    return Thread(koid);
  }
  fprintf(stderr, "ERROR: Unable to get current thread id.\n");
  return Thread(0);
}

std::string Thread::get_name() const {
  char name[ZX_MAX_NAME_LEN];
  zx_status_t status = zx_object_get_property(
      thrd_get_zx_handle(thrd_current()), ZX_PROP_NAME, name, sizeof(name));
  if (ZX_OK == status) {
    return std::string(name);
  }
  fprintf(stderr, "ERROR: Unable to get current thread name.\n");
  return std::string();
}

}  // namespace core
