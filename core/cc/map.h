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

#ifndef CORE_MAP_H
#define CORE_MAP_H

#include "vector.h"

namespace core {

// Map is a fixed-capacity container of plain-old-data elements that map from K to V.
// Do not use elements that require destruction in this container.
// The key type K must be equality comparable using the == operator, and both the key type K and
// the value type V must be assignable using the = operator.
template<typename K, typename V>
class Map {
public:
    struct Entry {
        K key;
        V value;
    };

    // Default constructor.
    // Map is unusable until it is assigned from a Map constructed using one of the other
    // constructors.
    Map();

    // Constructs a map using the specified address for storage.
    // The number of entries in the map cannot exceed capacity.
    Map(Entry* first, size_t capacity);

    // clear sets the map count to 0.
    inline void clear();

    // set inserts the key-value pair into the map, replacing any existing entry with the same
    // value.
    // It is a fatal error if the map has no more capacity.
    inline void set(const K& key, const V& value);

    // count returns the number of elements in the map.
    inline size_t count() const;

    // Support for range-based for looping
    inline Entry* begin() const;
    inline Entry* end() const;

private:
    Vector<Entry> mEntries;
};

template<typename K, typename V>
Map<K, V>::Map() {}

template<typename K, typename V>
Map<K, V>::Map(Entry* first, size_t capacity)
        : mEntries(first, 0, capacity) {}

template<typename K, typename V>
inline void Map<K, V>::clear() {
    mEntries.clear();
}

template<typename K, typename V>
inline void Map<K, V>::set(const K& key, const V& value) {
    for (size_t i = 0; i < mEntries.count(); i++) {
        if (mEntries[i].key == key) {
            mEntries[i].value = value;
            return;
        }
    }
    mEntries.append(Entry{key, value});
}

template<typename K, typename V>
inline size_t Map<K, V>::count() const {
    return mEntries.count();
}

template<typename K, typename V>
inline typename Map<K, V>::Entry* Map<K, V>::begin() const {
    return mEntries.begin();
}

template<typename K, typename V>
inline typename Map<K, V>::Entry* Map<K, V>::end() const {
    return mEntries.end();
}

}  // namespace core

#endif  // CORE_MAP_H
