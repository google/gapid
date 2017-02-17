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
    static std::shared_ptr<Pool> create(uint64_t size);

    ~Pool();

    // size returns the size of this pool in bytes.
    inline uint64_t size() const;

    // Pointer to first byte in the pool.
    inline void* base() const;

private:
    Pool(uint64_t size);
    Pool(const Pool&) = delete;
    Pool& operator=(const Pool&) = delete;

    void*    mData;
    uint64_t mSize;
};

inline uint64_t Pool::size() const {
    return mSize;
}

inline void* Pool::base() const {
    return mData;
}

}  // namespace gapii

#endif // GAPII_POOL_H
