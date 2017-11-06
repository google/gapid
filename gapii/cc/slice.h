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

#ifndef GAPII_SLICE_H
#define GAPII_SLICE_H

#include "pool.h"
#include "abort_exception.h"

#include "core/cc/assert.h"

#include <algorithm>
#include <memory>
#include <stdint.h>
#include <cstring>

namespace gapii {

template<typename T>
class Slice {
public:
    inline Slice();
    inline Slice(T* base, uint64_t count, const std::shared_ptr<Pool>& pool);

    // Equality operator.
    inline bool operator == (const Slice<T>& other) const;

    // Returns the number of elements in the slice.
    inline uint64_t count() const;

    // Returns the size of the slice in bytes.
    inline uint64_t size() const;

    // Returns true if this is a slice on the application pool (external memory).
    inline bool isApplicationPool() const;

    // Returns true if the slice contains the specified value.
    inline bool contains(const T& value) const;

    // Returns a new subset slice from this slice.
    inline Slice<T> operator()(uint64_t start, uint64_t end) const;

    // Returns a reference to a single element in the slice.
    // Care must be taken to not mutate data in the application pool.
    inline T& operator[](uint64_t index) const;

    // Copies count elements starting at start into the dst Slice starting at
    // dstStart.
    inline void copy(const Slice<T>& dst, uint64_t start, uint64_t count, uint64_t dstStart) const;

    // As casts this slice to a slice of type U.
    // The return slice length will be calculated so that the returned slice length is no longer
    // (in bytes) than this slice.
    template<typename U> inline Slice<U> as() const;

    // Support for range-based for looping
    inline T* begin() const;
    inline T* end() const;

private:
    T* mBase;
    uint64_t mCount;
    std::shared_ptr<Pool> mPool;
};

template<typename T>
inline Slice<T>::Slice() : mBase(nullptr), mCount(0) {}

template<typename T>
inline Slice<T>::Slice(T* base, uint64_t count, const std::shared_ptr<Pool>& pool)
    : mBase(base), mCount(count), mPool(pool) {
    GAPID_ASSERT(mBase != nullptr || count == 0 /* Slice: null pointer */);
}

template<typename T>
inline bool Slice<T>::operator == (const Slice<T>& other) const {
    return mBase == other.mBase && mCount == other.mCount;
}

template<typename T>
inline uint64_t Slice<T>::count() const {
    return mCount;
}

template<typename T>
inline uint64_t Slice<T>::size() const {
    return mCount * sizeof(T);
}

template<typename T>
inline bool Slice<T>::isApplicationPool() const {
    return mPool.get() == nullptr;
}

template<typename T>
inline bool Slice<T>::contains(const T& value) const {
    return std::find(begin(), end(), value) != end();
}

template<typename T>
inline Slice<T> Slice<T>::operator()(uint64_t start, uint64_t end) const {
    GAPID_ASSERT(start <= end /* Slice: start > end */);
    GAPID_ASSERT(end <= mCount /* Slice: index out of bounds */);
    GAPID_ASSERT(mBase != nullptr || (start == end) /* Slice: null pointer */);
    return Slice<T>(mBase+start, end-start, mPool);
}

template<typename T>
inline T& Slice<T>::operator[](uint64_t index) const {
    GAPID_ASSERT(index >= 0 && index < mCount /* Slice: index out of bounds */);
    return mBase[index];
}

template<typename T> template<typename U>
inline Slice<U> Slice<T>::as() const {
    uint64_t count = size() / sizeof(U);
    return Slice<U>(reinterpret_cast<U*>(mBase), count, mPool);
}

template<typename T>
inline T* Slice<T>::begin() const {
    return mBase;
}

template<typename T>
inline T* Slice<T>::end() const {
    return mBase + mCount;
}

template<typename T>
inline void Slice<T>::copy(const Slice<T>& dst, uint64_t start, uint64_t cnt, uint64_t dstStart) const {
    if(cnt == 0) {
        return;
    }
    GAPID_ASSERT((start < mCount) && (start + cnt <= mCount) /* Slice: start index out of bounds */);
    GAPID_ASSERT((dstStart < dst.mCount) && (dstStart + cnt <= dst.mCount) /* Slice: dst index out of bounds */);
    for(size_t i = 0; i < cnt; ++i) {
        dst.mBase[dstStart + i] = mBase[start + i];
    }
}

template<>
inline void Slice<uint8_t>::copy(const Slice<uint8_t>& dst, uint64_t start, uint64_t cnt, uint64_t dstStart) const {
    if(cnt == 0) {
        return;
    }
    GAPID_ASSERT((start < mCount) && (start + cnt <= mCount) /* Slice: start u8 index out of bounds */);
    GAPID_ASSERT((dstStart < dst.mCount) && (dstStart + cnt <= dst.mCount) /* Slice: dst u8 index out of bounds */);
    memmove(&dst.mBase[dstStart], &mBase[start], cnt);
}


}  // namespace gapii

#endif // GAPII_SLICE_H
