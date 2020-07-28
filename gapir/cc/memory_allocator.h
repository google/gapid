/*
 * Copyright (C) 2019 Google Inc.
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

#ifndef GAPIR_MEMORY_ALLOCATOR_H
#define GAPIR_MEMORY_ALLOCATOR_H

#include <map>
#include <memory>

namespace gapir {

class MemoryAllocator;

// An allocator class used for volatile and cached replay data.
// Volatile data must be resized for multi-part replays and may not be
// relocated, which prevents reallocation of a new, larger region when
// space runs out. To solve this problem, the allocator laid out here
// grabs a big piece of memory off the system allocator and attempts
// to efficiently parse it out into static (non relocatable, suitable
// for volatile memory) regions and purgable (relocatable, prematurely
// releaseable) regions suitable for cache data. Expansion of volatile
// memory in this scheme will simply relocate or purge cache data
// to ensure the program may continue to operate within a fixed memory
// footprint at the cost of deminished cache performance.

class MemoryAllocator {
 private:
  class MemoryRegion {
   public:
    MemoryRegion() : _offset(0), _size(0) {}
    MemoryRegion(size_t offset, size_t size) : _offset(offset), _size(size) {}
    virtual ~MemoryRegion() {}

    size_t getOffset() const { return _offset; }
    size_t getSize() const { return _size; }

    bool operator<(const MemoryRegion& rhs) const {
      return _offset < rhs._offset;
    }

   private:
    size_t _offset;
    size_t _size;
  };

 public:
  // This allocator class returns allocations by "Handle". This class provides
  // an extra level of indirection on top of a standard pointer. This allows
  // the allocator to relocate and prematurely release purgable allocations
  // without having to notify clients via callback. Be aware that indirecting
  // a returned handle multiple times may return different addresses.
  // Accordingly you should check for != nullptr before every use.
  // Be careful how this interacts with a multi-threaded environment.
  // No locking or pinning is provided in the current allocator design
  // so you need to take care of that yourself for now!
  class Handle {
   public:
    Handle() : _backing(_emptyMap.end()) {}

    bool operator!() const { return *this == nullptr; }
    bool operator==(unsigned char* rhs) const {
      return (_backing == _emptyMap.end() ? nullptr
                                          : _backing->second->first) == rhs;
    }
    bool operator!=(unsigned char* rhs) const {
      return (_backing == _emptyMap.end() ? nullptr
                                          : _backing->second->first) != rhs;
    }
    unsigned char& operator*() const { return *_backing->second->first; }
    unsigned char& operator[](size_t n) const {
      return _backing->second->first[n];
    }

   private:
    Handle(std::map<unsigned int,
                    std::map<unsigned char*, MemoryRegion>::iterator>::iterator
               backing)
        : _backing(backing) {}

    std::map<unsigned int,
             std::map<unsigned char*, MemoryRegion>::iterator>::iterator
        _backing;

    static std::map<unsigned int,
                    std::map<unsigned char*, MemoryRegion>::iterator>
        _emptyMap;

    friend class MemoryAllocator;
  };

 public:
  static std::unique_ptr<MemoryAllocator> create(size_t heapSize);

  MemoryAllocator(size_t heapSize);
  ~MemoryAllocator();

  Handle allocateStatic(size_t size);
  Handle allocatePurgable(size_t size, bool allowRelocate = true);

  bool resizeStaticAllocation(const Handle& address, size_t size);

  bool releaseAllocation(Handle& address);

  bool garbageCollect() { return compactPurgableMemory(); }

  // Get the amount of memory used for different classes of allocation. Warning:
  // This currently has O(N) cost. If that is causing you trouble go ahead and
  // maintain an internal record so you can answer this in O(1).
  size_t getTotalSize() const { return heapSize_; }
  size_t getTotalStaticDataUsage() const;
  size_t getTotalPurgableDataUsage() const;
  size_t getTotalDataUsage() const {
    return getTotalStaticDataUsage() + getTotalPurgableDataUsage();
  }

 private:
  void purgeOrRelocateRange(size_t start, size_t end);
  bool compactPurgableMemory();

  size_t getClosestStaticData(
      size_t belowOffset,
      std::map<unsigned char*, MemoryRegion>::const_iterator*
          closestStaticIter = nullptr) const;

  Handle registerStaticAllocate(const MemoryRegion& newRegion);
  Handle registerPurgableAllocate(const MemoryRegion& newRegion);

  void registerResize(const MemoryRegion& resizedRegion);

  void registerRelocate(const MemoryRegion& from, const MemoryRegion& to);
  void registerPurge(const MemoryRegion& purge);

  void registerStaticRelease(const MemoryRegion& release);
  void registerPurgableRelease(const MemoryRegion& release);

  size_t heapSize_;
  unsigned char* heap_;

  size_t purgableHead_;

  std::map<unsigned char*, MemoryRegion> staticRegionMap_;
  std::map<unsigned char*, MemoryRegion> purgableRegionMap_;

  std::map<unsigned int, std::map<unsigned char*, MemoryRegion>::iterator>
      relocationMap_;
  std::map<unsigned char*, unsigned int> inverseRelocationMap_;

  unsigned int idGenerator_;
};

}  // namespace gapir

#endif  // GAPIR_MEMORY_ALLOCATOR_H
