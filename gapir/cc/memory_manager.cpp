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
//   ┃             ┃        in-memory       ┃
//   ┃             ┃        resource        ┃
//   ┃             ┃          cache         ┃
//   ┃             ┣━━━━━━━━━━━━━━━━━━━━━━━━┨
//   ┃             ┃                        ┃
//   ┃             ┃         volatile       ┃
//   ┃   mMemory   ┃          memory        ┃
//   ┃             ┃                        ┃
//   ┃             ┣━━━━━━━━━━━━┳━━━━━━━━━━━┨
//   ┃             ┃            ┃           ┃
//   ┃             ┃   replay   ┃           ┃
//   ┃             ┃    data    ┣━━━━━━━━━━━┨
//   ┃             ┃            ┃  constant ┃
//   ┃             ┃            ┃   memory  ┃
//   ┗━━━━━━━━━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━┛
//                 mMemory[mSize]

namespace gapir {

namespace {
// Expected driver memory overhead to be left free as a factor of allocated managed memory.
const float kDriverOverheadFactor = 0.3f;
}

MemoryManager::MemoryRange::MemoryRange() : base(nullptr), size(0) {
}

MemoryManager::MemoryRange::MemoryRange(uint8_t* base, uint32_t size) : base(base), size(size) {
}

MemoryManager::MemoryManager(const std::vector<uint32_t>& sizeList) : mConstantMemory(nullptr, 0) {
    for (auto size : sizeList) {
        // Try over-allocating to leave at least (size * kDriverOverheadFactor) free bytes.
        mSize = static_cast<uint32_t>(size * (1 + kDriverOverheadFactor));
        mMemory.reset(new(std::nothrow) uint8_t[mSize]);
        if (mMemory) {
            // Free the over-allocation first, then attempt allocating the (smaller) original size.
            mMemory.reset();
            mSize = size;
            mMemory.reset(new(std::nothrow) uint8_t[mSize]);
            break;
        }
        GAPID_DEBUG("Failed to allocate %u bytes of volatile memory, continuing...", size);
    }

    if (!mMemory) {
        GAPID_FATAL("Couldn't allocate any volatile memory size.");
    }

    GAPID_DEBUG("Base address: %p", mMemory.get());
    setReplayDataSize(0);
    setVolatileMemory(mSize);
}

bool MemoryManager::setReplayDataSize(uint32_t size) {
    if (size > mSize) {
        return false;
    }

    mReplayData = {align(mMemory.get() + mSize - size), size};
    GAPID_DEBUG("Replay range: [%p,%p]",
        mReplayData.base, mReplayData.base + mReplayData.size - 1);
    return true;
}

bool MemoryManager::setVolatileMemory(uint32_t size) {
    if (size > mSize - mReplayData.size) {
        return false;
    }

    mVolatileMemory = {align(mReplayData.base - size), size};
    GAPID_DEBUG("Volatile range: [%p,%p]",
        mVolatileMemory.base, mVolatileMemory.base + mVolatileMemory.size - 1);
    return true;
}

void MemoryManager::setConstantMemory(const std::pair<const void*, uint32_t>& constantMemory) {
    mConstantMemory.base = const_cast<uint8_t*>(static_cast<const uint8_t*>(constantMemory.first));
    mConstantMemory.size = constantMemory.second;
    GAPID_DEBUG("Constant range: [%p,%p]",
        mConstantMemory.base, mConstantMemory.base + mConstantMemory.size - 1);
}

uint8_t* MemoryManager::align(uint8_t* addr) const {
    uintptr_t x = reinterpret_cast<uintptr_t>(addr);
    x -= x % MemoryManager::kAlignment;
    return reinterpret_cast<uint8_t*>(x);
}

}  // namespace gapir
