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

#include <cstdint>
#include <cstdlib>

namespace core {

AsyncJob::AsyncJob(const std::function<void()>& function)
    : mFunction(function) {
  _ = malloc(sizeof(pthread_t));
  pthread_t* thread = reinterpret_cast<pthread_t*>(_);

  pthread_create(thread, nullptr, &AsyncJob::RunJob, this);
}

AsyncJob::~AsyncJob() {
  pthread_t* thread = reinterpret_cast<pthread_t*>(_);
  pthread_join(*thread, nullptr);
  free(_);
}

}  // namespace core
