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

#ifndef __GAPIL_RUNTIME_REF_INC__
#define __GAPIL_RUNTIME_REF_INC__

#include "ref.h"

#include "core/cc/assert.h"
#include "core/memory/arena/cc/arena.h"

namespace gapil {

template<typename T>
Ref<T>::Ref() : ptr(nullptr) {}

template<typename T>
Ref<T>::Ref(const Ref<T>& s) {
    ptr = s.ptr;
    if (ptr != nullptr) {
        ptr->ref_count++;
    }
}

template<typename T>
Ref<T>::Ref(std::nullptr_t) : ptr(nullptr){}

template<typename T>
Ref<T>::Ref(Ref<T>&& s) {
    ptr = s.ptr;
    s.ptr = nullptr;
}

template<typename T>
Ref<T>::~Ref() {
    if (ptr != nullptr) {
        ptr->release();
    }
}

template<typename T>
Ref<T>::Ref(Allocation* p) : ptr(p) {}

template<typename T>
Ref<T>& Ref<T>::operator = (const Ref<T>& other) {
    if (ptr != other.ptr) {
        if (ptr != nullptr) {
            ptr->release();
        }
        ptr = other.ptr;
        if (ptr != nullptr) {
            ptr->reference();
        }
    }
    return *this;
}

template<typename T>
bool Ref<T>::operator == (const Ref<T>& other) const {
    return ptr == other.ptr;
}

template<typename T>
bool Ref<T>::operator != (const Ref<T>& other) const {
    return ptr != other.ptr;
}

template<typename T>
T* Ref<T>::get() const {
    return ptr ? &ptr->object : nullptr;
}

template<typename T>
T* Ref<T>::operator->() const {
    return get();
}

template<typename T>
T& Ref<T>::operator*() const {
    return ptr->object;
}

template<typename T>
Ref<T>::operator bool() const {
    return ptr != nullptr;
}

template<typename T>
void Ref<T>::Allocation::release() {
    GAPID_ASSERT_MSG(ref_count > 0, "attempting to release freed object");
    ref_count--;
    if (ref_count == 0) {
        auto arena = reinterpret_cast<core::Arena*>(this->arena);
        object.~T();
        arena->free(this);
    }
}

template<typename T>
void Ref<T>::Allocation::reference() {
    GAPID_ASSERT_MSG(ref_count > 0, "attempting to reference freed object");
    ref_count++;
}


}  // namespace gapil

#endif  // __GAPIL_RUNTIME_REF_INC__

