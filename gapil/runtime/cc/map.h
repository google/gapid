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

#ifndef __GAPIL_RUNTIME_MAP_H__
#define __GAPIL_RUNTIME_MAP_H__

#include "runtime.h"

#include <tuple>

namespace gapil {

template<typename K, typename V>
class Map : protected map_t {
    struct element;
public:
    using key_type = K;
    using value_type = V;

    Map();

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
                if (elem->used == mapElementFull) {
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
                if (elem->used == mapElementFull) {
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
            if (it.elem->used == mapElementFull) {
                break;
            }
            it.elem++;
        }
        return it;
    }

    iterator begin() {
        auto it = iterator{elements(), this};
        for (size_t i = 0; i < map_t::capacity; ++i) {
            if (it.elem->used == mapElementFull) {
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

    void erase(context_t* ctx, const K& k) {
        remove(ctx, k);
    }

    void erase(context_t* ctx, const_iterator it) {
        remove(ctx, it->first);
    }

    template<typename T>
    V& operator[](const typename std::pair<context_t*, T>& p) {
        V* v = index(p.first, p.second, true);
        return *v;
    }

    iterator find(context_t* ctx, const K& k) {
        V* idx = index(ctx, k, false);
        if (idx == nullptr) {
            return end();
        }
        size_t offs =
            (reinterpret_cast<uintptr_t>(idx) - reinterpret_cast<uintptr_t>(elements())) / sizeof(element);
        return iterator {elements() + offs, this};
    }

    const_iterator find(context_t* ctx, const K& k) const {
        // Sorry for the const_cast. We know that if the last element is false,
        // this wont be modified.
        const V* idx = const_cast<Map<K, V>*>(this)->index(ctx, k, false);
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

    bool contains(context_t*, K);
    V*   index(context_t*, K, bool);
    V    lookup(context_t*, K);
    void remove(context_t*, K);
    void clear(context_t*);
};

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_MAP_H__