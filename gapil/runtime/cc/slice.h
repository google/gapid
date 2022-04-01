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

#ifndef __GAPIL_RUNTIME_SLICE_H__
#define __GAPIL_RUNTIME_SLICE_H__

#include "runtime.h"

#include <functional>

namespace core {
class Arena;
}  // namespace core

namespace gapil {

// Slice is a vector of elements of type T backed by a pool. Slice is compatible
// with the slices produced by the gapil compiler.
// Slices hold references to their pool, and several slices may share the same
// underlying data.
template <typename T>
class Slice {
 public:
  // Constructs a slice that points to nothing.
  inline Slice();

  // Constructs a slice which shares ownership over the data with other.
  inline Slice(const Slice<T>& other);

  // Constructs a new slice and pool sized to the given number of elements.
  inline Slice(T* base, uint64_t count);

  // Constructs a new slice given the full explicit parameters.
  inline Slice(pool_t* pool, uint64_t root, uint64_t base, uint64_t size,
               uint64_t count, bool add_ref = true);

  // Creates and returns a new slice wrapping the given pool.
  // If add_ref is true then the pool's reference count will be incremented.
  inline static Slice create(pool_t* pool, bool add_ref);

  // Creates and returns a new slice and pool sized to the given number of
  // elements.
  inline static Slice create(context_t* ctx, uint64_t count);

  inline Slice(Slice<T>&&);
  inline ~Slice();

  // Copy assignment
  inline Slice<T>& operator=(const Slice<T>& other);

  // Equality operator.
  inline bool operator==(const Slice<T>& other) const;

  // Returns the number of elements in the slice.
  inline uint64_t count() const;

  // Returns the size of the slice in bytes.
  inline uint64_t size() const;

  // Returns true if this is a slice on the application pool (external memory).
  inline bool is_app_pool() const;

  // Returns the underlying pool identifier.
  inline uint32_t pool_id() const;

  // Returns the underlying pool.
  inline const pool_t* pool() const;

  // Returns true if the slice contains the specified value.
  inline bool contains(const T& value) const;

  // Returns a new subset slice from this slice.
  inline Slice<T> operator()(uint64_t start, uint64_t end) const;

  // Returns a reference to a single element in the slice.
  // Care must be taken to not mutate data in the application pool.
  inline T& operator[](uint64_t index) const;

  // Copies count elements starting at start into the dst Slice starting at
  // dstStart.
  inline void copy(const Slice<T>& dst, uint64_t start, uint64_t count,
                   uint64_t dstStart) const;

  // Casts this slice to a slice of type U.
  // The return slice length will be calculated so that the returned slice
  // length is no longer (in bytes) than this slice.
  template <typename U>
  inline Slice<U> as() const;

  // Support for range-based for looping
  inline T* begin() const;
  inline T* end() const;

 private:
  void init(pool_t* pool, uint64_t root, uint64_t base, uint64_t size,
            uint64_t count, bool add_ref = true);

  void reference() const;
  void release();

  slice_t data;
};

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_SLICE_H__
