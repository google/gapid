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

#ifndef GAPII_SPY_BASE_H
#define GAPII_SPY_BASE_H

#include "abort_exception.h"
#include "call_observer.h"
#include "pack_encoder.h"
#include "slice.h"

#include "core/cc/assert.h"
#include "core/cc/encoder.h"
#include "core/cc/id.h"
#include "core/cc/interval_list.h"
#include "core/cc/null_encoder.h"
#include "core/cc/scratch_allocator.h"
#include "core/cc/thread_local.h"
#include "core/cc/vector.h"

#include "core/os/device/deviceinfo/cc/query.h"

#include "gapis/capture/capture.pb.h"

#if (TARGET_OS == GAPID_OS_LINUX) || (TARGET_OS == GAPID_OS_ANDROID)
#include "core/memory_tracker/cc/memory_tracker.h"
#endif // TARGET_OS

#include <stdint.h>

#include <atomic>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>

namespace gapii {
const uint8_t kAllAPIs = 0xFF;

class SpyBase {
public:
    SpyBase();

    void init(CallObserver* observer);

    // Set whether to observe the application pool. If true, the default,
    // then reads and writes to the application pools are observed, but
    // writes do not change the memory contents. If false, then
    // no-observations are made and writes change the application memory.
    inline void setObserveApplicationPool(bool observeApplicationPool);

    // Encode and write data blob if we have not already sent it.
    // Returns the index of the resource which can be used to reference it.
    int64_t sendResource(uint8_t api, const void* data, size_t size);

    // writeHeader encodes a header with current tracing device and
    // ABI info then return true, if the encoder is ready. Otherwise returns false.
    bool writeHeader();

    // returns new unique pool ID.
    inline uint32_t getPoolID() { return mNextPoolID.fetch_add(1); }

    // Returns the transimission encoder.
    // TODO(qining): To support multithreaded uses, mutex is required to manage
    // the access to this encoder.
    std::shared_ptr<gapii::PackEncoder> getEncoder(uint8_t api) {
        return should_trace(api) ? mEncoder : mNullEncoder;
    }

    // Returns true if we should observe application pool.
    bool shouldObserveApplicationPool() { return mObserveApplicationPool; }

    // Returns true if this spy is suspended, i.e. We should not actually
    // be sending any data across to the server yet.
    bool is_suspended() const { return mIsSuspended; }

    void set_suspended(bool suspended) { mIsSuspended = suspended; }

    bool should_trace(uint8_t api) {
        return !is_suspended() && (api == kAllAPIs || (mWatchedApis & (1 << api)) != 0);
    }
    void set_valid_apis(uint32_t apis) { mWatchedApis = apis; }

    void set_observing(bool observing) { mIsObserving = observing; }
    bool is_observing() const { return mIsObserving; }

    // TODO(awoloszyn) We can remove this once we have switched over our
    // mid-execution over to pass across the serialized state.
    bool is_recording_state() const { return mIsRecordingState; }
    void set_recording_state(bool recording) { mIsRecordingState = recording; }
protected:
    static const size_t kMaxExtras = 16; // Per atom

    // lock begins the interception of a single command. It must be called
    // before invoking any command on the spy. Blocks if any other thread
    // is has called lock and not yet called unlock.
    void lock(CallObserver* observer);

    // unlock must be called after invoking any command.
    // resets the buffers reused between atoms.
    void unlock();

    // make constructs and returns a Slice backed by a new pool.
    template<typename T>
    inline Slice<T> make(uint64_t count);

    // slice returns a slice wrapping the application-pool pointer src, starting at elements s
    // ending at one element before e.
    template<typename T>
    inline Slice<T> slice(T* src, uint64_t s, uint64_t e) const;

    // slice returns a slice wrapping the application-pool pointer src, starting at s bytes
    // from src and ending at one byte before e.
    inline Slice<uint8_t> slice(void* src, uint64_t s, uint64_t e) const;

    // slice returns a Slice<char>, backed by a new pool, holding a copy of the string src.
    // src is observed as a read operation.
    inline Slice<char> slice(const std::string& src);

    // slice returns a sub-slice of src, starting at elements s and ending at one element before e.
    template<typename T>
    inline Slice<T> slice(const Slice<T>& src, uint64_t s, uint64_t e) const;

    // abort signals that the atom should stop execution immediately.
    void abort();

    // onPostDrawCall is after any command annotated with @draw_call
    inline virtual void onPostDrawCall(CallObserver*, uint8_t) {}

    // onPreStartOfFrame is before any command annotated with @frame_start
    inline virtual void onPreStartOfFrame(CallObserver*, uint8_t) {}

    // onPostStrartOfFrame is after any command annotated with @frame_start
    inline virtual void onPostStartOfFrame() {}

    // onPreEndOfFrame is before any command annotated with @frame_end
    inline virtual void onPreEndOfFrame(CallObserver*, uint8_t) {}

    // onPostEndOfFrame is after any command annotated with @frame_end
    inline virtual void onPostEndOfFrame() {}
    // onPostFence is called immediately after the driver call.
    inline virtual void onPostFence(CallObserver* observer) {}

    // The output stream encoder.
    PackEncoder::SPtr mEncoder;

    // setter of the tracing device info.
    void set_device_instance(device::Instance* inst) {
      mDeviceInstance.reset(inst);
    }
    // getters of the tracing device info.
    device::Instance* device_instance() const { return mDeviceInstance.get(); }

    // setter of the tracing ABI info.
    void set_current_abi(device::ABI* abi) { mCurrentABI.reset(abi); }
    // getter of the tracing ABI info.
    device::ABI* current_abi() const { return mCurrentABI.get(); }

#if COHERENT_TRACKING_ENABLED
    track_memory::MemoryTracker mMemoryTracker;
#endif // TARGET_OS

private:
    template <class T> bool shouldObserve(const Slice<T>& slice) const;

    // The stream encoder that does nothing.
    PackEncoder::SPtr mNullEncoder;

    // The information about the current tracing device and ABI.
    std::unique_ptr<device::Instance> mDeviceInstance;
    std::unique_ptr<device::ABI> mCurrentABI;

    // The list of resources that have already been encoded and sent.
    std::unordered_map<core::Id, int64_t> mResources;
    std::mutex mResourcesMutex;

    // The mutex that should be locked for the duration of each of the intercepted commands.
    std::recursive_mutex mMutex;

    // True if we should observe the application pool.
    bool mObserveApplicationPool;

    // Holds the next free pool ID.
    std::atomic<int> mNextPoolID;

    // True if we should not be currently tracing, false if we should be tracing.
    bool mIsSuspended;

    // This is the list of all Apis that are considered for tracing. This is a
    // bit set of apis where bit (1 << api) is set if a particular api
    // should be traced.
    uint32_t mWatchedApis;

    // This is true if we may be observing frame-buffers during the trace.
    // For some API's this will require that we modify some of the
    // image creation parameters
    bool mIsObserving;

    // This is true when all commands are used to record state. This means
    // the commands should still be recorded, but the underlying functions
    // should not be called.
    bool mIsRecordingState;
};

// finds a key in the map and returns the value. If no value is present
// returns the zero for that type.
template<typename Map>
const typename Map::mapped_type findOrZero(const Map& m, const typename Map::key_type& key) {
  auto it = m.find(key);
  if (it == m.end()) {
    return typename Map::mapped_type();
  }
  return it->second;
}

template <class T>
bool SpyBase::shouldObserve(const Slice<T>& slice) const {
    return mObserveApplicationPool && slice.isApplicationPool();
}

inline void SpyBase::setObserveApplicationPool(bool observeApplicationPool) {
    mObserveApplicationPool = observeApplicationPool;
}

template<typename T>
inline Slice<T> SpyBase::make(uint64_t count) {
    auto pool = Pool::create(getPoolID(), count * sizeof(T));
    return Slice<T>(reinterpret_cast<T*>(pool->base()), count, pool);
}

template<typename T>
inline Slice<T> SpyBase::slice(T* src, uint64_t s, uint64_t e) const {
    // TODO: Find the pool containing src
    return Slice<T>(src+s, e-s, std::shared_ptr<Pool>());
}

inline Slice<uint8_t> SpyBase::slice(void* src, uint64_t s, uint64_t e) const {
    return slice(reinterpret_cast<uint8_t*>(src), s, e);
}

inline Slice<char> SpyBase::slice(const std::string& src) {
    Slice<char> dst = make<char>(src.length());
    for (uint64_t i = 0; i < src.length(); i++) {
        dst[i] = src[i];
    }
    return dst;
}

template<typename T>
inline Slice<T> SpyBase::slice(const Slice<T>& src, uint64_t s, uint64_t e) const {
    return src(s, e);
}

}  // namespace gapii

#endif // GAPII_SPY_BASE_H
