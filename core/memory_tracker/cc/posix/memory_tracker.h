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

#ifndef GAPII_MEMORY_TRACKER_POSIX_H
#define GAPII_MEMORY_TRACKER_POSIX_H

#include "core/memory_tracker/cc/memory_protections.h"

#include <cstdint>
#include <signal.h>
#include <stdlib.h>
#include <sys/mman.h>
#include <unistd.h>


namespace gapii {
namespace track_memory {

inline bool set_protection(void* p, size_t size, PageProtections prot) {
  uint32_t protections = (prot == PageProtections::kRead) ? PROT_READ :
                    (prot == PageProtections::kWrite) ? PROT_WRITE :
                (prot == PageProtections::kReadWrite) ? PROT_READ | PROT_WRITE: 0;
  return mprotect(p, size, protections); 
}

// SignalBlocker blocks the specified signal when it is constructed, and
// unblock the signal when it is destroyed.
class SignalBlocker {
 public:
  SignalBlocker(int sig) : set_{0}, old_set_{0} {
    sigemptyset(&set_);
    sigaddset(&set_, sig);
    pthread_sigmask(SIG_BLOCK, &set_, &old_set_);
  }
  ~SignalBlocker() { pthread_sigmask(SIG_SETMASK, &old_set_, nullptr); }
  // Not copyable, not movable.
  SignalBlocker(const SignalBlocker&) = delete;
  SignalBlocker(SignalBlocker&&) = delete;
  SignalBlocker& operator=(const SignalBlocker&) = delete;
  SignalBlocker& operator=(SignalBlocker&&) = delete;

 private:
  sigset_t set_;
  sigset_t old_set_;
};

inline uint32_t GetPageSize() {
  return getpagesize();
}

} // namespace track_memory
} // namespace gapii

#endif // GAPII_MEMORY_TRACKER_POSIX_H