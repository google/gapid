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

#ifndef __GAPIL_RUNTIME_REF_H__
#define __GAPIL_RUNTIME_REF_H__

#include "runtime.h"

#include "core/memory/arena/cc/arena.h"

#include <cstddef> // nullptr_t

namespace gapil {

template<typename T>
class Ref {
public:
    using object_type = T;

    Ref();
    Ref(const Ref&);
    Ref(std::nullptr_t);
    Ref(Ref&&);
    ~Ref();

    template <class ...ARGS>
    inline static Ref create(core::Arena* arena, ARGS...);

    Ref& operator = (const Ref& other);

    bool operator == (const Ref& other) const;
    bool operator != (const Ref& other) const;

    T* get() const;

    T* operator->() const;
    T& operator*() const;

    operator bool() const;

private:
    struct Allocation {
        uint32_t ref_count;
        arena_t* arena; // arena that owns this object allocation.
        T        object;

        void reference();
        void release();
    };

    Ref(Allocation*);

    Allocation* ptr;
};


template<typename T>
template <class ...ARGS>
Ref<T> Ref<T>::create(core::Arena* arena, ARGS... args) {
    auto ptr = arena->create<Allocation>();
    ptr->ref_count = 1;
    ptr->arena = reinterpret_cast<arena_t*>(arena);
    new(&ptr->object) T(args...);
    return Ref(ptr);
}

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_REF_H__