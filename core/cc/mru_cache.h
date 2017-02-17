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

#ifndef CORE_MRU_CACHE_H
#define CORE_MRU_CACHE_H

#include <unordered_map>

namespace core {

// MRUCache is an implementation of a most-recent-used key-value cache.
// All operations are O(1) complexity.
template <typename K, typename V>
class MRUCache {
public:
    typedef K key_type;
    typedef V value_type;

    // Constructs the cache with the specified maximum capacity.
    inline MRUCache(size_t capacity = 16);
    inline ~MRUCache();

    // adds the key-value pair into the cache and makes this the most recently
    // used entry. If the cache is already full before calling add() then the
    // least recently used entry is evicted from the cache.
    inline void add(const K& key, const V& value);

    // get looks up the entry for the specified key.
    // If an entry with the given key exists in the cache then its value is
    // assigned to value, the entry becomes the most recently used and true is
    // returned. If no entry with the given key exists in the cache then false
    // is returned.
    inline bool get(const K& key, V& value);

    // clear removes all items from the cache.
    inline void clear();

    // size returns the number of entries in the cache.
    inline size_t size() const;

    // capacity returns the maximum capacity for the cache.
    inline size_t capacity() const;

private:
    MRUCache(const MRUCache&) = delete;
    MRUCache& operator=(MRUCache const&) = delete;

    struct Item {
        inline Item();
        inline void unlink();
        inline void link(Item* prev);

        K     key;
        V     value;
        Item* prev;
        Item* next;
    };

    typedef std::unordered_map<K, Item*> Map;

    Map    mItems; // Dummy item used for head / tail of linked-list.
    Item   mList;
    size_t mCapacity;
};

template <typename K, typename V>
inline MRUCache<K, V>::Item::Item()
    : prev(nullptr), next(nullptr) {}

template <typename K, typename V>
inline void MRUCache<K, V>::Item::unlink() {
    if (prev != nullptr) {
        prev->next = next;
        next->prev = prev;
        next = prev = nullptr;
    }
}

template <typename K, typename V>
inline void MRUCache<K, V>::Item::link(Item* prev) {
    unlink();
    next = prev->next;
    next->prev = this;
    this->prev = prev;
    prev->next = this;
}

template <typename K, typename V>
inline MRUCache<K, V>::MRUCache(size_t capacity /* = 16 */)
        : mCapacity(capacity) {
    mList.next = mList.prev = &mList;
}

template <typename K, typename V>
inline MRUCache<K, V>::~MRUCache<K, V>() {
    clear();
}

template <typename K, typename V>
inline void MRUCache<K, V>::add(const K& key, const V& value) {
    auto it = mItems.find(key);
    if (it == mItems.end()) {
        Item* item;
        if (size() < capacity()) {
            item = new Item();
        } else {
            item = mList.prev;
            item->unlink();
            mItems.erase(item->key);
        }
        item->key = key;
        item->value = value;
        item->link(&mList);
        mItems[item->key] = item;
    } else {
        it->second->unlink();
        it->second->link(&mList);
        it->second->value = value;
    }
}

template <typename K, typename V>
inline bool MRUCache<K, V>::get(const K& key, V& value) {
    auto it = mItems.find(key);
    if (it == mItems.end()) {
        return false;
    }
    it->second->unlink();
    it->second->link(&mList);
    value = it->second->value;
    return true;
}

template <typename K, typename V>
inline void MRUCache<K, V>::clear() {
    for (auto it : mItems) {
        auto item = it.second;
        item->unlink();
        delete item;
    }
    mItems.clear();
}

template <typename K, typename V>
inline size_t MRUCache<K, V>::size() const {
    return mItems.size();
}

template <typename K, typename V>
inline size_t MRUCache<K, V>::capacity() const {
    return mCapacity;
}

}  // namespace core

#endif  // CORE_MRU_CACHE_H
