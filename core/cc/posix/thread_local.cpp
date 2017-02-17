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

#include "core/cc/thread_local.h"

#include <cstdint>
#include <pthread.h>

namespace core {

ThreadLocalValue::ThreadLocalValue() {
    auto tls = new pthread_key_t();
    pthread_key_create(tls, NULL);
    _ = tls;
}

ThreadLocalValue::~ThreadLocalValue() {
   auto tls = reinterpret_cast<pthread_key_t*>(_);
   pthread_key_delete(*tls);
   delete tls;
}

size_t ThreadLocalValue::get() {
    auto tls = reinterpret_cast<pthread_key_t*>(_);
    void* val = pthread_getspecific(*tls);
    return static_cast<size_t>(reinterpret_cast<uintptr_t>(val));
}

void ThreadLocalValue::set(size_t val) {
    auto tls = reinterpret_cast<pthread_key_t*>(_);
    (void) pthread_setspecific(*tls, reinterpret_cast<void*>(static_cast<uintptr_t>(val)));
}

}  // namespace core

