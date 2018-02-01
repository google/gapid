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

#ifndef __GAPIL_RUNTIME_MAP_H__
#define __GAPIL_RUNTIME_MAP_H__

#include "runtime.h"

namespace core {
class Arena;
}  // namespace core

namespace gapil {

template<typename K, typename V>
class Map : protected map_t {
    struct element;
public:
    using key_type = K;
    using value_type = V;

    static Map<K, V>* create(core::Arena* a);

    void reference();
    void release();

    class iterator {
        friend class Map<K, V>;
        element* elem;
        map_t* map;

        iterator(element* elem, map_t* map):
           elem(elem), map(map) {}

    public:
        iterator(const iterator& it):
            elem(it.elem), map(it.map) {
        }

        bool operator==(const iterator& other) {
            return map == other.map && elem == other.elem;
        }

        bool operator!=(const iterator& other){
            return !(*this == other);
        }

        element& operator*() {
            return *elem;
        }

        element* operator->() {
            return elem;
        }

        const iterator& operator++() {
            size_t offset = elem - reinterpret_cast<element*>(map->elements);
            for (size_t i = offset; i < map->capacity; ++i) {
                ++elem;
                if (elem->used == GAPIL_MAP_ELEMENT_FULL) {
                    break;
                }
            }
            return *this;
        }

        iterator operator++(int) {
            iterator ret = *this;
            ++(*this);
            return ret;
        }
    };

    class const_iterator {
        const element* elem;
        const map_t* map;

        const_iterator(const element* elem, const map_t* map):
           elem(elem), map(map) {}
        const_iterator(const iterator& it):
            elem(it.elem), map(it.map) {
        }

    public:
        bool operator==(const const_iterator& other) {
            return map == other.map && elem == other.elem;
        }

        bool operator!=(const const_iterator& other){
            return !(*this == other);
        }

        const element& operator*() {
            return *elem;
        }

        const element* operator->() {
            return elem;
        }

        const_iterator& operator++() {
            size_t offset = elem - reinterpret_cast<element*>(map->elements);
            for (size_t i = offset; i < map->capacity; ++i) {
                ++elem;
                if (elem->used == GAPIL_MAP_ELEMENT_FULL) {
                    break;
                }
            }
            return *this;
        }

        const_iterator operator++(int) {
            const_iterator ret = *this;
            ++(*this);
            return ret;
        }
    };

    uint64_t capacity() const {
        return map_t::capacity;
    }

    uint64_t count() const {
        return map_t::count;
    }

    const const_iterator begin() const {
        auto it = const_iterator{elements(), this};
        for (size_t i = 0; i < map_t::capacity; ++i) {
            if (it.elem->used == GAPIL_MAP_ELEMENT_FULL) {
                break;
            }
            it.elem++;
        }
        return it;
    }

    iterator begin() {
        auto it = iterator{elements(), this};
        for (size_t i = 0; i < map_t::capacity; ++i) {
            if (it.elem->used == GAPIL_MAP_ELEMENT_FULL) {
                break;
            }
            it.elem++;
        }
        return it;
    }

    iterator end() {
        return iterator{elements() + capacity(), this};
    }

    const_iterator end() const {
        return const_iterator{elements() + capacity(), this};
    }

    void erase(const K& k) {
        remove(k);
    }

    void erase(const_iterator it) {
        remove(it->first);
    }

    template<typename T>
    V& operator[](const T& key) {
        V* v = index(key, true);
        return *v;
    }

    iterator find(const K& key) {
        V* idx = index(key, false);
        if (idx == nullptr) {
            return end();
        }
        size_t offs =
            (reinterpret_cast<uintptr_t>(idx) - reinterpret_cast<uintptr_t>(elements())) / sizeof(element);
        return iterator {elements() + offs, this};
    }

    const_iterator find(const K& k) const {
        // Sorry for the const_cast. We know that if the last element is false,
        // this wont be modified.
        const V* idx = const_cast<Map<K, V>*>(this)->index(k, false);
        if (idx == nullptr) {
            return end();
        }
        size_t offs =
            (reinterpret_cast<uintptr_t>(idx) - reinterpret_cast<uintptr_t>(elements())) / sizeof(element);
        return const_iterator {elements() + offs, this};
    }

private:
    struct element {
        uint64_t used;
        K first;
        V second;
    };

    const element* elements() const {
        return reinterpret_cast<const element*>(map_t::elements);
    }

    element* elements() {
        return reinterpret_cast<element*>(map_t::elements);
    }

    bool contains(K);
    V*   index(K, bool);
    V    lookup(K);
    void remove(K);
    void clear();

    Map() = delete;
    Map(const Map<K,V>&) = delete;
    Map<K,V>& operator = (const Map<K,V>&) = delete;
};

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_MAP_H__