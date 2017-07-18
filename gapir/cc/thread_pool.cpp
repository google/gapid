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

#include "thread_pool.h"

#include <functional>
#include <queue>

namespace gapir {

ThreadPool::ThreadPool() {

}

ThreadPool::~ThreadPool() {
    for (auto it : mThreads) {
        delete it.second;
    }
}

void ThreadPool::enqueue(ThreadID id, const F& work) {
    std::lock_guard<std::mutex> lock(mMutex);
    auto it = mThreads.find(id);
    if (it == mThreads.end()) {
        auto thread = new Thread();
        thread->enqueue(work);
        mThreads[id] = thread;
    } else {
        it->second->enqueue(work);
    }
}

void ThreadPool::Thread::worker(Thread* thread) {
    while (true) {
        thread->mSemaphore.acquire();

        std::unique_lock<std::mutex> lock(thread->mMutex);
        if (thread->mWork.size() == 0) {
            return; // Semaphore signal with no work means exit thread.
        }
        auto work = thread->mWork.front();
        thread->mWork.pop();
        lock.unlock();

        work();
    };
}

ThreadPool::Thread::Thread() {
    mThread = new std::thread(Thread::worker, this);
}

ThreadPool::Thread::~Thread() {
    mSemaphore.release();
    mThread->join();
    delete mThread;
}

void ThreadPool::Thread::enqueue(const F& work) {
    mMutex.lock();
    mWork.push(work);
    mMutex.unlock();
    mSemaphore.release();
}

}  // namespace gapir
