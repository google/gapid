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
    , mPassthrough(false)
    , mCommandStartEndCounter(0)
    , mExpectedNextCommandStartCounterValue(0)
    , mNullEncoder(PackEncoder::noop())
#ifdef COHERENT_TRACKING_ENABLED
    , mMemoryTracker()
#endif // TARGET_OS
{
    mCurrentThread = core::Thread::current().id();
}

void SpyBase::init(CallObserver* observer, PackEncoder::SPtr encoder) {
    auto threadID = core::Thread::current().id();
    mEncoder = encoder;
    mObserveApplicationPool = true;
    mPassthrough = false;
    mAbortHandler = nullptr;
    mCurrentThread = threadID;
    mIsSuspended = false;
    onThreadSwitched(observer, threadID);
}

bool SpyBase::try_to_enter() {
  if (mPassthrough || mReentrantFlag.get() != 0) {
    return false;
  }
  mReentrantFlag.set(1u);
  return true;
}

void SpyBase::exit() {
  mReentrantFlag.set(0);
}

void SpyBase::lock(CallObserver* observer, const char* name) {
    mMutex.lock();
    observer->setCurrentCommandName(name);

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
    throw AbortException(AbortException::NORMAL, "");
}

void SpyBase::defaultAbortHandler(CallObserver* observer, const AbortException& e) {
    switch (e.category()) {
        case AbortException::NORMAL: {
            GAPID_DEBUG("normal abort handled; no action taken");
            return;
        }
        case AbortException::ASSERT:  // fallthrough
        default: {
            GAPID_WARNING("assert handled; switching to passthrough mode %s", e.message().c_str());
            mPassthrough = true;
        }
    }
    // Record the abort as an extra.
    auto aborted = new atom_pb::Aborted();
    aborted->set_isassert(e.category() == AbortException::ASSERT);
    aborted->set_reason(e.message());
    observer->addExtra(aborted);
}

}  // namespace gapii
