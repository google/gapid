/* Copyright 2019 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#ifndef CORE_MEMORY_ARENA_CC_STL_COMPATIBLE_ALLOCATOR_H__
#define CORE_MEMORY_ARENA_CC_STL_COMPATIBLE_ALLOCATOR_H__

#include <limits>
#include "core/memory/arena/cc/arena.h"

namespace core {

// This allocator implements the allocator interface
// as needed by STL objects.
template <typename T>
struct StlCompatibleAllocator {
  typedef T value_type;
  typedef T* pointer;
  typedef const T* const_pointer;
  typedef T& reference;
  typedef const T& const_reference;
  typedef size_t size_type;
  typedef size_t difference_type;
  // An equivalent STL allocator for a different type.
  template <class U>
  struct rebind {
    typedef StlCompatibleAllocator<U> other;
  };

  // Creation of the allocator is allowed, but if anyone were to try to use
  // an object with a null allocator, it would fail. This however allows us to
  // default-construct a bunch of objects in a container, and fill, in their
  // allocators on first use.
  StlCompatibleAllocator() : arena_(nullptr) {}

  // All allocations will be done through this allocator. It must remain
  // valid until this StlCompatibleAllocator and all allocators created
  // from it have been destroyed.
  StlCompatibleAllocator(Arena* arena) : arena_(arena) {}

  template <typename U>
  StlCompatibleAllocator(const StlCompatibleAllocator<U>& other) {
    arena_ = other.arena_;
  }

  // Copy constructs an object of type T at the location given by p.
  void construct(pointer p, const_reference val) { new (p) T(val); }

  // Constructs an object of Type U at the location given by P passing
  // through all other arguments to the constructor.
  template <typename U, typename... Args>
  void construct(U* p, Args&&... args) {
    ::new ((void*)p) U(std::forward<Args>(args)...);
  }

  // Deconstructs the object at p. It does not free the memory.
  void destroy(pointer p) { ((T*)p)->~T(); }

  // Deconstructs the object at p. It does not free the memory.
  template <typename U>
  void destroy(U* p) {
    p->~U();
  }

  // Allocates the memory for n objects of type T. Does not
  // actually construct the objects.
  T* allocate(std::size_t n) {
    return reinterpret_cast<T*>(arena_->allocate(sizeof(T) * n, 1));
  }
  // Deallocates the memory for n Objects of size T.
  void deallocate(T* p, std::size_t n) { arena_->free(p); }

  // Returns the internal allocator. This is useful to get at the allocation
  // information.
  const Arena* get_internal() const { return arena_; }

  // Returns the maximum theoretically possible number of T stored in this
  // allocator.
  size_type max_size() const {
    return std::numeric_limits<size_type>::max() / sizeof(value_type);
  }

 private:
  template <typename U>
  friend struct StlCompatibleAllocator;
  Arena* arena_;
};

template <typename T, typename U>
bool operator==(const StlCompatibleAllocator<T>& a,
                const StlCompatibleAllocator<U>& b) {
  return a.get_internal() == b.get_internal();
}

template <typename T, typename U>
bool operator!=(const StlCompatibleAllocator<T>& a,
                const StlCompatibleAllocator<U>& b) {
  return !(a == b);
}
}  // namespace core

#endif  //  CORE_MEMORY_ARENA_CC_STL_COMPATIBLE_ALLOCATOR_H__
