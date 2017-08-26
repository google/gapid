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

#ifndef VK_VIRTUAL_SWAPCHAIN_THREADING_H_
#define VK_VIRTUAL_SWAPCHAIN_THREADING_H_

#ifdef _WIN32
// We need to define these to get CONDITION_VARIABLE
#undef _WIN32_WINNT
#define _WIN32_WINNT 0x0600
#include <windows.h>
#else
#include <pthread.h>
#endif

#include <mutex>

namespace swapchain {
namespace threading {

class mutex {
public:
  mutex(const mutex &) = delete;
  mutex &operator=(const mutex &) = delete;
  mutex() {
#ifdef _WIN32
    InitializeCriticalSection(&mutex_);
#else
    pthread_mutex_init(&mutex_, nullptr);
#endif
  }

  ~mutex() {
#ifdef _WIN32
    DeleteCriticalSection(&mutex_);
#else
    pthread_mutex_destroy(&mutex_);
#endif
  }

  void lock() {
#ifdef _WIN32
    EnterCriticalSection(&mutex_);
#else
    pthread_mutex_lock(&mutex_);
#endif
  }

  void unlock() {
#ifdef _WIN32
    LeaveCriticalSection(&mutex_);
#else
    pthread_mutex_unlock(&mutex_);
#endif
  }

#ifdef _WIN32
  CRITICAL_SECTION &native_handle() { return mutex_; }
#else
  pthread_mutex_t &native_handle() { return mutex_; }
#endif

private:
#ifdef _WIN32
  CRITICAL_SECTION mutex_;
#else
  pthread_mutex_t mutex_;
#endif
};

enum class cv_status { timeout, no_timeout };

class condition_variable {
public:
  condition_variable(const condition_variable &) = delete;
  condition_variable &operator=(const condition_variable &) = delete;
  condition_variable() {
#ifdef _WIN32
    InitializeConditionVariable(&condition_);
#else
    pthread_cond_init(&condition_, nullptr);
#endif
  }
  ~condition_variable() {
#ifdef _WIN32
    InitializeConditionVariable(&condition_);
#else
    pthread_cond_destroy(&condition_);
#endif
  }
  template <class Rep, class Period>
  cv_status wait_for(std::unique_lock<mutex> &lock,
                     const std::chrono::duration<Rep, Period> &rel_time) {
    auto &native_handle = lock.mutex()->native_handle();
#ifdef _WIN32
    auto time =
        std::chrono::duration_cast<std::chrono::milliseconds>(rel_time).count();
    return (0 != SleepConditionVariableCS(&condition_, &native_handle, time))
               ? cv_status::no_timeout
               : cv_status::timeout;
#else
    timespec tv;
    const std::chrono::seconds sec =
        std::chrono::duration_cast<std::chrono::seconds>(rel_time);
    tv.tv_sec = sec.count();
    tv.tv_nsec =
        std::chrono::duration_cast<std::chrono::nanoseconds>(rel_time - sec)
            .count();

    return (0 == pthread_cond_timedwait(&condition_, &native_handle, &tv))
               ? cv_status::no_timeout
               : cv_status::timeout;
#endif
  }

  void wait(std::unique_lock<mutex> &lock) {
    auto &native_handle = lock.mutex()->native_handle();
#ifdef _WIN32
    SleepConditionVariableCS(&condition_, &native_handle, INFINITE);
#else
    pthread_cond_wait(&condition_, &native_handle);
#endif
  }

  void notify_one() {
#ifdef _WIN32
    WakeConditionVariable(&condition_);
#else
    pthread_cond_signal(&condition_);
#endif
  }

  void notify_all() {
#ifdef _WIN32
    WakeAllConditionVariable(&condition_);
#else
    pthread_cond_broadcast(&condition_);
#endif
  }

private:
#ifdef _WIN32
  CONDITION_VARIABLE condition_;
#else
  pthread_cond_t condition_;
#endif
};

} // threading
} // swapchain
#endif // VK_VIRTUAL_SWAPCHAIN_THREADING_H_