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

#include "abort_exception.h"
#include "pack_encoder.h"

#include "gapil/runtime/cc/encoder.h"
#include "gapil/runtime/cc/runtime.h"
#include "gapil/runtime/cc/slice.inc"
#include "gapil/runtime/cc/string.h"

#include "core/cc/interval_list.h"
#include "core/cc/vector.h"
#include "core/memory/arena/cc/arena.h"

#include <stack>
#include <type_traits>
#include <unordered_map>

namespace gapii {

typedef uint32_t GLenum_Error;

class SpyBase;

// CallObserver collects observation data in API function calls. It is supposed
// to be created at the beginning of each intercepted API function call and
// deleted at the end.
class CallObserver : public context_t, gapil::Encoder {
 public:
  template <class T>
  using enable_if_encodable = typename std::enable_if<
      std::is_member_function_pointer<decltype(&T::encode)>::value>::type;

  typedef std::function<void(const slice_t*)> OnSliceEncodedCallback;

  CallObserver(SpyBase* spy_p, CallObserver* parent, uint8_t api);

  virtual ~CallObserver();

  inline CallObserver* getParent() { return mParent; }

  // setCurrentCommandName sets the name of the current command that is being
  // observed by this observer. The storage of cmd_name must remain valid for
  // the lifetime of this observer object, ideally it should be static
  // string.
  void setCurrentCommandName(const char* cmd_name) {
    mCurrentCommandName = cmd_name;
  }

  const char* getCurrentCommandName() { return mCurrentCommandName; }

  // getCurrentThread returns the current thread identifier.
  inline uint64_t getCurrentThread() { return mCurrentThread; }

  // Returns the unique reference identifier for the given object address,
  // and true when the address is seen for the first time.
  // Nullptr address is always mapped to identifier 0.
  inline std::pair<uint64_t, bool> reference_id(const void* address) {
    auto it = mSeenReferences.emplace(address, mSeenReferences.size());
    return std::pair<uint64_t, bool>(it.first->second, it.second);
  }

  // on_slice_encoded sets the callback to be invoked when slice_encoded is
  // called.
  inline void on_slice_encoded(OnSliceEncodedCallback f) {
    mOnSliceEncoded = f;
  }

  // encodeType returns a new positive unique reference identifer if
  // the type has not been encoded before in this scope, otherwise it returns
  // the negated ID of the previously encoded type identifier.
  int64_t encodeType(const char* name, uint32_t desc_size,
                     const void* desc) override;

  // encodeObject encodes the object.
  // If is_group is true, a new encoder will be returned for encoding
  // sub-objects. If is_group is false then encodeObject will return null.
  void* encodeObject(uint8_t is_group, uint32_t type, uint32_t data_size,
                     void* data) override;

  // encodeBackref returns a new positive unique reference identifer if
  // object has not been encoded before in this scope, otherwise it returns the
  // negated ID of the previously encoded object identifier.
  int64_t encodeBackref(const void* object) override;

  // sliceEncoded is called whenever a slice is encoded. This callback
  // can be used to write the slice's data into the encoder's stream.
  inline void sliceEncoded(const void*) override {
    if (mOnSliceEncoded) {
      mOnSliceEncoded(reinterpret_cast<const slice_t*>(slice));
    }
  }

  // arena returns the active memory arena.
  core::Arena* arena() const override;

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
  inline void read(const gapil::Slice<T>& slice);

  // read records and returns the i'th element from the slice src. The actual
  // copying of the data is deferred until the data is to be sent.
  template <typename T>
  inline T read(const gapil::Slice<T>& src, uint64_t i);

  // write records the memory for the given slice as a write operation. The
  // actual copying of the data is deferred until the data is to be sent.
  template <typename T>
  inline void write(const gapil::Slice<T>& slice);

  // write records a value to i'th element in the slice dst. The actual
  // copying of the data is deferred until the data is to be sent.
  template <typename T>
  inline void write(const gapil::Slice<T>& dst, uint64_t i, const T& value);

  // copy copies N elements from src to dst, where N is the smaller of
  // src.count() and dst.count().
  // copy observes the sub-slice of src as a read operation.  The sub-slice
  // of dst is returned so that the write observation can be made after the
  // call to the imported function.
  template <typename T>
  inline gapil::Slice<T> copy(const gapil::Slice<T>& dst,
                              const gapil::Slice<T>& src);

  // clone observes src as a read operation and returns a copy of src in a
  // new Pool.
  template <typename T>
  inline gapil::Slice<T> clone(const gapil::Slice<T>& src);

  // string returns a gapil::String from the null-terminated string str.
  // str is observed as a read operation.
  gapil::String string(const char* str);

  // string returns a gapil::String from the gapil::Slice<char> slice.
  // slice is observed as a read operation.
  gapil::String string(const gapil::Slice<char>& slice);

  // encoder returns the PackEncoder currently in use.
  inline PackEncoder::SPtr encoder();

  // enter encodes cmd as a group. All encodables will be encoded to this
  // group until exit() is called.
  void enter(const ::google::protobuf::Message* cmd);

  // encode encodes cmd the proto message to the PackEncoder.
  void encode_message(const ::google::protobuf::Message* cmd);

  // encodeAndDelete encodes the proto message to the PackEncoder and then
  // deletes the message.
  void encodeAndDelete(::google::protobuf::Message* cmd);

  // enter encodes the encodable to the PackEncoder. All encodables will be
  // encoded to this group until exit() is called.
  template <typename T, typename = enable_if_encodable<T> >
  inline void enter(const T& obj);

  // encode encodes the encodable to the PackEncoder.
  template <typename T, typename = enable_if_encodable<T> >
  inline void encode(const T& obj);

  // resume updates whether the observer should keep on tracing or not. It is
  // meant to be called when observation of a threadsafe command is resumed
  // after the actual driver call, and after re-acquiring the Spy lock.
  void resume();

  // exit returns encoding to the group bound before calling enter().
  void exit();

  // observePending observes and encodes all the pending memory observations.
  // The list of pending memory observations is cleared on returning.
  void observePending();

  // observeTimestamp encodes a timestamp extra in the trace
  void observeTimestamp();

 private:
  // shouldObserve returns true if the given slice is located in application
  // pool and we are supposed to observe application pool.
  template <class T>
  bool shouldObserve(const gapil::Slice<T>& slice) const {
    return mObserveApplicationPool && slice.is_app_pool();
  }

  // Make a slice on a new Pool.
  template <typename T>
  inline gapil::Slice<T> make(uint64_t count);

  // Ends the current trace if requested by client.
  void endTraceIfRequested();

  // A pointer to the spy instance.
  SpyBase* mSpy;

  // A pointer to the parent CallObserver.
  CallObserver* mParent;

  // The encoder stack.
  std::stack<PackEncoder::SPtr> mEncoderStack;

  // A map of object pointer to encoded reference identifier.
  std::unordered_map<const void*, uint64_t> mSeenReferences;

  // A pointer to the static array that contains the current command name.
  const char* mCurrentCommandName;

  // True if we should observe the application poool.
  bool mObserveApplicationPool;

  // The list of pending reads or writes observations that are yet to be made.
  core::IntervalList<uintptr_t> mPendingObservations;

  // The current API that this call-observer is observing.
  uint8_t mApi;

  // Whether or not we should be tracing with this call observer
  bool mShouldTrace;

  // The current thread id.
  uint64_t mCurrentThread;

  // Callback invoked whenever slice_encoded() is called.
  OnSliceEncodedCallback mOnSliceEncoded;
};

template <typename T>
inline void CallObserver::read(const gapil::Slice<T>& slice) {
  if (shouldObserve(slice)) {
    read(slice.begin(), slice.count() * sizeof(T));
  }
}

template <typename T>
inline T CallObserver::read(const gapil::Slice<T>& src, uint64_t index) {
  T& elem = src[index];
  if (shouldObserve(src)) {
    read(&elem, sizeof(T));
  }
  return elem;
}

template <typename T>
inline void CallObserver::write(const gapil::Slice<T>& slice) {
  if (shouldObserve(slice)) {
    write(slice.begin(), slice.count() * sizeof(T));
  }
}

template <typename T>
inline void CallObserver::write(const gapil::Slice<T>& dst, uint64_t index,
                                const T& value) {
  if (!shouldObserve(
          dst)) {  // The spy must not mutate data in the application pool.
    dst[index] = value;
  } else {
    write(&dst[index], sizeof(T));
  }
}

template <typename T>
inline gapil::Slice<T> CallObserver::copy(const gapil::Slice<T>& dst,
                                          const gapil::Slice<T>& src) {
  read(src);
  if (!shouldObserve(
          dst)) {  // The spy must not mutate data in the application pool.
    uint64_t c = (src.count() < dst.count()) ? src.count() : dst.count();
    src.copy(dst, 0, c, 0);
  }
  return dst;
}

template <typename T>
inline gapil::Slice<T> CallObserver::clone(const gapil::Slice<T>& src) {
  auto dst = make<T>(src.count());
  // Make sure that we actually fill the data the first time.
  // If we use ::copy(), then the copy will only happen if
  // the observer is active.
  read(src);
  src.copy(dst, 0, src.count(), 0);
  return dst;
}

template <typename T>
inline gapil::Slice<T> CallObserver::make(uint64_t count) {
  return gapil::Slice<T>::create(this, count);
}

inline PackEncoder::SPtr CallObserver::encoder() { return mEncoderStack.top(); }

template <typename T, typename /* = enable_if_encodable<T> */>
inline void CallObserver::enter(const T& obj) {
  endTraceIfRequested();
  if (!mShouldTrace) {
    return;
  }
  auto group = reinterpret_cast<PackEncoder*>(obj.encode(this, true));
  GAPID_ASSERT_MSG(group != nullptr,
                   "encode() for group did not return sub-encoder");
  mEncoderStack.push(PackEncoder::SPtr(group));
}

template <typename T, typename /* = enable_if_encodable<T> */>
inline void CallObserver::encode(const T& obj) {
  if (!mShouldTrace) {
    return;
  }
  auto group = obj.encode(this, false);
  GAPID_ASSERT_MSG(group == nullptr,
                   "encode() for non-group returned sub-encoder");
}

}  // namespace gapii

#endif  // GAPII_CALL_OBSERVER_H
