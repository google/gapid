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

#ifndef GAPIR_STACK_H
#define GAPIR_STACK_H

#include "base_type.h"
#include "memory_manager.h"

#include "core/cc/log.h"
#include "core/cc/static_array.h"

#include <stdint.h>
#include <string.h>

#include <assert.h>
#include <type_traits>
#include <typeindex>

namespace gapir {

class Stack {
 public:
  class VolatilePointer {
   public:
    explicit VolatilePointer(uint32_t offset) : mOffset(offset) {}
    bool operator==(const VolatilePointer& rhs) const {
      return mOffset == rhs.mOffset;
    }
    uint32_t getOffset() const { return mOffset; }

   private:
    uint32_t mOffset;
  };

  class ConstantPointer {
   public:
    explicit ConstantPointer(uint32_t offset) : mOffset(offset) {}
    bool operator==(const ConstantPointer& rhs) const {
      return mOffset == rhs.mOffset;
    }
    uint32_t getOffset() const { return mOffset; }

   private:
    uint32_t mOffset;
  };

  Stack(uint32_t size, const MemoryManager* memoryManager)
      : mNumEntries(0),
        mMaxEntries(size),
        mStack(new Entry[size]),
        mMemoryManager(memoryManager) {}

  ~Stack() { delete[] mStack; }

  template <typename T>
  T pop(typename std::enable_if<std::is_pointer<T>::value>::type* = 0) {
    GAPID_ASSERT(mNumEntries > 0);

    if (isVolatilePointerAtTop()) {
      VolatilePointer volPtr = mStack[--mNumEntries].get<VolatilePointer>();
      return (T)mMemoryManager->volatileToAbsolute(volPtr.getOffset());
    } else if (typeAtTop() == typeid(ConstantPointer)) {
      ConstantPointer constPtr = mStack[--mNumEntries].get<ConstantPointer>();
      return (T)mMemoryManager->constantToAbsolute(constPtr.getOffset());
    }

    return (T)mStack[--mNumEntries].get<void*>();
  }

  template <typename T>
  T pop(typename std::enable_if<!std::is_pointer<T>::value>::type* = 0) {
    GAPID_ASSERT(mNumEntries > 0);
    return mStack[--mNumEntries].get<T>();
  }

  template <typename T, int N>
  core::StaticArray<T, N> pop() {
    T* ptr = pop<T*>();
    core::StaticArray<T, N> ret;
    for (int i = 0; i < N; i++) {
      ret[i] = ptr[i];
    }
    return ret;
  }

  template <typename T>
  void push(T value,
            typename std::enable_if<std::is_pointer<T>::value>::type* = 0) {
    GAPID_ASSERT(mNumEntries < mMaxEntries);
    mStack[mNumEntries++].set((void*)value);
  }

  template <typename T>
  void push(T value,
            typename std::enable_if<!std::is_pointer<T>::value>::type* = 0) {
    GAPID_ASSERT(mNumEntries < mMaxEntries);
    mStack[mNumEntries++].set(value);
  }

  void discard(uint32_t count) {
    GAPID_ASSERT(mNumEntries >= count);
    mNumEntries -= count;
  }

  void clone(uint32_t n) {
    GAPID_ASSERT(mNumEntries > n);
    mStack[mNumEntries] = mStack[mNumEntries - 1 - n];
    mNumEntries++;
  }

  void popTo(void* address) {
    GAPID_ASSERT(mNumEntries > 0);
    if (isVolatilePointerAtTop()) {
      void* val = pop<void*>();
      *((void**)address) = val;
      return;
    }
    mStack[--mNumEntries].copyTo(address);
  }

  bool isEmpty() const { return mNumEntries == 0; }

  const std::type_index& typeAtTop() const {
    return mStack[mNumEntries - 1].getType();
  }
  bool isVolatilePointerAtTop() const {
    return mStack[mNumEntries - 1].isVolatilePointer();
  }

 private:
  class Entry {
   public:
    Entry()
        : mDataSize(0), mDataType(typeid(void)), mIsVolatilePointer(false) {}

    template <class T>
    static void assertCanStore() {
      static_assert(
          sizeof(T) <= maxEntrySize,
          "You need to bump the value of gapir::Stack::Entry::maxEntrySize");
      static_assert(entryAlignment % alignof(T) == 0,
                    "You need to change the value of "
                    "gapir::Stack::Entry::entryAlignment");
    }

    template <class T>
    void set(const T& val);

    template <class T>
    const T& get() {
      assertCanStore<T>();
      GAPID_DEBUG_ASSERT(sizeof(T) == mDataSize && mDataType == typeid(T));
      return *((T*)&mData);
    }

    const std::type_index& getType() const { return mDataType; }
    bool isVolatilePointer() const { return mIsVolatilePointer; }

    void copyTo(void* address) { memcpy(address, (void*)&mData, mDataSize); }

    Entry& operator=(const Entry& rhs) {
      this->mDataSize = rhs.mDataSize;
      this->mDataType = rhs.mDataType;
      memcpy((void*)&mData, (void*)&rhs.mData, maxEntrySize);
      return *this;
    }

   private:
    constexpr static const size_t maxEntrySize = 8;
    constexpr static const size_t entryAlignment = maxEntrySize;
    typename std::aligned_storage<maxEntrySize, entryAlignment>::type mData;

    size_t mDataSize;
    std::type_index mDataType;
    bool mIsVolatilePointer;
  };

  uint32_t mNumEntries;
  uint32_t mMaxEntries;
  Entry* mStack;

  const MemoryManager* mMemoryManager;
};

template <class T>
void Stack::Entry::set(const T& val) {
  assertCanStore<T>();
  *((T*)&mData) = val;
  mDataSize = sizeof(T);
  mDataType = std::type_index(typeid(T));
  mIsVolatilePointer = false;
}
template <>
void Stack::Entry::set<Stack::VolatilePointer>(
    const Stack::VolatilePointer& val);

}  // namespace gapir

#endif  // GAPIR_STACK_H
