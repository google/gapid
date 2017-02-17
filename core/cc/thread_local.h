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

#ifndef CORE_THREAD_LOCAL_H
#define CORE_THREAD_LOCAL_H

#include <cstddef>
#include "core/cc/target.h"

namespace core {

// Stores a single size_t value per-thread.
// The value is initially 0 for every thread that reads it.
class ThreadLocalValue {
public:
    ThreadLocalValue();
    ~ThreadLocalValue();

    // Sets the value to be returned on subsequent calls to get() by the
    // current thread.
    void set(size_t val);

    // Returns the last value that was provided to the set() method by
    // the current thread. If the current thread has never called set(),
    // returns 0.
    size_t get();
protected:
    void* _; // A pointer to the OS TLS object.
};
}  // namespace core

#endif  // CORE_THREAD_LOCAL_H
