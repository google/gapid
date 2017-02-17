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
#include <windows.h>

namespace core {

ThreadLocalValue::ThreadLocalValue() {
    DWORD key = TlsAlloc();
    _ = reinterpret_cast<void*>(static_cast<uintptr_t>(key));
}

ThreadLocalValue::~ThreadLocalValue() {
    DWORD key = static_cast<DWORD>(reinterpret_cast<uintptr_t>(_));
    TlsFree(key);
}

size_t ThreadLocalValue::get() {
    DWORD key = static_cast<DWORD>(reinterpret_cast<uintptr_t>(_));
    void* val = TlsGetValue(key);
    return reinterpret_cast<size_t>(val);
}

void ThreadLocalValue::set(size_t val) {
    DWORD key = static_cast<DWORD>(reinterpret_cast<uintptr_t>(_));
    TlsSetValue(key, reinterpret_cast<void*>(static_cast<uintptr_t>(val)));
}

}  // namespace core

