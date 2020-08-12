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

#include "core/cc/log.h"
#include "core/cc/timer.h"

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

// CurrentCaptureVersion is incremented on breaking changes to the capture
// format. NB: Also update equally named field in gapis/capture/grahics.go
static const int CurrentCaptureVersion = 3;

using core::Interval;

namespace gapii {

SpyBase::SpyBase()
    :
#if COHERENT_TRACKING_ENABLED
      mMemoryTracker(),
#endif  // TARGET_OS
      mDisableCoherentMemoryTracker(false),
      mHideUnknownExtensions(false),
      mNextPoolId(1),  // Start at 1 as 0 is reserved for application pool
      mNullEncoder(PackEncoder::noop()),
      mDeviceInstance(nullptr),
      mCurrentABI(nullptr),
      mResources{{core::Id{{0}}, 0}},
      mObserveApplicationPool(true),
      mWatchedApis(0xFFFFFFFF),
      mIsRecordingState(false),
      mRecordTimestamps(false) {
}

void SpyBase::init(CallObserver* observer) {
  mObserveApplicationPool = true;
  mIsSuspended = false;
}

void SpyBase::lock() { mSpinLock.Lock(); }

void SpyBase::unlock() { mSpinLock.Unlock(); }

void SpyBase::abort() {
  GAPID_DEBUG("Command aborted");
  throw AbortException();
}

int64_t SpyBase::sendResource(uint8_t api, const void* data, size_t size) {
  GAPID_ASSERT(should_trace(api));
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
  int64_t index = res.first->second;
  if (res.second) {  // Inserted/new.
    // Keep the resource mutex during send to ensure other thread
    // can not read the index and reference it before we send it.
    resource.set_index(index);
    getEncoder(api)->object(&resource);
  }

  return index;
}

bool SpyBase::writeHeader() {
  capture::Header file_header;
  file_header.set_version(CurrentCaptureVersion);
  if (mDeviceInstance != nullptr) {
    device::Instance* t = new device::Instance(*device_instance());
    file_header.set_allocated_device(t);
  }
  if (mCurrentABI != nullptr) {
    device::ABI* t = new device::ABI(*current_abi());
    file_header.set_allocated_abi(t);
  }
  file_header.set_start_time(core::GetNanoseconds());
  if (mEncoder != nullptr && mEncoder != mNullEncoder) {
    mEncoder->object(&file_header);
    return true;
  }
  return false;
}

}  // namespace gapii
