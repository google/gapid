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

#include "arena.h"

#include "core/cc/assert.h"
#include "core/cc/target.h"

#include <algorithm>
#include <cassert>
#include <cstdlib>
#include <cstring>

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

#if (TARGET_OS == GAPID_OS_LINUX) || (TARGET_OS == GAPID_OS_ANDROID) || \
    (TARGET_OS == GAPID_OS_OSX) || (TARGET_OS == GAPID_OS_FUCHSIA)
#include <sys/mman.h>
#include <unistd.h>  // getpagesize
#elif TARGET_OS == GAPID_OS_WINDOWS
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#undef WIN32_LEAN_AND_MEAN
#endif

namespace {

// k[Min,Max]BlockSize are the smallest and largest allocatable
// blocks. Anything smaller will be rounded up, anything larger
// will get a dedicated allocation.
static const uint32_t kMinBlockSize = 1 << core::kMinBlockSizePower;
static const uint32_t kMaxBlockSize = 1 << core::kMaxBlockSizePower;

// A single chunk is 2^kChunkSizePower bytes large.
static const uint32_t kChunkSizePower = 21;
static_assert(kChunkSizePower > core::kMaxBlockSizePower,
              "There must be at least 2 blocks in every chunk");

static const uint32_t kChunkSize = 1 << kChunkSizePower;
static const uintptr_t kChunkMask = ~(static_cast<uintptr_t>(kChunkSize) - 1);

static_assert(sizeof(core::chunk_header) <= kMinBlockSize,
              "Cannot fit the chunk header inside a single block");

// Returns the exponent of the next power of 2 larger than
// val
uint32_t next_power_of_2(uint32_t val) { return (33 - __builtin_clz(val - 1)); }

// round_up_to rounds the given value to the next multiple.
// This is invalid if multiple is 0.
uint32_t round_up_to(uint32_t val, uint32_t multiple) {
  int rem = val % multiple;
  if (rem == 0) {
    return val;
  }
  return val + multiple - rem;
}

void* allocate_aligned(uint32_t val, uint32_t alignment) {
#if TARGET_OS == GAPID_OS_WINDOWS
  return _aligned_malloc(val, alignment);
#else
  void* allocated = nullptr;
  GAPID_ASSERT(0 == posix_memalign(&allocated, alignment, val));
  return allocated;
#endif
}

void free_aligned(void* val) {
#if TARGET_OS == GAPID_OS_WINDOWS
  _aligned_free(val);
#else
  ::free(val);
#endif
}

void protect_range(void* val, uint32_t size) {
#if TARGET_OS == GAPID_OS_WINDOWS
  DWORD old_protect = 0;
  VirtualProtect(val, size, PAGE_READONLY, &old_protect);
#else
  mprotect(val, size, PROT_READ);
#endif
}

void unprotect_range(void* val, uint32_t size) {
#if TARGET_OS == GAPID_OS_WINDOWS
  DWORD old_protect = 0;
  VirtualProtect(val, size, PAGE_READWRITE, &old_protect);
#else
  mprotect(val, size, PROT_READ | PROT_WRITE);
#endif
}

}  // anonymous namespace

namespace core {

Arena::Arena() : lock_(0), protected_(false) {
#if (TARGET_OS == GAPID_OS_LINUX) || (TARGET_OS == GAPID_OS_ANDROID) || \
    (TARGET_OS == GAPID_OS_OSX)
  page_size_ = getpagesize();
#elif (TARGET_OS == GAPID_OS_WINDOWS)
  SYSTEM_INFO si;
  GetSystemInfo(&si);
  page_size_ = si.dwPageSize;
#endif
}

Arena::~Arena() {
  if (protected_) {
    unprotect();
  }
  for (auto& it : dedicated_allocations_) {
    free_aligned(it.first);
  }
  for (auto chunk : chunks_) {
    free_aligned(chunk);
  }
}

void* Arena::allocate(uint32_t size, uint32_t align) {
  size = std::max(size, kMinBlockSize);
  void* allocation = nullptr;
  if (size > kMaxBlockSize) {
    // Align must be a power of 2.
    // Use normal assert here so it goes away in release.
    assert((align & (align - 1)) == 0);

    // posix_memalign states that the alignment must be a multiple
    // of sizeof(void*).
    align = std::max(align, page_size_);
    allocation = allocate_aligned(round_up_to(size, page_size_), align);
    lock();
    { dedicated_allocations_[reinterpret_cast<uint8_t*>(allocation)] = size; }
    unlock();
  } else {
    // Calculate the bucket index.
    const uint32_t bucket = next_power_of_2(size) - kMinBlockSizePower - 1;
    const uint32_t block_size = 1 << (bucket + kMinBlockSizePower);
    block_data& data = blocks_[bucket];
    chunk_header* header;
    lock();
    if (data.next) {
      // If there was a free block in the freelist, use that.
      allocation = data.next;
      data.next = data.next->next;
      uintptr_t val = reinterpret_cast<uintptr_t>(allocation);
      val &= kChunkMask;
      header = reinterpret_cast<chunk_header*>(val);
    } else if (data.current_chunk) {
      // If there was no free block in the free-list, but we
      // have at least one free block in the current chunk
      // then use that.
      allocation = data.current_chunk + data.offset_of_next_allocation_in_chunk;
      header = reinterpret_cast<chunk_header*>(data.current_chunk);
      data.offset_of_next_allocation_in_chunk += block_size;
      if (data.offset_of_next_allocation_in_chunk == kChunkSize) {
        // If this was the last block in the chunk, then stop using
        // this chunk.
        data.offset_of_next_allocation_in_chunk = 0;
        data.current_chunk = nullptr;
      }
    } else {
      chunk_header* val = 0;
      // If we can't actually allocate memory that is a problem.
      val =
          static_cast<chunk_header*>(allocate_aligned(kChunkSize, kChunkSize));
      data.current_chunk = reinterpret_cast<uint8_t*>(val);
      chunks_.push_back(val);
      val->block_size = block_size;
      val->block_index = bucket;
      val->num_allocations = 0;
      // The first allocation is offset by 1 block.
      allocation = data.current_chunk + block_size;
      data.offset_of_next_allocation_in_chunk = 2 * block_size;
      header = reinterpret_cast<chunk_header*>(data.current_chunk);
    }
    header->num_allocations += 1;
    unlock();
  }
  return allocation;
}

void* Arena::reallocate(void* ptr, uint32_t size, uint32_t align) {
  if (ptr == nullptr) {
    return allocate(size, align);
  }

  bool reallocate = false;
  uint32_t old_size = 0;
  lock();
  auto it = dedicated_allocations_.find(static_cast<uint8_t*>(ptr));
  if (it != dedicated_allocations_.end()) {
    reallocate = true;
    old_size = it->second;
    unlock();
  } else {
    unlock();
    uintptr_t val = reinterpret_cast<uintptr_t>(ptr);
    val &= kChunkMask;
    chunk_header* header = reinterpret_cast<chunk_header*>(val);
    if (header->block_size < size) {
      reallocate = true;
      old_size = header->block_size;
    }
  }

  if (reallocate) {
    void* new_ptr = allocate(size, align);
    memcpy(new_ptr, ptr, std::min(size, old_size));
    Arena::free(ptr);
    return new_ptr;
  }
  return ptr;
}

void Arena::free(void* ptr) {
  if (!ptr) {
    return;
  }
  bool dedicated = false;
  lock();
  dedicated = dedicated_allocations_.erase(static_cast<uint8_t*>(ptr));

  if (!dedicated) {
    uintptr_t val = reinterpret_cast<uintptr_t>(ptr);
    val &= kChunkMask;
    chunk_header* header = reinterpret_cast<chunk_header*>(val);
    header->num_allocations -= 1;
    const uint16_t bucket = header->block_index;

    reinterpret_cast<free_list_node*>(ptr)->next = blocks_[bucket].next;
    blocks_[bucket].next = reinterpret_cast<free_list_node*>(ptr);
    unlock();
  } else {
    unlock();
    free_aligned(ptr);
  }
}

size_t Arena::num_allocations() const {
  lock();
  size_t alloc_count = dedicated_allocations_.size();
  for (auto chunk : chunks_) {
    alloc_count += chunk->num_allocations;
  }
  unlock();
  return alloc_count;
}

size_t Arena::num_bytes_allocated() const {
  lock();
  size_t alloc_amount = 0;
  for (auto& it : dedicated_allocations_) {
    alloc_amount += it.second;
  }
  for (auto chunk : chunks_) {
    alloc_amount += chunk->num_allocations * chunk->block_size;
  }
  unlock();
  return alloc_amount;
}

void Arena::dump_allocator_stats() const {
  uint32_t total_chunk_memory = chunks_.size() * kChunkSize;
  uint32_t total_dedicated_memory = 0;
  uint32_t total_used_dedicated_memory = 0;
  for (auto& it : dedicated_allocations_) {
    total_dedicated_memory += round_up_to(it.second, page_size_);
    total_used_dedicated_memory += it.second;
  }
  uint32_t total_used_chunk_memory = 0;
  uint32_t total_header_memory = 0;
  for (auto chunk : chunks_) {
    total_used_chunk_memory += chunk->num_allocations * chunk->block_size;
    total_header_memory += chunk->block_size;
  }

  GAPID_ERROR("----------------- ARENA STATS -----------------");
  GAPID_ERROR("Num Chunks: %35zu", chunks_.size());
  GAPID_ERROR("Num Dedicated Allocations: %20zu",
              dedicated_allocations_.size());
  GAPID_ERROR("Total Memory Reserved: %24" PRIu32,
              total_chunk_memory + total_dedicated_memory);
  GAPID_ERROR("Total Memory Reserved [Chunks]: %15" PRIu32, total_chunk_memory);
  GAPID_ERROR("Total Memory Reserved [Dedicated]: %12" PRIu32,
              total_dedicated_memory);
  GAPID_ERROR("Total Memory Used [Chunks]: %19" PRIu32,
              total_used_chunk_memory);
  GAPID_ERROR("Total Memory Used [Dedicated]: %16" PRIu32,
              total_used_dedicated_memory);
  GAPID_ERROR("Memory Overhead [Headers]: %20" PRIu32, total_header_memory);
  GAPID_ERROR(
      "Memory Overhead [Unused]: %21" PRIu32,
      total_chunk_memory - total_header_memory - total_used_chunk_memory);
  GAPID_ERROR("Memory Overhead [Dedicated]: %18" PRIu32,
              total_dedicated_memory - total_used_dedicated_memory);
  GAPID_ERROR("Memory Efficiency [Chunks] %20f",
              (float)total_used_chunk_memory / (float)total_chunk_memory);
  GAPID_ERROR(
      "Memory Efficiency [Dedicated] %17f",
      (float)total_used_dedicated_memory / (float)total_dedicated_memory);
  GAPID_ERROR("---------------- FREELIST STATS ---------------");

  uint32_t i = 0;
  for (auto& block : blocks_) {
    uint32_t freelist_count = 0;
    for (free_list_node* node = block.next; node != nullptr;
         node = node->next) {
      freelist_count += 1;
    }
    GAPID_ERROR("Freelist [%6" PRIu32 "]: %28" PRIu32,
                1 << (kMinBlockSizePower + i), freelist_count);
    i++;
  }
  GAPID_ERROR("-----------------------------------------------");
}

void Arena::protect() {
  for (auto& it : dedicated_allocations_) {
    protect_range(it.first, round_up_to(it.second, page_size_));
  }
  for (auto chunk : chunks_) {
    protect_range(chunk, kChunkSize);
  }
  protected_ = true;
}

void Arena::unprotect() {
  for (auto& it : dedicated_allocations_) {
    unprotect_range(it.first, round_up_to(it.second, page_size_));
  }
  for (auto chunk : chunks_) {
    unprotect_range(chunk, kChunkSize);
  }
  protected_ = false;
}

}  // namespace core

extern "C" {

arena* arena_create() { return reinterpret_cast<arena*>(new core::Arena()); }

void arena_destroy(arena* a) { delete reinterpret_cast<core::Arena*>(a); }

void* arena_alloc(arena* a, uint32_t size, uint32_t align) {
  return reinterpret_cast<core::Arena*>(a)->allocate(size, align);
}

void* arena_realloc(arena* a, void* ptr, uint32_t size, uint32_t align) {
  return reinterpret_cast<core::Arena*>(a)->reallocate(ptr, size, align);
}

void arena_free(arena* a, void* ptr) {
  reinterpret_cast<core::Arena*>(a)->free(ptr);
}

// arena_stats returns statistics of the current state of the arena.
void arena_stats(arena* a, size_t* num_allocations,
                 size_t* num_bytes_allocated) {
  auto arena = reinterpret_cast<core::Arena*>(a);
  *num_allocations = arena->num_allocations();
  *num_bytes_allocated = arena->num_bytes_allocated();
}

}  // extern "C"
