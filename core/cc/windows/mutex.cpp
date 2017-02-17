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

#include "core/cc/mutex.h"

#include <windows.h>

namespace core {

Mutex::Mutex() {
    // Not really a mutex, but is faster and provides us with what we need.
    auto cs = new CRITICAL_SECTION();
    InitializeCriticalSection(cs);
    _ = cs;
}

Mutex::~Mutex() {
    auto cs = reinterpret_cast<CRITICAL_SECTION*>(_);
    DeleteCriticalSection(cs);
    delete cs;
}

void Mutex::lock() {
    auto cs = reinterpret_cast<CRITICAL_SECTION*>(_);
    EnterCriticalSection(cs);
}

void Mutex::unlock() {
    auto cs = reinterpret_cast<CRITICAL_SECTION*>(_);
    LeaveCriticalSection(cs);
}

}  // namespace core
