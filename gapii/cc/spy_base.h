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
#include "return_handler.h"
#include "slice.h"

#include "core/cc/scratch_allocator.h"
#include "core/cc/encoder.h"
#include "core/cc/null_encoder.h"
#include "core/cc/interval_list.h"
#include "core/cc/mutex.h"
#include "core/cc/thread_local.h"
#include "core/cc/vector.h"

#include "core/cc/coder/memory.h"
#include "core/cc/coder/atom.h"

#if (TARGET_OS == GAPID_OS_LINUX) || (TARGET_OS == GAPID_OS_ANDROID)
#include <core/memory_tracker/cc/memory_tracker.h>
#endif // TARGET_OS

#include <stdint.h>

#include <memory>
#include <string>
#include <unordered_set>

namespace gapii {

class SpyBase {
public:
    SpyBase();

    void init(CallObserver* observer, std::shared_ptr<core::Encoder> encoder);

    // lock begins the interception of a single command. It must be called
    // before invoking any command on the spy. Blocks if any other thread
    // is has called lock and not yet called unlock.
    void lock(CallObserver* observer, const char* name);

    // unlock must be called after invoking any command.
    // resets the buffers reused between atoms.
    void unlock();

    // Set whether to observe the application pool. If true, the default,
    // then reads and writes to the application pools are observed, but
    // writes do not change the memory contents. If false, then
    // no-observations are made and writes change the application memory.
    inline void setObserveApplicationPool(bool observeApplicationPool);
    // Set whether to just pass forward the calls directly from the
    // interceptor to the underlying driver. If false, the default, then
    // API state logic is run and an atom is emitted to the trace stream.
    // If true, the spy is just a trampoline.
    inline void setPassthrough(bool passthrough);
    // Set the handler to use when handle() is called with an abort
    // exception. This allows different treatment for different
    // aborts without requiring API specific knowledge.
    typedef std::function<void(CallObserver* observer, const AbortException&)> AbortHandler;
    inline void setHandler(AbortHandler handler);
    // Returns true if the spy should compute the expected return value and
    // call setExpectedReturn(v). Default is false.
    inline bool shouldComputeExpectedReturn() const;
    // Set the handler to use when setExpectedReturn(val) is called.
    // By default the return value specified is ignored.
    inline void setReturnHandler(std::shared_ptr<ReturnHandler> handler);

    // Returns the set of resources ids.
    // TODO(qining): To support multithreaded uses, mutex is required to manage
    // the access to this set.
    std::unordered_set<core::Id>& getResources() { return mResources; }
    // Returns the transimission encoder.
    // TODO(qining): To support multithreaded uses, mutex is required to manage
    // the access to this encoder.
    std::shared_ptr<core::Encoder> getEncoder() { return is_suspended()?
        mNullEncoder: mEncoder;
    }

    // Returns true if we should observe application pool.
    bool shouldObserveApplicationPool() { return mObserveApplicationPool; }

    // Tries to enter this function. If SpyBase has already been entered before
    // by the same thread, this returns false. e.g.  If the driver calls the
    // function recursively.
    bool try_to_enter();

    // Leaves this function. Only valid to call whenever we have succeeded
    // at a call of try_to_enter.
    void exit();

    // Returns true if this spy is suspended, i.e. We should not actually
    // be sending any data across to the server yet.
    bool is_suspended() const { return mIsSuspended; }

    void set_suspended(bool suspended) { mIsSuspended = suspended; }
protected:
    static const size_t kMaxExtras = 16; // Per atom

    typedef core::coder::atom::Observation Observation;

    typedef std::unordered_set<core::Id> IdSet;
    typedef std::shared_ptr<core::Encoder> EncoderSPtr;

    // onThreadSwitched is invoked by enter() whenever the current thread changes.
    virtual void onThreadSwitched(CallObserver* observer, uint64_t threadID) = 0;

    // make constructs and returns a Slice backed by a new pool.
    template<typename T>
    inline Slice<T> make(uint64_t count) const;

    // slice returns a slice wrapping the application-pool pointer src, starting at elements s
    // ending at one element before e.
    template<typename T>
    inline Slice<T> slice(T* src, uint64_t s, uint64_t e) const;

    // slice returns a slice wrapping the application-pool pointer src, starting at s bytes
    // from src and ending at one byte before e.
    inline Slice<uint8_t> slice(void* src, uint64_t s, uint64_t e) const;

    // slice returns a Slice<char>, backed by a new pool, holding a copy of the string src.
    // src is observed as a read operation.
    inline Slice<char> slice(const std::string& src) const;

    // slice returns a sub-slice of src, starting at elements s and ending at one element before e.
    template<typename T>
    inline Slice<T> slice(const Slice<T>& src, uint64_t s, uint64_t e) const;

    // abort signals that the atom should stop execution immediately.
    void abort();

    // handle is called in the abort exception catch. Used to allow
    // customization of abort handling.
    inline void handleAbort(CallObserver* observer, const AbortException& e);

    // onPostDrawCall is after any command annotated with @DrawCall
    inline virtual void onPostDrawCall() {}

    // onPreEndOfFrame is before any command annotated with @EndOfFrame
    inline virtual void onPreEndOfFrame() {}

    // onPostEndOfFrame is after any command annotated with @EndOfFrame
    inline virtual void onPostEndOfFrame(CallObserver* observer) {}

    // onPostFence is called immediately after the driver call.
    inline virtual void onPostFence(CallObserver* observer) {}

    // Abort handler used when if no other handler has been specified
    void defaultAbortHandler(CallObserver* observer, const AbortException& e);

    // Returns true if the current thread is currently "in" the spy, where
    // "in" is defined as "the time between a true return of try_to_enter and
    // a matching call to exit".
    bool has_entered() {
      return mReentrantFlag.get() != 0;
    }

    // If true the interceptor calls the underlying function directly.
    bool mPassthrough;

    // A counter that is incremented each time a graphics command starts or
    // ends. The first command start gets a value of 0 for its starting command
    // counter value.
    uint64_t mCommandStartEndCounter;

    // The expected counter value for the starting of the next command. This
    // equals the counter value of the last command ending plus one. This value
    // starts at 0 before any atoms have been sent.
    uint64_t mExpectedNextCommandStartCounterValue;

    // Stores the extra if the command aborted.
    core::coder::atom::Aborted* mAborted;

    // Used by the generated code to indicate that the API file compute t as the
    // return for this call. The actual return value comes from the driver.
    template <typename T> void setExpectedReturn(const T& t);

#if (TARGET_OS == GAPID_OS_LINUX) || (TARGET_OS == GAPID_OS_ANDROID)
    TrackMemory::MemoryTracker mMemoryTracker;
#endif // TARGET_OS
private:
    template <class T> bool shouldObserve(const Slice<T>& slice) const;

    // The output stream encoder.
    EncoderSPtr mEncoder;

    std::shared_ptr<core::NullEncoder>  mNullEncoder;

    // The list of observations that have already been encoded.
    IdSet mResources;

    // The current thread ID.
    uint64_t mCurrentThread;

    // The mutex that should be locked for the duration of each of the intercepted commands.
    core::Mutex mMutex;
    // True if we should observe the application pool.
    bool mObserveApplicationPool;

    // If non-null this handler is used instead of defaultAbortHandler.
    AbortHandler mAbortHandler;

    // If non-null this is a class which can accept return values of arbitrary
    // types (with a copy constructor and assignment operator).
    std::shared_ptr<ReturnHandler> mReturnHandler;

    // Initially set to zero for all threads. This is set to a non-zero value
    // for every thread that calls try_to_enter with a true return value,
    // and reset for that thread when the matching exit() function is called.
    core::ThreadLocalValue mReentrantFlag;

    // True if we should not be currently tracing, false if we should be tracing.
    bool mIsSuspended;
};

// finds a key in the map and returns the value. If no value is present
// returns the zero for that type.
template<typename Map>
const typename Map::mapped_type& findOrZero(const Map& m, const typename Map::key_type& key) {
  auto it = m.find(key);
  if (it == m.end()) {
    static auto zero = typename Map::mapped_type();
    return zero;
  }
  return it->second;
}

inline bool SpyBase::shouldComputeExpectedReturn() const {
    return !mObserveApplicationPool && mReturnHandler != nullptr;
}

inline void SpyBase::setReturnHandler(std::shared_ptr<ReturnHandler> handler) {
    mReturnHandler = handler;
}

template <typename T>
void SpyBase::setExpectedReturn(const T& t) {
    spyAssert(shouldComputeExpectedReturn(), "setExpectedReturn called, but shouldComputeExpectedReturn is false");
    mReturnHandler->setReturnValue(t);
}

template <class T>
bool SpyBase::shouldObserve(const Slice<T>& slice) const {
    return mObserveApplicationPool && slice.isApplicationPool();
}

inline void SpyBase::setObserveApplicationPool(bool observeApplicationPool) {
    mObserveApplicationPool = observeApplicationPool;
}

inline void SpyBase::setPassthrough(bool passthrough) {
    mPassthrough = passthrough;
}

inline void SpyBase::setHandler(AbortHandler handler) {
    mAbortHandler = handler;
}

inline void SpyBase::handleAbort(CallObserver* observer, const AbortException& e) {
    if (mAbortHandler == nullptr) {
        defaultAbortHandler(observer, e);
    } else {
        mAbortHandler(observer, e);
    }
}

template<typename T>
inline Slice<T> SpyBase::make(uint64_t count) const {
    auto pool = Pool::create(count * sizeof(T));
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

inline Slice<char> SpyBase::slice(const std::string& src) const {
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
