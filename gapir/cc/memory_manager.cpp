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

#include "memory_manager.h"

#include "core/cc/log.h"

#include <new>
#include <utility>
#include <vector>

//                 mMemory[0]
//   ┏━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━┓
//   ┃             ┃                        ┃
//   ┃             ┃                        ┃
//   ┃             ┃                        ┃
//   ┃             ┃         volatile       ┃
//   ┃   mMemory   ┃          memory        ┃
//   ┃             ┃                        ┃
//   ┃             ┃                        ┨
//   ┃             ┣━━━━━━━━━━━━━━━━━━━━━━━━┨
//   ┃             ┃                        ┃
//   ┃             ┃        in-memory       ┃
//   ┃             ┃        resource        ┃
//   ┃             ┃          cache         ┃
//   ┃             ┃                        ┃
//   ┗━━━━━━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━━━━━┛
//                 mMemory[mSize]

namespace gapir {

namespace {
// Expected driver memory overhead to be left free as a factor of allocated
// managed memory.
const float kDriverOverheadFactor = 0.3f;
}  // namespace

template <typename T>
MemoryManager::MemoryRange<T>::MemoryRange() : base(nullptr), size(0) {}

template <typename T>
MemoryManager::MemoryRange<T>::MemoryRange(T* base, uint32_t size)
    : base(base), size(size) {}

MemoryManager::MemoryManager(const std::vector<uint32_t>& sizeList)
    : mConstantMemory(nullptr, 0) {
  for (auto size : sizeList) {
    // Try over-allocating to leave at least (size * kDriverOverheadFactor) free
    // bytes.
    mSize = static_cast<uint32_t>(size * (1 + kDriverOverheadFactor));
    mMemory.reset(new (std::nothrow) uint8_t[mSize]);
    if (mMemory) {
      // Free the over-allocation first, then attempt allocating the (smaller)
      // original size.
      mMemory.reset();
      mSize = size;
      mMemory.reset(new (std::nothrow) uint8_t[mSize]);
      break;
    }
    GAPID_DEBUG("Failed to allocate %u bytes of volatile memory, continuing...",
                size);
  }

  if (!mMemory) {
    GAPID_FATAL("Couldn't allocate any volatile memory size.");
  }

  GAPID_DEBUG("Base address: %p", mMemory.get());
  setVolatileMemory(mSize);
}

bool MemoryManager::setVolatileMemory(uint32_t size) {
  if (size > mSize) {
    return false;
  }

  mVolatileMemory = {&mMemory[0], size};
  GAPID_DEBUG("Volatile range: [%p,%p]", mVolatileMemory.base,
              mVolatileMemory.base + mVolatileMemory.size - 1);
  return true;
}

uint8_t* MemoryManager::align(uint8_t* addr) const {
  uintptr_t x = reinterpret_cast<uintptr_t>(addr);
  x -= x % MemoryManager::kAlignment;
  return reinterpret_cast<uint8_t*>(x);
}

void MemoryManager::setReplayData(const uint8_t* constantMemoryBase,
                                  uint32_t constantMemorySize,
                                  const uint8_t* opcodeMemoryBase,
                                  uint32_t opcodeMemorySize) {
  mConstantMemory =
      MemoryRange<const uint8_t>{constantMemoryBase, constantMemorySize};
  mOpcodeMemory =
      MemoryRange<const uint8_t>{opcodeMemoryBase, opcodeMemorySize};
}

}  // namespace gapir
