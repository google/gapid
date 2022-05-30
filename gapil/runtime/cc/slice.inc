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
void Slice<T>::init(pool_t* pool, uint64_t root, uint64_t base, uint64_t size, uint64_t count, bool add_ref /* = true */) {
    data.pool = pool;
    data.root = root;
    data.base = base;
    data.size = size;
    data.count = count;

    if (add_ref && pool != nullptr) {
        reference();
    }
}

template <typename T>
Slice<T>::Slice() {
    init(nullptr, 0, 0, 0, 0);
}

template <typename T>
Slice<T>::Slice(const Slice<T>& other) {
    init(other.data.pool,
         other.data.root,
         other.data.base,
         other.data.size,
         other.data.count);
}

template <typename T>
Slice<T>::Slice(Slice<T>&& other) {
    init(other.data.pool,
         other.data.root,
         other.data.base,
         other.data.size,
         other.data.count);

    other.data.pool = nullptr;
}

template <typename T>
Slice<T>::Slice(T* base, uint64_t count) {
    auto ptr = static_cast<uint64_t>(reinterpret_cast<uintptr_t>(base));
    init(/* pool */  nullptr,
         /* root */  ptr,
         /* base */  ptr,
         /* size */  count * sizeof(T),
         /* count */ count);
}

template <typename T>
Slice<T>::Slice(pool_t* pool, uint64_t root, uint64_t base, uint64_t size, uint64_t count, bool add_ref /* = true */) {
    init(pool, root, base, size, count, add_ref);
}

template <typename T>
Slice<T>::~Slice() {
    if (data.pool != nullptr) {
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
    if (data.pool != nullptr) {
        release();
    }
    init(other.data.pool,
         other.data.root,
         other.data.base,
         other.data.size,
         other.data.count);
    return *this;
}

template <typename T>
bool Slice<T>::operator == (const Slice<T>& other) const {
    return data.pool == other.data.pool &&
           data.root == other.data.root &&
           data.base == other.data.base &&
           data.size == other.data.size &&
           data.count == other.data.count;
}

template <typename T>
uint64_t Slice<T>::root() const {
    return data.root;
}

template <typename T>
uint64_t Slice<T>::base() const {
    return data.base;
}

template <typename T>
uint64_t Slice<T>::count() const {
    return data.count;
}

template <typename T>
uint64_t Slice<T>::size() const {
    return data.size;
}

template <typename T>
bool Slice<T>::is_app_pool() const {
    return data.pool == nullptr;
}

template <typename T>
uint32_t Slice<T>::pool_id() const {
    return (data.pool != nullptr) ? data.pool->id : 0;
}

template <typename T>
const pool_t* Slice<T>::pool() const {
    return data.pool;
}

template <typename T>
const void* Slice<T>::instance_ptr() const {
    return &data;
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
    Slice<T> out;
    out.data.pool = data.pool;
    out.data.root = data.root;
    out.data.base = data.base + start * sizeof(T);
    out.data.size = count * sizeof(T);
    out.data.count = count;
    if (data.pool != nullptr) {
        reference();
    }
    return out;
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
    auto count = data.size / sizeof(U);
    return Slice<U>(/* pool */  data.pool,
                    /* root */  data.root,
                    /* base */  data.base,
                    /* size */  count * sizeof(U),
                    /* count */ count);
}

template <typename T>
T* Slice<T>::begin() const {
    uint64_t pool_base = 0;
    if (data.pool != nullptr) {
        pool_base = static_cast<uint64_t>(reinterpret_cast<uintptr_t>(data.pool->buffer));
    }
    return reinterpret_cast<T*>(pool_base + data.base);
}

template <typename T>
T* Slice<T>::end() const {
    return begin() + count();
}

template <typename T>
void Slice<T>::release() {
    GAPID_ASSERT_MSG(data.pool->ref_count > 0, "attempting to release freed pool");
    data.pool->ref_count--;
    if (data.pool->ref_count == 0) {
        gapil_free_pool(data.pool);
    }
    data.pool = nullptr;
}

template <typename T>
void Slice<T>::reference() const {
    GAPID_ASSERT_MSG(data.pool->ref_count > 0, "attempting to reference freed pool");
    data.pool->ref_count++;
}

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_SLICE_INC__

