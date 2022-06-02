// Copyright (C) 2017 Google Inc.
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

#ifndef CORE_ARENA_H
#define CORE_ARENA_H

#include <stddef.h>
#include <stdint.h>

#include <array>
#include <atomic>
#include <list>
#include <unordered_map>

namespace core {

// Blocks will be created with sizes that range from
// [2^kMinBlockSizePower, 2^kMaxBlockSizePower] in powers of 2.
static const uint32_t kMinBlockSizePower = 5;
static const uint32_t kMaxBlockSizePower = 14;

// free_list_node is a simple linked list node that is used
// to track all of the currently unused blocks.
struct free_list_node {
  free_list_node* next;
};

// block_data contains metadata for all blocks if a particular
// size.
struct block_data {
  // next is the next unused block in this chunk.
  free_list_node* next = nullptr;
  // current_chunk is where the memory for this chunk resides.
  uint8_t* current_chunk = nullptr;
  // offset_of_next_allocation_in_chunk is the first location
  // in current_chunk that has never been touched. Used
  // when there are no blocks left in the free_list.
  size_t offset_of_next_allocation_in_chunk = 0;
};

// The header data for a chunk of memory. We can
// use this area to store information. It should always be
// smaller than the smallest block size.
struct chunk_header {
  uint16_t block_size;
  uint16_t block_index;
  uint32_t num_allocations;
};

// Arena is a memory allocator that owns each of the allocations made by it.
// If there are any outstanding allocations when the Arena is destructed then
// these allocations are automatically freed.
class Arena {
 public:
  Arena();
  ~Arena();

  // allocates a contiguous block of memory of at least the requested size and
  // alignment. This is internally synchronized and may be called from
  // multiple threads at once.
  void* allocate(uint32_t size, uint32_t align);

  // reallocates a block of memory previously allocated by this arena.
  // Data held in the previous allocation will be copied to the reallocated
  // address, but data may be trimmed if the new size is smaller than the
  // previous allocation.
  // This is internally synchronized and may be called from multiple
  // threads at once.
  void* reallocate(void* ptr, uint32_t size, uint32_t align);

  // free releases the memory previously allocated by this arena.
  // Once the memory is freed, it must not be used.
  // This is internalyl synchronized and may be called from multiple
  // threads at once.
  void free(void* ptr);

  // create constructs and returns a pointer to a new T.
  // This is internally synchronized and may be called from multiple
  // threads at once.
  template <typename T, typename... ARGS>
  inline T* create(ARGS&&... args);

  // destroy destructs an object constructed with create<T>().
  // This is internally synchronized and may be called from multiple
  // threads at once.
  template <typename T>
  inline void destroy(T* ptr);

  // returns the total number of allocations owned by this arena.
  size_t num_allocations() const;

  // returns the total number of bytes allocated by this arena.
  size_t num_bytes_allocated() const;

  // Dumps allocator stats to GAPID_ERROR.
  void dump_allocator_stats() const;

  // protects all of the memory in the arena. None of the memory
  // that was created by this arena may be written to after this
  // point.
  // No allocation / free operations may be in progress while this
  // is happening, futhermore, allocate/free may not be called
  // after the arena is protected.
  void protect();

  // unprotects all of the memory in the arena.
  void unprotect();

 private:
  void lock() const {
    uint32_t l = 0;
    while (!lock_.compare_exchange_weak(l, 1, std::memory_order_acquire)) {
      l = 0;
    }
  }
  void unlock() const { lock_.store(0, std::memory_order_release); }

  // chunks_ contains every chunk that has ever been allocated.
  std::list<chunk_header*> chunks_;
  // dedicated_allocations_ contains data about every allocation
  // that was too large for a block.
  std::unordered_map<uint8_t*, uint32_t> dedicated_allocations_;
  // blocks_ contains all of the freelists and blocksize specific information.
  std::array<block_data, kMaxBlockSizePower - kMinBlockSizePower + 1> blocks_;
  // lock_ is a simple atomic lock spinlock for locking. We have this lock
  // only for very brief periods of time.
  // We mark this mutable, so we can use lock/unlock in
  // num_allocations/num_bytes_allocated.
  mutable std::atomic<uint32_t> lock_;

  // page_size_ is needed for protecting memory.
  uint32_t page_size_;
  // protected_ is true when the memory in this allocator has been
  // protected.
  bool protected_;
};

template <typename T, typename... ARGS>
inline T* Arena::create(ARGS&&... args) {
  auto buf = allocate(sizeof(T), alignof(T));
  return new (buf) T(std::forward<ARGS>(args)...);
}

template <typename T>
inline void Arena::destroy(T* ptr) {
  ptr->~T();
  free(ptr);
}

}  // namespace core

#endif  //  CORE_ARENA_H
