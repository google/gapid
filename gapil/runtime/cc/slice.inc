// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#ifndef __GAPIL_RUNTIME_SLICE_INC__
#define __GAPIL_RUNTIME_SLICE_INC__

#include "slice.h"

#include "core/cc/assert.h"

namespace gapil {

template <typename T>
Slice<T>::~Slice() {
    if (pool_ != nullptr) {
        release();
    }
}

template <typename T>
Slice<T> Slice<T>::create(pool_t* pool, bool add_ref) {
    return Slice<T>(/* pool */ pool,
                    /* root */ 0,
                    /* base */ 0,
                    /* size */ pool->size,
                    /* count */ pool->size / sizeof(T),
                    /* add_ref */ add_ref);
}

template <typename T>
Slice<T> Slice<T>::create(context_t* ctx, uint64_t count) {
    auto size = count * sizeof(T);
    auto pool = gapil_make_pool(ctx, size);
    return Slice<T>(/* pool */ pool,
                    /* root */ 0,
                    /* base */ 0,
                    /* size */ pool->size,
                    /* count */ count,
                    /* add_ref */ false);
}

template <typename T>
Slice<T>& Slice<T>::operator = (const Slice<T>& other) {
    if (pool_ != other.pool_) {
        if (pool_ != nullptr) {
            release();
        }
        pool_ = other.pool_;
        if (pool_ != nullptr) {
            reference();
        }
    }
    root_ = other.root_;
    base_ = other.base_;
    size_ = other.size_;
    count_ = other.count_;
    return *this;
}

template <typename T>
bool Slice<T>::operator == (const Slice<T>& other) const {
    return pool_ == other.pool_ &&
           root_ == other.root_ &&
           base_ == other.base_ &&
           size_ == other.size_ &&
           count_ == other.count_;
}

template <typename T>
bool Slice<T>::contains(const T& value) const {
    for (auto el : *this) {
        if (el == value) {
            return true;
        }
    }
    return false;
}

template <typename T>
Slice<T> Slice<T>::operator()(uint64_t start, uint64_t end) const {
    GAPID_ASSERT_MSG(start <= end, "slice start is after end");
    GAPID_ASSERT_MSG(end <= count(), "slice index out of bounds");
    auto count = end - start;
    return Slice<T>(/* pool */  pool_,
                    /* root */  root_,
                    /* base */  base_ + start * sizeof(T),
                    /* size */  count * sizeof(T),
                    /* count */ count);
}

template <typename T>
T& Slice<T>::operator[](uint64_t index) const {
    GAPID_ASSERT_MSG(index < count(), "slice index out of bounds");
    return begin()[index];
}

template <typename T>
void Slice<T>::copy(const Slice<T>& dst, uint64_t start, uint64_t count, uint64_t dstStart) const {
    if (count == 0) {
        return;
    }
    for(size_t i = 0; i < count; ++i) {
        dst[dstStart + i] = (*this)[start + i];
    }
}

template <typename T>
template <typename U>
Slice<U> Slice<T>::as() const {
    auto count = size_ / sizeof(U);
    return Slice<U>(/* pool */  pool_,
                    /* root */  root_,
                    /* base */  base_,
                    /* size */  count * sizeof(U),
                    /* count */ count);
}

template <typename T>
T* Slice<T>::begin() const {
    uint64_t pool_base = 0;
    if (pool_ != nullptr) {
        pool_base = static_cast<uint64_t>(reinterpret_cast<uintptr_t>(pool_->buffer));
    }
    return reinterpret_cast<T*>(pool_base + base_);
}

template <typename T>
T* Slice<T>::end() const {
    return begin() + count();
}

template <typename T>
void Slice<T>::release() {
    GAPID_ASSERT_MSG(pool_->ref_count > 0, "attempting to release freed pool");
    pool_->ref_count--;
    if (pool_->ref_count == 0) {
        gapil_free_pool(pool_);
    }
    pool_ = nullptr;
}

template <typename T>
void Slice<T>::reference() const {
    GAPID_ASSERT_MSG(pool_->ref_count > 0, "attempting to reference freed pool");
    pool_->ref_count++;
}

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_SLICE_INC__

