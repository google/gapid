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

#include <functional>

namespace gapii {
namespace track_memory {

class PosixMemoryTracker {
public:
  PosixMemoryTracker(std::function<bool(void *)> segfault_function)
      : orig_action_(), handle_segfault_(segfault_function) {}

  bool IsInstalled() const {
    struct sigaction orig_action = {};
    sigaction(SIGSEGV, nullptr, &orig_action);
    return orig_action.sa_sigaction == &SegfaultHandlerFunction;
  }

protected:
  // A static wrapper of HandleSegfault() as sigaction() asks for a static
  // function.
  static void SegfaultHandlerFunction(int sig, siginfo_t *info, void *unused);

  // EnableMemoryTrackerImpl calls sigaction() to register the new segfault
  // handler to the thread (and affects all the child threads), stores the
  // original segfault handler. This method sets the static pointer:
  // unique_tracker to |this| pointer. The signal handler will not be set again
  // if the signal handler has already been set by the same memory tracker
  // instance before, all following calls to this function will just return
  // true.
  bool inline EnableMemoryTrackerImpl();

  // DisableMemoryTrackerImpl recovers the original segfault signal
  // handler. Returns true if the handler is recovered successfully,
  // othwerwise returns false.
  bool inline DisableMemoryTrackerImpl();

private:
  struct sigaction orig_action_; // The original signal action for SIGSEGV
  std::function<bool(void *)>
      handle_segfault_; // The function to call on a segfault
};

typedef MemoryTrackerImpl<PosixMemoryTracker> MemoryTracker;
extern PosixMemoryTracker *unique_tracker;

void inline PosixMemoryTracker::SegfaultHandlerFunction(int sig,
                                                        siginfo_t *info,
                                                        void *unused) {
  if (unique_tracker) {
    if (!unique_tracker->handle_segfault_(info->si_addr)) {
#ifndef NDEBUG
      raise(SIGTRAP);
#endif // NDEBUG
      (*unique_tracker->orig_action_.sa_sigaction)(sig, info, unused);
    }
  }
}

bool inline PosixMemoryTracker::EnableMemoryTrackerImpl() {
  if (IsInstalled()) {
    return true;
  }

  unique_tracker = this;
  struct sigaction sa = {};
  sa.sa_flags = SA_SIGINFO;
  sigemptyset(&sa.sa_mask);
  sa.sa_sigaction = &SegfaultHandlerFunction;
  return sigaction(SIGSEGV, &sa, &orig_action_) != 1;
}

bool inline PosixMemoryTracker::DisableMemoryTrackerImpl() {
  if (IsInstalled()) {
    return sigaction(SIGSEGV, &orig_action_, nullptr) != 1;
  }
  return true;
}

} // namespace track_memory
} // namespace gapii
