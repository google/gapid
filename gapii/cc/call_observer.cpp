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
#include "spy_base.h"

#include "core/cc/thread.h"

#include "gapis/memory/memory_pb/memory.pb.h"

#include <tuple>

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
CallObserver::CallObserver(SpyBase* spy, CallObserver* parent, uint8_t api)
    : mSpy(spy),
      mParent(parent),
      mSeenReferences{{ nullptr, 0 }},
      mCurrentCommandName(nullptr),
      mObserveApplicationPool(spy->shouldObserveApplicationPool()),
      mError(0 /*GL_NO_ERROR*/),
      mApi(api),
      mCurrentThread(core::Thread::current().id()) {

    // context_t initialization.
    this->context_t::id = 0;
    this->context_t::location = 0;
    this->context_t::next_pool_id = &spy->next_pool_id();
    this->context_t::globals = nullptr;
    this->context_t::arena = reinterpret_cast<arena_t*>(spy->arena());

    mEncoderStack.push((parent == nullptr) ?
            mSpy->getEncoder(mApi) : parent->encoder());
    mPendingObservations.setMergeThreshold(MEMORY_MERGE_THRESHOLD);
}

// Releases the observation data memory at the end.
CallObserver::~CallObserver() {}

core::Arena* CallObserver::arena() const {
    return mSpy->arena();
}

void CallObserver::read(const void* base, uint64_t size) {
    if (!mSpy->should_trace(mApi)) return;
    if (size > 0) {
        uintptr_t start = reinterpret_cast<uintptr_t>(base);
        uintptr_t end = start + static_cast<uintptr_t>(size);
        mPendingObservations.merge(Interval<uintptr_t>{start, end});
    }
}

void CallObserver::write(const void* base, uint64_t size) {
    if (!mSpy->should_trace(mApi)) return;
    if (size > 0) {
        uintptr_t start = reinterpret_cast<uintptr_t>(base);
        uintptr_t end = start + static_cast<uintptr_t>(size);
        mPendingObservations.merge(Interval<uintptr_t>{start, end});
    }
}

void CallObserver::observePending() {
    if (!mSpy->should_trace(mApi)) {
        return;
    }
    for (auto p : mPendingObservations) {
        uint8_t* data = reinterpret_cast<uint8_t*>(p.start());
        uint64_t size = p.end() - p.start();
        auto resIndex = mSpy->sendResource(mApi, data, size);
        auto observation = new memory::Observation();
        observation->set_base(p.start());
        observation->set_size(size);
        observation->set_resindex(resIndex);
        encodeAndDelete(observation);
    }
    mPendingObservations.clear();
}

void CallObserver::enter(const ::google::protobuf::Message* cmd) {
    mEncoderStack.push(encoder()->group(cmd));
}

void CallObserver::encode(const ::google::protobuf::Message* cmd) {
    encoder()->object(cmd);
}

void CallObserver::exit() {
    mEncoderStack.pop();
}

void CallObserver::encodeAndDelete(::google::protobuf::Message* cmd) {
    encoder()->object(cmd);
    delete cmd;
}

gapil::String CallObserver::string(const char* str) {
    if (str == nullptr) {
        return gapil::String();
    }
    for (uint64_t i = 0;; i++) {
        if (str[i] == 0) {
            read(str, i + 1);
            return gapil::String(mSpy->arena(), str, str + i);
        }
    }
}

gapil::String CallObserver::string(const gapil::Slice<char>& slice) {
    read(slice);
    return gapil::String(mSpy->arena(), slice.begin(), slice.end());
}

}  // namespace gapii
