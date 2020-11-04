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

#ifndef CORE_THREAD_H
#define CORE_THREAD_H

#include <stdint.h>
#include <functional>
#include <string>

namespace core {

// Thread represents a single thread of execution in the process.
class Thread {
 public:
  // current returns the Thread representing the current thread of execution.
  static Thread current();

  // id returns the process-unique identifier for the Thread.
  inline uint64_t id() const;

  std::string get_name() const;

 private:
  inline Thread(uint64_t id);

  uint64_t mId;
};

inline Thread::Thread(uint64_t id) : mId(id) {}

inline uint64_t Thread::id() const { return mId; }

class AsyncJob {
 public:
  AsyncJob(const std::function<void()>& function);
  ~AsyncJob();  // Waits on the thread to finish.
 private:
  static void* RunJob(void* _data) {
    AsyncJob* job = reinterpret_cast<AsyncJob*>(_data);
    job->mFunction();
    return nullptr;
  }
  std::function<void()> mFunction;
  void* _;  // A pointer to the OS thread object.
};

}  // namespace core

#endif  // CORE_THREAD_H
