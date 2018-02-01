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
Map<K, V>* Map<K, V>::create(core::Arena* a) {
    GAPID_ASSERT_MSG(a != nullptr, "arena must not be null");
    auto buf = a->allocate(sizeof(Map<K, V>), alignof(Map<K, V>));
    auto map = reinterpret_cast<Map<K, V>*>(buf);
    map->map_t::arena = reinterpret_cast<arena_t*>(a);
    map->map_t::capacity = 0;
    map->map_t::count = 0;
    map->map_t::elements = nullptr;
    map->map_t::ref_count = 1;
    return map;
}

template<typename K, typename V>
inline void Map<K, V>::reference() {
    GAPID_ASSERT_MSG(map_t::ref_count > 0, "Attempting to reference deleted map");
    map_t::ref_count++;
}

template<typename K, typename V>
inline void Map<K, V>::release() {
    GAPID_ASSERT_MSG(map_t::ref_count > 0, "Attempting to release deleted map");
    map_t::ref_count--;
    if (map_t::ref_count > 0) {
        return;
    }
    auto arena = reinterpret_cast<core::Arena*>(map_t::arena);
    if (map_t::elements != nullptr) {
        arena->free(map_t::elements);
    }
    arena->free(this);
}

template<typename K, typename V>
bool Map<K, V>::contains(K key) {
    return index(key, false) != nullptr;
}

template<typename K, typename V>
V* Map<K, V>::index(K key, bool insert) {
    auto hasher = std::hash<K>{};
    auto eq = std::equal_to<K>{};
    uint64_t hash = hasher(key);

    auto elems = elements();

    for (uint64_t i = 0; i < map_t::capacity; ++i) {
        bool leave = false;
        uint64_t lookup_pos = (hash + i) % map_t::capacity;
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
        auto elems = elements();
        for (uint64_t i = 0; i < map_t::capacity; ++i) {
            uint64_t x = (h + i) %  map_t::capacity;
            if (elems[x].used != GAPIL_MAP_ELEMENT_FULL) {
                return x;
            }
        }
        return uint64_t(0);
    };

    if (insert) {
        bool resize = (map_t::elements == nullptr);
        resize = resize || ((float)map_t::count / (float)map_t::capacity) > GAPIL_MAP_MAX_CAPACITY;

        if (resize) {
            if (map_t::elements == nullptr) {
                map_t::capacity = GAPIL_MIN_MAP_SIZE;
                map_t::elements = gapil_alloc(map_t::arena, sizeof(element) * GAPIL_MIN_MAP_SIZE, alignof(V));
                for (uint64_t i = 0; i < map_t::capacity; ++i) {
                    elements()[i].used = GAPIL_MAP_ELEMENT_EMPTY;
                 }
            } else {
                 auto oldElements = elements();
                 auto oldCapacity = map_t::capacity;

                 map_t::capacity = map_t::capacity * GAPIL_MAP_GROW_MULTIPLIER;
                 map_t::elements = gapil_alloc(map_t::arena, sizeof(element) * map_t::capacity, alignof(V));
                 for (uint64_t i = 0; i < map_t::capacity; ++i) {
                    elements()[i].used = GAPIL_MAP_ELEMENT_EMPTY;
                 }
                 auto new_elements = elements();
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
                 gapil_free(map_t::arena, oldElements);
             }
        }

        uint64_t bucket_location = storageBucket(hasher(key));
        new(&elements()[bucket_location].second) V();
        new(&elements()[bucket_location].first) K(key);
        elements()[bucket_location].used = GAPIL_MAP_ELEMENT_FULL;
        map_t::count++;

        return &elements()[bucket_location].second;
    }

    return nullptr;
}

template<typename K, typename V>
V Map<K, V>::lookup(K key) {
    V* v = index(key, false);
    return *v;
}

template<typename K, typename V>
void Map<K, V>::remove(K key) {
    auto hasher = std::hash<K>{};
    auto eq = std::equal_to<K>{};
    uint64_t hash = hasher(key);
    auto elems = elements();

    for (uint64_t i = 0; i < map_t::capacity; ++i) {
        uint64_t lookup_pos = (hash + i) % map_t::capacity;
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
                    --map_t::count;
                    return;
                }
        }
    }
}

template<typename K, typename V>
void Map<K, V>::clear() {
    auto elems = elements();
    for (uint64_t i = 0; i < map_t::capacity; ++i) {
        switch(elems[i].used) {
            case GAPIL_MAP_ELEMENT_EMPTY:
            case GAPIL_MAP_ELEMENT_USED:
                continue;
            case GAPIL_MAP_ELEMENT_FULL:
                elems[i].first.~K();
                elems[i].second.~V();
                --map_t::count;
        }
    }
    gapil_free(map_t::arena, map_t::elements);
    map_t::count = 0;
    map_t::capacity = 0;
    map_t::elements = nullptr;
}

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_MAP_INC__
