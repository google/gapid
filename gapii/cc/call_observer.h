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

#ifndef GAPII_CALL_OBSERVER_H
#define GAPII_CALL_OBSERVER_H

#include "gapii/cc/abort_exception.h"
#include "gapii/cc/gles_types.h"
#include "gapii/cc/pack_encoder.h"
#include "gapii/cc/slice.h"

#include "gapis/memory/memory_pb/memory.pb.h"

#include "core/cc/interval_list.h"
#include "core/cc/scratch_allocator.h"
#include "core/cc/vector.h"

#include <google/protobuf/arena.h>

#include <stack>

namespace gapii {

typedef uint32_t GLenum_Error;

class SpyBase;

// CallObserver collects observation data in API function calls. It is supposed
// to be created at the beginning of each intercepted API function call and
// deleted at the end.
class CallObserver {
public:
    typedef memory_pb::Observation Observation;

    CallObserver(SpyBase* spy_p, CallObserver* parent, uint8_t api);

    ~CallObserver();

    inline CallObserver* getParent() { return mParent; }

    // setCurrentCommandName sets the name of the current command that is being
    // observed by this observer. The storage of cmd_name must remain valid for
    // the lifetime of this observer object, ideally it should be static
    // string.
    void setCurrentCommandName(const char* cmd_name) {
        mCurrentCommandName = cmd_name;
    }

    const char* getCurrentCommandName() { return mCurrentCommandName; }

    // Get or set the GL error code for this call.
    GLenum_Error getError() { return mError; }
    void setError(GLenum_Error err) { mError = err; }

    // getScratch returns a scratch allocator which holds the temporary memory
    // assigned to this observer. The memory assigned to this allocator will be
    // released when CallObserver is destructed.
    // TODO(qining): Implementation for a real thread-local allocator is
    // required.
    core::DefaultScratchAllocator* getScratch() { return &mScratch; }

    // getCurrentThread returns the current thread identifier.
    uint64_t getCurrentThread() { return mCurrentThread; }

    // read is called to make a read memory observation of size bytes, starting
    // at base. It only records the range of the read memory, the actual
    // copying of the data is deferred until the data is to be sent.
    void read(const void* base, uint64_t size);

    // write is called to make a write memory observation of size bytes,
    // starting at base. It only records the range of the write memory, the
    // actual copying of the data is deferred until the data is to be sent.
    void write(const void* base, uint64_t size);

    // read records the memory range for the given slice as a read operation.
    // The actual copying of the data is deferred until the data is to be sent.
    template <typename T>
    inline void read(const Slice<T>& slice);

    // read records and returns the i'th element from the slice src. The actual
    // copying of the data is deferred until the data is to be sent.
    template <typename T>
    inline T read(const Slice<T>& src, uint64_t i);

    // write records the memory for the given slice as a write operation. The
    // actual copying of the data is deferred until the data is to be sent.
    template <typename T>
    inline void write(const Slice<T>& slice);

    // write records a value to i'th element in the slice dst. The actual
    // copying of the data is deferred until the data is to be sent.
    template <typename T>
    inline void write(const Slice<T>& dst, uint64_t i, const T& value);

    // copy copies N elements from src to dst, where N is the smaller of
    // src.count() and dst.count().
    // copy observes the sub-slice of src as a read operation.  The sub-slice
    // of dst is returned so that the write observation can be made after the
    // call to the imported function.
    template <typename T>
    inline Slice<T> copy(const Slice<T>& dst, const Slice<T>& src);

    // clone observes src as a read operation and returns a copy of src in a
    // new Pool.
    template <typename T>
    inline Slice<T> clone(const Slice<T>& src);

    // string returns a std::string from the null-terminated string str.
    // str is observed as a read operation.
    inline std::string string(const char* str);

    // string returns a std::string from the Slice<char> slice.
    // slice is observed as a read operation.
    inline std::string string(const Slice<char>& slice);

    // encoder returns the PackEncoder currently in use.
    inline PackEncoder::SPtr encoder();

    // enter calls toProto() on obj, then passes the proto to enterAndDelete.
    template <typename T>
    inline void enter(const T& obj);

    // encode calls toProto() on obj, then passes the proto to encodeAndDelete.
    template <typename T>
    inline void encode(const T& obj);

    // enter encodes and deletes cmd as a group. All protobuf messages will be
    // encoded to this group until exit() is called.
    void enterAndDelete(::google::protobuf::Message* cmd);

    // exit returns encoding to the group bound before calling enter().
    void exit();

    // encodeAndDelete encodes the proto message to the PackEncoder and then
    // deletes the message.
    void encodeAndDelete(::google::protobuf::Message* cmd);

    // observePending observes and encodes all the pending memory observations.
    // The list of pending memory observations is cleared on returning.
    void observePending();

private:
    // shouldObserve returns true if the given slice is located in application
    // pool and we are supposed to observe application pool.
    template <class T>
    bool shouldObserve(const Slice<T>& slice) const {
        return mObserveApplicationPool && slice.isApplicationPool();
    }

    // Make a slice on a new Pool.
    template <typename T>
    inline Slice<T> make(uint64_t count) const;

    // A pointer to the spy instance.
    SpyBase* mSpy;

    // A pointer to the parent CallObserver.
    CallObserver* mParent;

    // The encoder stack.
    std::stack<PackEncoder::SPtr> mEncoderStack;

    // A pointer to the static array that contains the current command name.
    const char* mCurrentCommandName;

    // True if we should observe the application poool.
    bool mObserveApplicationPool;

    // A pre-allocated memory allocator to store observation data.
    core::DefaultScratchAllocator mScratch;

    // The protobuf Arena to use for proto allocations.
    google::protobuf::Arena mArena;

    // The list of pending reads or writes observations that are yet to be made.
    core::IntervalList<uintptr_t> mPendingObservations;

    // Record GL error which was raised during this call.
    GLenum_Error mError;

    // The current API that this call-observer is observing.
    uint8_t mApi;

    // The current thread id.
    uint64_t mCurrentThread;
};

template <typename T>
inline void CallObserver::read(const Slice<T>& slice) {
    if (shouldObserve(slice)) {
        read(slice.begin(), slice.count() * sizeof(T));
    }
}

template <typename T>
inline T CallObserver::read(const Slice<T>& src, uint64_t index) {
    T& elem = src[index];
    if (shouldObserve(src)) {
        read(&elem, sizeof(T));
    }
    return elem;
}

template <typename T>
inline void CallObserver::write(const Slice<T>& slice) {
    if (shouldObserve(slice)) {
        write(slice.begin(), slice.count() * sizeof(T));
    }
}

template <typename T>
inline void CallObserver::write(const Slice<T>& dst, uint64_t index,
                                const T& value) {
    if (!shouldObserve(
            dst)) {  // The spy must not mutate data in the application pool.
        dst[index] = value;
    } else {
        write(&dst[index], sizeof(T));
    }
}

template <typename T>
inline Slice<T> CallObserver::copy(const Slice<T>& dst, const Slice<T>& src) {
    read(src);
    if (!shouldObserve(
            dst)) {  // The spy must not mutate data in the application pool.
        uint64_t c = (src.count() < dst.count()) ? src.count() : dst.count();
        src.copy(dst, 0, c, 0);
    }
    return dst;
}

template <typename T>
inline Slice<T> CallObserver::clone(const Slice<T>& src) {
    Slice<T> dst = make<T>(src.count());
    // Make sure that we actually fill the data the first time.
    // If we use ::copy(), then the copy will only happen if
    // the observer is active.
    read(src);
    src.copy(dst, 0, src.count(), 0);
    return dst;
}

template <typename T>
inline Slice<T> CallObserver::make(uint64_t count) const {
    auto pool = Pool::create(count * sizeof(T));
    return Slice<T>(reinterpret_cast<T*>(pool->base()), count, pool);
}

inline std::string CallObserver::string(const char* str) {
    GAPID_ASSERT(str != nullptr);
    for (uint64_t i = 0;; i++) {
        if (str[i] == 0) {
            read(str, i + 1);
            return std::string(str, str + i);
        }
    }
}

inline std::string CallObserver::string(const Slice<char>& slice) {
    read(slice);
    return std::string(slice.begin(), slice.end());
}

inline PackEncoder::SPtr CallObserver::encoder() {
    return mEncoderStack.top();
}

template <typename T>
inline void CallObserver::enter(const T& obj) {
    enterAndDelete(obj.toProto());
}

template <typename T>
inline void CallObserver::encode(const T& obj) {
    encodeAndDelete(obj.toProto());
}

}  // namespace gapii

#endif  // GAPII_CALL_OBSERVER_H
