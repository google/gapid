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

#include <windows.h>

namespace core {

Thread Thread::current() {
  auto thread = GetCurrentThreadId();
  return Thread(static_cast<uint64_t>(thread));
}

AsyncJob::AsyncJob(const std::function<void()>& function)
    : mFunction(function) {
  LPTHREAD_START_ROUTINE start = [](void* _this) -> long unsigned int {
    AsyncJob::RunJob(_this);
    return 0;
  };
  _ = reinterpret_cast<void*>(
      CreateThread(nullptr, 0, start, this, 0, nullptr));
}

AsyncJob::~AsyncJob() {
  WaitForSingleObject(reinterpret_cast<HANDLE>(_), INFINITE);
}

std::string Thread::get_name() const {
  // Only starting in windows 10 are thread names supported.
  return std::to_string(mId);
}

}  // namespace core
