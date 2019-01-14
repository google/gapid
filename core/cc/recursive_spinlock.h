/*
 * Copyright (C) 2019 Google Inc.
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

#include <atomic>
#include <thread>

#ifndef CORE_RECURSIVE_SPINLOCK_H__
#define CORE_RECURSIVE_SPINLOCK_H__

namespace core {
static const std::thread::id kUnlocked;
// RecursiveSpinLock is a spin lock implemented with atomic variable and
// operations. Mutiple calls to Lock in a single thread are valid.
class RecursiveSpinLock {
 public:
  RecursiveSpinLock() : owning_id_(kUnlocked), count_(0) {}
  // Lock acquires the lock.
  void Lock() {
    static thread_local std::thread::id this_thread =
        std::this_thread::get_id();
    // If ownining_id_ != this_thread, then it can never become this thread,
    //   behind out backs.
    if (owning_id_.load() != this_thread) {
      std::thread::id id = kUnlocked;
      while (!owning_id_.compare_exchange_weak(id, this_thread,
                                               std::memory_order_acquire,
                                               std::memory_order_relaxed)) {
        id = kUnlocked;
      }
    }
    ++count_;
  }
  // Unlock releases the lock.
  void Unlock() {
    if (--count_ == 0) {
      owning_id_.store(kUnlocked, std::memory_order_release);
    }
  }

 private:
  std::atomic<std::thread::id> owning_id_;
  // count_ does not have to be atomic, since it is only
  //        ever modified when locked.
  size_t count_;
};
}  // namespace core

#endif  // CORE_RECURSIVE_SPINLOCK_H__
