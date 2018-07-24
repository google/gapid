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

#ifndef CORE_SEMAPHORE_H
#define CORE_SEMAPHORE_H

#include <condition_variable>
#include <mutex>

namespace core {

class Semaphore {
 public:
  inline Semaphore(unsigned int inital_count = 0);

  // acquire takes count counters from the semaphore, blocking until count are
  // available, or the semaphore is closed.
  // If the semaphore is closed false is returned.
  inline bool acquire(int count = 1);

  // acquire_all takes all counters from the semaphore, blocking until at least
  // one is available, or the semaphore is closed. The number of counters
  // acquired is returned, or -1 if the semaphore is closed.
  inline int acquire_all();

  // wait_until blocks until the counter reaches at least count, without taking
  // any counters from the semaphore.
  // If the semaphore is closed false is returned.
  inline bool wait_until(int count = 1);

  // release returns count counters to the semaphore, possibly unblocking a call
  // to acquire().
  // If the semaphore is closed false is returned.
  inline bool release(int count = 1);

  // close unblocks any blocking calls on the semaphore, and makes all future
  // calls return their closed value.
  inline void close();

 private:
  Semaphore(const Semaphore&) = delete;
  Semaphore& operator=(const Semaphore&) = delete;

  std::mutex mutex_;
  std::condition_variable signal_;
  int count_;
  bool closed_;
};

Semaphore::Semaphore(unsigned int inital_count /* = 0 */)
    : count_(inital_count), closed_(false) {}

inline bool Semaphore::acquire(int count /* = 1 */) {
  std::unique_lock<std::mutex> lock(mutex_);
  signal_.wait(lock,
               [this, count] { return this->count_ >= count || closed_; });
  count_ -= count;
  return !closed_;
}

inline int Semaphore::acquire_all() {
  std::unique_lock<std::mutex> lock(mutex_);
  signal_.wait(lock, [this] { return this->count_ > 0 || closed_; });
  auto count = count_;
  count_ = 0;
  if (closed_) {
    return -1;
  }
  return count;
}

inline bool Semaphore::wait_until(int count /* = 1 */) {
  std::unique_lock<std::mutex> lock(mutex_);
  signal_.wait(lock,
               [this, count] { return this->count_ >= count || closed_; });
  return !closed_;
}

inline bool Semaphore::release(int count /* = 1 */) {
  std::unique_lock<std::mutex> lock(mutex_);
  count_ += count;
  signal_.notify_all();
  return !closed_;
}

inline void Semaphore::close() {
  std::unique_lock<std::mutex> lock(mutex_);
  closed_ = true;
  signal_.notify_all();
}

}  // namespace core

#endif  // CORE_SEMAPHORE_H
