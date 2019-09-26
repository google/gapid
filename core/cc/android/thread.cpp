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

#include <sys/syscall.h>
#include <unistd.h>
#include <fstream>
#include <iostream>

namespace core {

Thread Thread::current() {
  pid_t tid = syscall(SYS_gettid);
  return Thread(static_cast<uint64_t>(static_cast<uintptr_t>(tid)));
}

std::string Thread::get_name() const {
  std::ifstream comm{"/proc/self/task/" + std::to_string(mId) + "/comm"};
  std::string name;
  getline(comm, name);
  return name;
}

}  // namespace core
