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

#include "resource_in_memory_cache.h"
#include "replay_connection.h"

#include "core/cc/assert.h"

#include <string.h>

#include <algorithm>
#include <string>
#include <utility>
#include <vector>

namespace gapir {

std::unique_ptr<ResourceInMemoryCache> ResourceInMemoryCache::create(
        std::unique_ptr<ResourceProvider> fallbackProvider, void* buffer) {
    return std::unique_ptr<ResourceInMemoryCache>(
            new ResourceInMemoryCache(std::move(fallbackProvider), buffer));
}

ResourceInMemoryCache::ResourceInMemoryCache(std::unique_ptr<ResourceProvider> fallbackProvider,
                                             void* buffer)
        : ResourceCache(std::move(fallbackProvider))
        , mHead(new Block(0, 0))
        , mBuffer(static_cast<uint8_t*>(buffer))
        , mBufferSize(0) {
}

ResourceInMemoryCache::~ResourceInMemoryCache() {
    while (mHead->next != mHead) {
        destroy(mHead->next);
    }
    delete mHead;
}

void ResourceInMemoryCache::prefetch(const Resource*         resources,
                                     size_t                  count,
                                     ReplayConnection*       conn,
                                     void*                   temp,
                                     size_t                  tempSize) {
    if (temp == nullptr) {
        return;
    }
    GAPID_DEBUG("ResourceInMemoryCache::prefetch(count: %d, mBufferSize: %d, tempSize: %d)", count, mBufferSize, tempSize);
    Batch batch(temp, tempSize);
    size_t space = mBufferSize;
    for (size_t i = 0; i < count; i++) {
        const Resource& resource = resources[i];
        if (space < resource.size) {
            break;
        }
        space -= resource.size;
        if (mCache.find(resource.id) != mCache.end()) {
            continue;
        }
        if (!batch.append(resource)) {
            batch.flush(*this, conn);
            batch = Batch(temp, tempSize);
        }
    }
    batch.flush(*this, conn);
}

void ResourceInMemoryCache::clear() {
    mCache.clear();
    while (mHead->next != mHead) {
        mHead = destroy(mHead);
    }
    *mHead = Block(0, mBufferSize);
    mHead->next = mHead->prev = mHead;
}

void ResourceInMemoryCache::resize(size_t newSize) {
    GAPID_DEBUG("Cache resizing: %d -> %d", mBufferSize, newSize);
    if (newSize == mBufferSize) {
        return; // No change.
    }

    Block* first = this->first();
    Block* last = this->last();

    // Remove all the blocks that are completely beyond the end of the new size.
    while (last != first && last->offset > newSize) {
        last = last->prev;
        destroy(last->next);
    }

    if (!last->isFree()) {
        if (last->end() > newSize) {
            // The last block wraps the buffer. We need to evict this as we've
            // changed the wrapping point.
            free(last);
        } else {
            // Buffer has grown. Add new space block.
            last = new Block(last->end(), 0);
            last->linkBefore(first);
        }
    }
    // Whether we've grown or shrunk, the last block will always be free.
    // Re-adjust the size so that it touches the first block.
    last->size = (newSize - last->offset) + first->offset;

    // If there's only one block remaining, it's free. Make sure it starts at 0.
    if (last == first) {
        last->offset = 0;
    }

    mHead = last; // Move head to the space.
    mBufferSize = newSize;
}

void ResourceInMemoryCache::dump(FILE* out) {
    Block* first = last()->next;
    foreach_block(first, [&](Block* block) { fprintf(out, (block == first) ? "┏━━━━━━━━━━━━━━━━" : "┳━━━━━━━━━━━━━━━━"); });
    fprintf(out, "┓\n");
    foreach_block(first, [&](Block* block) { fprintf(out, "┃ offset: %6zu ", block->offset); });
    fprintf(out, "┃\n");
    foreach_block(first, [&](Block* block) { fprintf(out, "┃ size:   %6zu ", block->size); });
    fprintf(out, "┃\n");
    foreach_block(first, [&](Block* block) {
        if (block->isFree()) {
            fprintf(out, "┃ free           ");
        } else {
            fprintf(out, "┃ id: %10.10s ", block->id.c_str());
        }
    });
    fprintf(out, "┃\n");
    foreach_block(first, [&](Block* block) { fprintf(out, (block == mHead) ? "┃ head           " : "┃                "); });
    fprintf(out, "┃\n");
    foreach_block(first, [&](Block* block) { fprintf(out, (block == first) ? "┗━━━━━━━━━━━━━━━━" : "┻━━━━━━━━━━━━━━━━"); });
    fprintf(out, "┛\n");
}

void ResourceInMemoryCache::putCache(const Resource& resource, const void* data) {
    if (resource.size > mBufferSize) {
        return; // Wouldn't fit even if everything was evicted.
    }

    // Merge mHead into next block(s) until it is big enough to hold our resource.
    while (mHead->size < resource.size) {
        mHead->size += mHead->next->size;
        destroy(mHead->next);
    }

    if (mHead->size > resource.size) {
        // We've got some left-over space in this block. Split it.
        size_t space = mHead->size - resource.size;
        size_t offset = (mHead->offset + resource.size) % mBufferSize;
        auto next = new Block(offset, space);
        next->linkAfter(mHead);
        mHead->size = resource.size;
    }

    // Update mCache.
    mCache.erase(mHead->id);
    mCache.emplace(resource.id, mHead->offset);
    mHead->id = resource.id;

    // Copy data.
    if (mHead->offset + resource.size <= mBufferSize) {
        memcpy(mBuffer + mHead->offset, data, resource.size);
    } else {
        // Wraps the end of the buffer
        const uint8_t* dst = reinterpret_cast<const uint8_t*>(data);
        size_t a = mBufferSize - mHead->offset;
        size_t b = resource.size - a;
        memcpy(mBuffer + mHead->offset, dst, a);
        memcpy(mBuffer, dst + a, b);
    }

    // Move head on to the next block.
    mHead = mHead->next;
}

bool ResourceInMemoryCache::getCache(const Resource& resource, void* data) {
    auto iter = mCache.find(resource.id);
    if (iter == mCache.end()) {
        return false;
    }
    // Cached resource found. Copy data.
    size_t offset = iter->second;
    if (offset + resource.size <= mBufferSize) {
        memcpy(data, mBuffer + offset, resource.size);
    } else {
        // Wraps the end of the buffer
        uint8_t* dst = reinterpret_cast<uint8_t*>(data);
        size_t a = mBufferSize - offset;
        size_t b = resource.size - a;
        memcpy(dst, mBuffer + offset, a);
        memcpy(dst + a, mBuffer, b);
    }
    return true;
}

}  // namespace gapir
