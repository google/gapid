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

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

using core::Interval;

namespace gapii {

SpyBase::SpyBase()
    : mObserveApplicationPool(true)
    , mNullEncoder(PackEncoder::noop())
    , mResources{{core::Id{{0}}, 0}}
    , mWatchedApis(0xFFFFFFFF)
#if COHERENT_TRACKING_ENABLED
    , mMemoryTracker()
#endif // TARGET_OS
    , mIsRecordingState(false)
{
}

void SpyBase::init(CallObserver* observer) {
    mObserveApplicationPool = true;
    mIsSuspended = false;
}

void SpyBase::lock(CallObserver* observer) {
    mMutex.lock();
}

void SpyBase::unlock() {
    mMutex.unlock();
}

void SpyBase::abort() {
    GAPID_DEBUG("Command aborted");
    throw AbortException();
}

int64_t SpyBase::sendResource(uint8_t api, const void* data, size_t size) {
    auto hash = core::Id::Hash(data, size);

    // Fast-path if resource with the same hash was already send.
    {
        std::lock_guard<std::mutex> lock(mResourcesMutex);
        auto it = mResources.find(hash);
        if (it != mResources.end()) {
            return it->second;
        }
    }

    // Slow-path if we need to encode and send the resource.
    capture::Resource resource;
    resource.set_data(data, size);
    std::lock_guard<std::mutex> lock(mResourcesMutex);
    auto res = mResources.emplace(hash, mResources.size());
    if (res.second) {
        // Keep the resource mutex during send to ensure other thread
        // can not read the index and reference it before we send it.
        getEncoder(api)->object(&resource);
    }

    return res.first->second;
}

}  // namespace gapii
