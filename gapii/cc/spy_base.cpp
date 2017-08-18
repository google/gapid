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

#include "spy_base.h"
#include "to_proto.h"

#include "core/cc/log.h"
#include "core/cc/thread.h"

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

using core::Interval;

namespace gapii {

SpyBase::SpyBase()
    : mObserveApplicationPool(true)
    , mNullEncoder(PackEncoder::noop())
    , mWatchedApis(0xFFFFFFFF)
#ifdef COHERENT_TRACKING_ENABLED
    , mMemoryTracker()
#endif // TARGET_OS
{
    mCurrentThread = core::Thread::current().id();
}

void SpyBase::init(CallObserver* observer) {
    auto threadID = core::Thread::current().id();
    mObserveApplicationPool = true;
    mCurrentThread = threadID;
    mIsSuspended = false;
    onThreadSwitched(observer, threadID);
}

void SpyBase::lock(CallObserver* observer) {
    mMutex.lock();
    auto threadID = core::Thread::current().id();
    if (threadID != mCurrentThread) {
        GAPID_DEBUG("Changing threads: %" PRIu64 "-> %" PRIu64, mCurrentThread, threadID);
        mCurrentThread = threadID;
        onThreadSwitched(observer, threadID);
    }
}

void SpyBase::unlock() {
    mMutex.unlock();
}

void SpyBase::abort() {
    throw AbortException();
}

}  // namespace gapii
