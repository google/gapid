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

#ifndef GAPII_POOL_H
#define GAPII_POOL_H

#include <memory>
#include <stdint.h>

namespace gapii {

class Pool {
public:
    static std::shared_ptr<Pool> create(uint32_t id, uint64_t size);
// This creates a pool that can be serialized, but has no actual
// backing memory
    static std::shared_ptr<Pool> create_virtual(uint32_t id, uint64_t size);

    ~Pool();

    // id returns the ID this pool.
    inline uint32_t id() const { return mId; }

    // size returns the size of this pool in bytes.
    inline uint64_t size() const { return mSize; }

    // Pointer to first byte in the pool.
    inline void* base() const { return mData; }

    const bool is_virtual() const { return mIsVirtual; }
private:
    struct virtual_pool {};
    Pool(virtual_pool, uint32_t id, uint64_t size);
    Pool(uint32_t id, uint64_t size);
    Pool(const Pool&) = delete;
    Pool& operator=(const Pool&) = delete;

    uint32_t mId;
    void*    mData;
    uint64_t mSize;
    bool mIsVirtual;
};
}  // namespace gapii

#endif // GAPII_POOL_H
