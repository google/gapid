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

#include "maker.h"

#include "core/memory/arena/cc/arena.h"

#include <cstddef>  // nullptr_t

namespace gapil {

// Ref is a smart pointer implementation that is compatible with the ref
// shared pointers used by the gapil compiler. Several refs may share the same
// object.
template <typename T>
class Ref {
 public:
  using object_type = T;

  // Constructs a new ref that points to nothing.
  Ref();
  Ref(std::nullptr_t);

  // Constructs a new ref that shares ownership other other's data.
  Ref(const Ref& other);

  Ref(Ref&&);
  ~Ref();

  // Returns a ref that owns a new T constructed with the given arguments.
  template <typename... ARGS>
  inline static Ref create(core::Arena* arena, ARGS&&...);

  // Replaces the object owned by ref with the one owned by other.
  Ref& operator=(const Ref& other);

  // Returns true if the object owned by this is the same as the one owned by
  // other.
  bool operator==(const Ref& other) const;

  // Returns true if the object owned by this is not the same as the one owned
  // by other.
  bool operator!=(const Ref& other) const;

  // Returns a raw-pointer to the owned object.
  T* get() const;

  // Dereferences the stored pointer. The behavior is undefined if the stored
  // pointer is null.
  T* operator->() const;
  T& operator*() const;

  // Returns true if the ref points to an object (is non-null).
  operator bool() const;

 private:
  struct Allocation {
    uint32_t ref_count;
    core::Arena* arena;  // arena that owns this object allocation.
    T object;

    void reference();
    void release();
  };

  Ref(Allocation*);

  Allocation* ptr;
};

template <typename T>
template <typename... ARGS>
Ref<T> Ref<T>::create(core::Arena* arena, ARGS&&... args) {
  auto buf = arena->allocate(sizeof(Allocation), alignof(Allocation));
  auto ptr = reinterpret_cast<Allocation*>(buf);
  ptr->ref_count = 1;
  ptr->arena = arena;
  inplace_new(&ptr->object, arena, std::forward<ARGS>(args)...);
  return Ref(ptr);
}

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_REF_H__
