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

#ifndef GAPII_MEMORY_TRACKER_WINDOWS_H
#define GAPII_MEMORY_TRACKER_WINDOWS_H

#include <atomic>
#include <cstdint>

#include "core/memory_tracker/cc/memory_protections.h"

namespace gapii {
namespace track_memory {

bool set_protection(void* p, size_t size, PageProtections prot);
#define IS_POSIX 0
// SignalBlocker is a no-op on Windows.
class SignalBlocker {
 public:
  SignalBlocker(int) {}
  ~SignalBlocker() {}
  // Not copyable, not movable.
  SignalBlocker(const SignalBlocker&) = delete;
  SignalBlocker(SignalBlocker&&) = delete;
  SignalBlocker& operator=(const SignalBlocker&) = delete;
  SignalBlocker& operator=(SignalBlocker&&) = delete;
};

uint32_t GetPageSize();

}  // namespace track_memory
}  // namespace gapii

#endif  // GAPII_MEMORY_TRACKER_WINDOWS_H