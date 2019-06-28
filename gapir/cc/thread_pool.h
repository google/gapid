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

#ifndef GAPIR_THREAD_POOL_H
#define GAPIR_THREAD_POOL_H

#include <functional>
#include <mutex>
#include <queue>
#include <thread>
#include <unordered_map>

#include "core/cc/semaphore.h"

namespace gapir {

typedef uint64_t ThreadID;

// ThreadPool holds a number of threads that can have work assigned to them.
class ThreadPool {
 public:
  typedef std::function<void()> F;

  ThreadPool();

  // Destructor. Waits for all threads to finish their work before returning.
  ~ThreadPool();

  // Appends F to the queue of work for the thread with the given ID.
  // If this is the first time enqueue has been called with the given thread
  // ID then it is created.
  void enqueue(ThreadID, const F&);

 private:
  class Thread {
   public:
    Thread();
    ~Thread();
    void enqueue(const F&);

   private:
    Thread(const Thread&) = delete;
    Thread& operator=(const Thread&) = delete;

    static void worker(Thread*);

    std::thread* mThread;        // The thread.
    std::queue<F> mWork;         // The queue of pending work.
    std::mutex mMutex;           // Guards mWork.
    core::Semaphore mSemaphore;  // Number of work items.
  };

  ThreadPool(const ThreadPool&) = delete;
  ThreadPool& operator=(const ThreadPool&) = delete;

  std::mutex mMutex;  // Guards mThreads
  std::unordered_map<ThreadID, Thread*> mThreads;
};

}  // namespace gapir

#endif  // GAPIR_THREAD_POOL_H
