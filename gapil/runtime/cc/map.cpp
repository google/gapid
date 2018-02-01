// Copyright (C) 2017 Google Inc.
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

#include "map.h"
#include <unordered_map>

extern "C" {

void* gapil_remap_pointer(context* ctx, uint64_t pointer, uint64_t length) {
    return reinterpret_cast<void*>(pointer);
}

void gapil_get_code_location(context* ctx, char** file, uint32_t* line) {}

}  // extern "C"

namespace gapil {

template<typename K, typename V>
Map<K, V>::Map() {
    map_t::capacity = 0;
    map_t::count = 0;
    map_t::elements = nullptr;
    map_t::ref_count = 0;
}

template<typename K, typename V>
bool Map<K, V>::contains(context_t* ctx, K key) {
    return index(ctx, key, false) != nullptr;
}

template<typename K, typename V>
V* Map<K, V>::index(context_t* ctx, K key, bool insert) {
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
                map_t::elements = gapil_alloc(ctx, sizeof(element) * GAPIL_MIN_MAP_SIZE, alignof(V));
                for (uint64_t i = 0; i < map_t::capacity; ++i) {
                    elements()[i].used = GAPIL_MAP_ELEMENT_EMPTY;
                 }
            } else {
                 auto oldElements = elements();
                 auto oldCapacity = map_t::capacity;

                 map_t::capacity = map_t::capacity * GAPIL_MAP_GROW_MULTIPLIER;
                 map_t::elements = gapil_alloc(ctx, sizeof(element) * map_t::capacity, alignof(V));
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
                 gapil_free(ctx, oldElements);
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
V Map<K, V>::lookup(context_t* ctx, K key) {
    V* v = index(ctx, key, false);
    return *v;
}

template<typename K, typename V>
void Map<K, V>::remove(context_t*, K key) {
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
void Map<K, V>::clear(context_t* ctx) {
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
    gapil_free(ctx, map_t::elements);
    map_t::count = 0;
    map_t::capacity = 0;
    map_t::elements = nullptr;
}

}  // namespace gapil
