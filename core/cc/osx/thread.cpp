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

#include "core/cc/thread.h"

#include <pthread.h>
#include <string>

namespace core {

Thread Thread::current() {
  auto thread = pthread_self();
  return Thread(static_cast<uint64_t>(reinterpret_cast<uintptr_t>(thread)));
}

std::string Thread::get_name() const {
  char name[17] = {'\0'};

  if (0 ==
      pthread_getname_np(
          reinterpret_cast<pthread_t>(static_cast<uintptr_t>(mId)), name, 16)) {
    return name;
  }
  return std::to_string(mId);
}

}  // namespace core
