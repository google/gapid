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

#ifndef __GAPIL_RUNTIME_MAP_INC__
#define __GAPIL_RUNTIME_MAP_INC__

#include "map.h"

#include "core/cc/assert.h"
#include "core/memory/arena/cc/arena.h"

#include <unordered_map>

namespace gapil {

template<typename K, typename V>
Map<K, V>::Map(core::Arena* a) {
    GAPID_ASSERT_MSG(a != nullptr, "arena must not be null");
    ptr = a->create<Map<K, V>::Allocation>();
    ptr->arena = reinterpret_cast<arena_t*>(a);
    ptr->capacity = 0;
    ptr->count = 0;
    ptr->elements = nullptr;
    ptr->ref_count = 1;
}

template<typename K, typename V>
Map<K, V>::Map(const Map<K, V>& s) {
    ptr = s.ptr;
    ptr->ref_count++;
}

template<typename K, typename V>
Map<K, V>::Map(Map<K, V>&& s) {
    ptr = s.ptr;
    s.ptr = nullptr;
}

template<typename K, typename V>
Map<K, V>::~Map() {
    if (ptr != nullptr) { // note: nullptr is only valid in the case of a move
        ptr->release();
    }
}

template<typename K, typename V>
Map<K, V>& Map<K, V>::operator = (const Map<K, V>& other) {
    GAPID_ASSERT_MSG(other.ptr->ref_count > 0, "attempting to reference freed map");
    if (ptr != other.ptr) {
        ptr->release();
        ptr = other.ptr;
        ptr->reference();
    }
    return *this;
}

template<typename K, typename V>
inline void Map<K, V>::Allocation::reference() {
    GAPID_ASSERT_MSG(ref_count > 0, "Attempting to reference deleted map");
    ref_count++;
}

template<typename K, typename V>
inline void Map<K, V>::Allocation::release() {
    GAPID_ASSERT_MSG(ref_count > 0, "Attempting to release deleted map");
    ref_count--;
    if (ref_count > 0) {
        return;
    }
    auto arena = reinterpret_cast<core::Arena*>(this->arena);
    if (elements != nullptr) {
        arena->free(elements);
    }
    arena->free(this);
}

template<typename K, typename V>
bool Map<K, V>::Allocation::contains(K key) {
    return index(key, false) != nullptr;
}

template<typename K, typename V>
V* Map<K, V>::Allocation::index(K key, bool insert) {
    auto hasher = std::hash<K>{};
    auto eq = std::equal_to<K>{};
    uint64_t hash = hasher(key);

    auto elems = els();

    for (uint64_t i = 0; i < capacity; ++i) {
        bool leave = false;
        uint64_t lookup_pos = (hash + i) % capacity;
        switch(elems[lookup_pos].used) {
            case GAPIL_MAP_ELEMENT_EMPTY:
                leave = true;
                break;
            case GAPIL_MAP_ELEMENT_USED:
                continue;
            case GAPIL_MAP_ELEMENT_FULL:
                if (eq(key, elems[lookup_pos].first)) {
                    return &elems[lookup_pos].second;
                }
        }
        if (leave) {
            break;
        }
    }

    // storageBucket assumes there is at least one open cell.
    // Make sure before you run this, that is the case.
    auto storageBucket = [&](uint64_t h) {
        auto elems = els();
        for (uint64_t i = 0; i < capacity; ++i) {
            uint64_t x = (h + i) %  capacity;
            if (elems[x].used != GAPIL_MAP_ELEMENT_FULL) {
                return x;
            }
        }
        return uint64_t(0);
    };

    if (insert) {
        bool resize = (elements == nullptr);
        resize = resize || ((float)count / (float)capacity) > GAPIL_MAP_MAX_CAPACITY;

        if (resize) {
            if (elements == nullptr) {
                capacity = GAPIL_MIN_MAP_SIZE;
                elements = gapil_alloc(arena, sizeof(element) * GAPIL_MIN_MAP_SIZE, alignof(V));
                for (uint64_t i = 0; i < capacity; ++i) {
                    this->els()[i].used = GAPIL_MAP_ELEMENT_EMPTY;
                 }
            } else {
                 auto oldElements = this->els();
                 auto oldCapacity = capacity;

                 capacity = capacity * GAPIL_MAP_GROW_MULTIPLIER;
                 elements = gapil_alloc(arena, sizeof(element) * capacity, alignof(V));
                 for (uint64_t i = 0; i < capacity; ++i) {
                    els()[i].used = GAPIL_MAP_ELEMENT_EMPTY;
                 }
                 auto new_elements = els();
                 for (uint64_t i = 0; i < oldCapacity; ++i) {
                     if (oldElements[i].used == GAPIL_MAP_ELEMENT_FULL) {
                        uint64_t bucket_location = storageBucket(hasher(oldElements[i].first));
                        new(&new_elements[bucket_location].second) V(std::move(oldElements[i].second));
                        new(&new_elements[bucket_location].first) K(std::move(oldElements[i].first));
                        new_elements[bucket_location].used = GAPIL_MAP_ELEMENT_FULL;
                        oldElements[i].second.~V();
                        oldElements[i].first.~K();
                     }
                 }
                 gapil_free(arena, oldElements);
             }
        }

        uint64_t bucket_location = storageBucket(hasher(key));
        new(&els()[bucket_location].second) V();
        new(&els()[bucket_location].first) K(key);
        els()[bucket_location].used = GAPIL_MAP_ELEMENT_FULL;
        count++;

        return &els()[bucket_location].second;
    }

    return nullptr;
}

template<typename K, typename V>
V Map<K, V>::Allocation::lookup(K key) {
    V* v = index(key, false);
    return *v;
}

template<typename K, typename V>
void Map<K, V>::Allocation::remove(K key) {
    auto hasher = std::hash<K>{};
    auto eq = std::equal_to<K>{};
    uint64_t hash = hasher(key);
    auto elems = els();

    for (uint64_t i = 0; i < capacity; ++i) {
        uint64_t lookup_pos = (hash + i) % capacity;
        switch(elems[lookup_pos].used) {
            case GAPIL_MAP_ELEMENT_EMPTY:
                return;
            case GAPIL_MAP_ELEMENT_USED:
                continue;
            case GAPIL_MAP_ELEMENT_FULL:
                if (eq(key, elems[lookup_pos].first)) {
                    elems[lookup_pos].used = GAPIL_MAP_ELEMENT_USED;
                    elems[lookup_pos].first.~K();
                    elems[lookup_pos].second.~V();
                    --count;
                    return;
                }
        }
    }
}

template<typename K, typename V>
void Map<K, V>::Allocation::clear() {
    auto elems = els();
    for (uint64_t i = 0; i < capacity; ++i) {
        switch(elems[i].used) {
            case GAPIL_MAP_ELEMENT_EMPTY:
            case GAPIL_MAP_ELEMENT_USED:
                continue;
            case GAPIL_MAP_ELEMENT_FULL:
                elems[i].first.~K();
                elems[i].second.~V();
                --count;
        }
    }
    gapil_free(arena, elements);
    count = 0;
    capacity = 0;
    elements = nullptr;
}

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_MAP_INC__
