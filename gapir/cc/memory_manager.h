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

#ifndef GAPIR_MEMORY_MANAGER_H
#define GAPIR_MEMORY_MANAGER_H

#include <stdint.h>

#include <memory>
#include <type_traits>
#include <utility>
#include <vector>

#include "memory_allocator.h"

namespace gapir {

class MemoryAllocator;

// Memory manager class for managing the memory used by the replay system.
//
// The layout of the memory managed by the memory manager (extra paddings are
// possible between the different memory regions): | In memory resource cache |
// Volatile Memory | Replay data |
class MemoryManager {
 public:
  MemoryManager(std::shared_ptr<MemoryAllocator> allocator);
  ~MemoryManager();

  // Sets the size of the replay data.
  void setReplayData(const uint8_t* constantMemoryBase,
                     uint32_t constantMemorySize,
                     const uint8_t* opcodeMemoryBase,
                     uint32_t opcodeMemorySize);

  // Sets the size of the volatile memory. Returns true if the given size fits
  // in the memory and false otherwise
  bool setVolatileMemory(uint32_t size);

  // Returns the size and the base address of the different memory regions
  // managed by the memory manager
  void* getBaseAddress() const { return &mMemory[0]; }

  const void* getOpcodeAddress() const { return mOpcodeMemory.base; }
  const void* getConstantAddress() const { return mConstantMemory.base; }
  void* getVolatileAddress() const { return mVolatileMemory.base; }
  uint32_t getOpcodeSize() const { return mOpcodeMemory.size; }
  uint32_t getConstantSize() const { return mConstantMemory.size; }
  uint32_t getVolatileSize() const { return mVolatileMemory.size; }

  // Converts a given relative (constant or volatile) pointer to an absolute
  // pointer without checking if the given offset is inside the range of that
  // memory
  const void* constantToAbsolute(uint32_t offset) const;
  void* volatileToAbsolute(uint32_t offset) const;

  // Converts an absolute pointer to a relative (constant or volatile) pointer
  // without checking if the address is inside the range
  uint32_t absoluteToConstant(const void* address) const;
  uint32_t absoluteToVolatile(const void* address) const;

  // Checks if the given absolute pointer points inside the constant or inside
  // the volatile memory
  bool isConstantAddress(const void* address) const;
  bool isVolatileAddress(const void* address) const;

  // Checks if the given absolute pointer points inside the constant or inside
  // the volatile memory. Check that address+size is also in the same range.
  bool isConstantAddressWithSize(const void* address, size_t size) const;
  bool isVolatileAddressWithSize(const void* address, size_t size) const;

  // Returns true if address is marker absolute address value which is used
  // to indicate a value which should not be observed.
  bool isNotObservedAbsoluteAddress(const void* address) const;

 private:
  // Struct to represent a memory interval inside the memory manager with its
  // base address and its size
  template <typename T = uint8_t>
  struct MemoryRange {
    MemoryRange();
    MemoryRange(T* base, uint32_t size);
    T* end() const { return base + size; }

    T* toAbsolute(uint32_t offset) const { return base + offset; }

    bool isInRange(const void* address) const {
      return address >= base && address < base + size;
    }

    bool isInRangeWithSize(const void* address, size_t s) const {
      const T* addr = static_cast<const T*>(address);
      return address >= base && addr + s <= base + size;
    }

    uint32_t toOffset(const void* address) const {
      const T* addr = static_cast<const T*>(address);
      return static_cast<uint32_t>(addr - base);
    }

    T* base;
    uint32_t size;
  };

  // Alignment used for each memory region in bytes
  static const uint32_t kAlignment = std::alignment_of<double>::value;

  // Align a given pointer based on the 'kAlignment' value. The return address
  // is always smaller or equal to the given address what is required by the
  // memory layout used
  uint8_t* align(uint8_t* addr) const;

  // The memory allocator from which to draw a static allocation for volatile
  // storage.
  std::shared_ptr<MemoryAllocator> mAllocator;

  // The base address of the memory block managed by the memory
  // manager. This pointer owns the allocated memory
  MemoryAllocator::Handle mMemory;

  // The size and base address of the opcode memory. This opcode memory
  // is passed to us, and is expected to live inside the payload itself.
  MemoryRange<const uint8_t> mOpcodeMemory;

  // The size and base address of the constant memory. The constant memory
  // is located inside the payload.
  MemoryRange<const uint8_t> mConstantMemory;

  // The size and the base address of the volatile memory. The memory range
  // specified by these values have to specify a subset of the memory managed by
  // the memory manager.
  MemoryRange<> mVolatileMemory;
};

inline const void* MemoryManager::constantToAbsolute(uint32_t offset) const {
  return mConstantMemory.toAbsolute(offset);
}

inline void* MemoryManager::volatileToAbsolute(uint32_t offset) const {
  return mVolatileMemory.toAbsolute(offset);
}

inline uint32_t MemoryManager::absoluteToConstant(const void* address) const {
  return mConstantMemory.toOffset(address);
}

inline uint32_t MemoryManager::absoluteToVolatile(const void* address) const {
  return mVolatileMemory.toOffset(address);
}

inline bool MemoryManager::isConstantAddress(const void* address) const {
  return mConstantMemory.isInRange(address);
}

inline bool MemoryManager::isVolatileAddress(const void* address) const {
  return mVolatileMemory.isInRange(address);
}

inline bool MemoryManager::isConstantAddressWithSize(const void* address,
                                                     size_t size) const {
  return mConstantMemory.isInRangeWithSize(address, size);
}

inline bool MemoryManager::isVolatileAddressWithSize(const void* address,
                                                     size_t size) const {
  return mVolatileMemory.isInRangeWithSize(address, size);
}

inline bool MemoryManager::isNotObservedAbsoluteAddress(
    const void* address) const {
  // Pointer is not observed. This can be legal - for example
  // glVertexAttribPointer may have been passed a pointer that was never
  // observed. In this situation we pass a pointer that should cause an access
  // violation if it is dereferenced. We opt to not use 0x00 as this is often
  // overloaded to mean something else.
  // Must match value used in replay/builder/builder.go
  return address == reinterpret_cast<const void*>(uintptr_t(0xBADF00D));
}

}  // namespace gapir

#endif  // GAPIR_MEMORY_MANAGER_H
