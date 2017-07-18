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

#include <mutex>
#include <condition_variable>

namespace core {

class Semaphore {
public:
    inline Semaphore(unsigned int count = 0);

    // acquire takes a counter from the semaphore, blocking until one is 
    // available.
    inline void acquire();

    // release returns a counter to the semaphore, possibly unblocking a call
    // to acquire().
    inline void release();

private:
    Semaphore(const Semaphore&) = delete;
    Semaphore& operator = (const Semaphore&) = delete;

    std::mutex mMutex;
    std::condition_variable mSignal;
    int mCount;
};

Semaphore::Semaphore(unsigned int count /* = 0 */) : mCount(count) {}

inline void Semaphore::acquire() {
    std::unique_lock<std::mutex> lock(mMutex);
    mSignal.wait(lock, [this]{ return this->mCount > 0; });
    --mCount;
}

inline void Semaphore::release() {
    std::unique_lock<std::mutex> lock(mMutex);
    ++mCount;
    mSignal.notify_one();
}

}  // namespace core

#endif  // CORE_SEMAPHORE_H
