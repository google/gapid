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

#include "call_observer.h"

#include <tuple>

#include "spy_base.h"

using core::Interval;

namespace {

// Minimum byte gap between memory observations before globbing together.
const size_t MEMORY_MERGE_THRESHOLD = 256;

// Size of the temporary heap buffer to use when the scratch stack buffer is
// filled.
const size_t SCRATCH_BUFFER_SIZE = 512*1024;

// Buffer creating function for scratch allocator.
std::tuple<uint8_t*, size_t> createBuffer(size_t request_size,
                                                 size_t min_buffer_size) {
    size_t size =
        request_size > min_buffer_size ? request_size : min_buffer_size;
    return std::make_tuple(new uint8_t[size], size);
}

// Buffer releasing function for scratch allocator.
void releaseBuffer(uint8_t* buffer) { delete[] buffer; }

}  // anonymous namespace

namespace gapii {
// Creates a CallObserver with a given spy and applies the memory space for
// observation data from the spy instance.
CallObserver::CallObserver(SpyBase* spy_p, uint8_t api)
    : mSpyPtr(spy_p),
      mCurrentCommandName(nullptr),
      mObserveApplicationPool(spy_p->shouldObserveApplicationPool()),
      mScratch(
          [](size_t size) { return createBuffer(size, SCRATCH_BUFFER_SIZE); },
          [](uint8_t* buffer) { return releaseBuffer(buffer); }),
      mError(GLenum::GL_NO_ERROR),
      mApi(api) {
    mPendingObservations.setMergeThreshold(MEMORY_MERGE_THRESHOLD);
}

// Releases the observation data memory at the end.
CallObserver::~CallObserver() {}

void CallObserver::read(const void* base, uint64_t size) {
    if (!mSpyPtr->should_trace(mApi)) return;
    if (size > 0) {
        uintptr_t start = reinterpret_cast<uintptr_t>(base);
        uintptr_t end = start + static_cast<uintptr_t>(size);
        mPendingObservations.merge(Interval<uintptr_t>{start, end});
    }
}

void CallObserver::write(const void* base, uint64_t size) {
    if (!mSpyPtr->should_trace(mApi)) return;
    if (size > 0) {
        uintptr_t start = reinterpret_cast<uintptr_t>(base);
        uintptr_t end = start + static_cast<uintptr_t>(size);
        mPendingObservations.merge(Interval<uintptr_t>{start, end});
    }
}

void CallObserver::observePending() {
    if (!mSpyPtr->should_trace(mApi)) {
        return;
    }
    for (auto p : mPendingObservations) {
        core::Vector<uint8_t> data(reinterpret_cast<uint8_t*>(p.start()),
                                    p.end() - p.start());
        core::Id id = core::Id::Hash(data.data(), data.count());
        if (mSpyPtr->getResources().count(id) == 0) {
            capture::Resource resource;
            resource.set_id(reinterpret_cast<const char*>(id.data), sizeof(id.data));
            resource.set_data(data.data(), data.count());
            mSpyPtr->getEncoder(mApi)->message(&resource);
            mSpyPtr->getResources().emplace(id);
        }
        auto observation = new memory_pb::Observation();
        observation->set_base(p.start());
        observation->set_size(data.count());
        observation->set_id(reinterpret_cast<const char*>(id.data), sizeof(id.data));
        encodeAndDelete(observation);
    }
    mPendingObservations.clear();
}

void CallObserver::encodeAndDelete(::google::protobuf::Message* cmd) {
    auto encoder = mSpyPtr->getEncoder(mApi);
    encoder->message(cmd);
    delete cmd;
}

bool CallObserver::isActive() const {
    return mSpyPtr->should_trace(mApi);
}

}  // namespace gapii
