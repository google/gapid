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


#include "map.inc"

#include "core/cc/assert.h"
#include "core/memory/arena/cc/arena.h"

#include <gtest/gtest.h>

extern "C" {

void* gapil_remap_pointer(context* ctx, uint64_t pointer, uint64_t length) {
    return reinterpret_cast<void*>(pointer);
}

void gapil_get_code_location(context* ctx, char** file, uint32_t* line) {}

}  // extern "C"

template<typename T>
class MapTest : public ::testing::Test {
    void TearDown() {
        size_t num_allocations = 0;
        size_t num_bytes_allocated = 0;
        arena.stats(&num_allocations, &num_bytes_allocated);

        EXPECT_EQ(0, num_allocations);
        EXPECT_EQ(0, num_bytes_allocated);
    }

public:
    size_t num_allocations() const {
        size_t num_allocations = 0;
        size_t num_bytes_allocated = 0;
        arena.stats(&num_allocations, &num_bytes_allocated);
        return num_allocations;
    }

    size_t num_bytes_allocated() const {
        size_t num_allocations = 0;
        size_t num_bytes_allocated = 0;
        arena.stats(&num_allocations, &num_bytes_allocated);
        return num_bytes_allocated;
    }

    core::Arena arena;
};

using MapTestTypes = ::testing::Types<
    gapil::Map<uint32_t, uint32_t>,
    gapil::Map<uint16_t, uint32_t>,
    gapil::Map<uint32_t, uint64_t>>;

TYPED_TEST_CASE(MapTest, MapTestTypes);

TYPED_TEST(MapTest, basic_insert) {
    using key_type = typename TypeParam::key_type;
    using value_type = typename TypeParam::value_type;

    auto map = TypeParam::create(&this->TestFixture::arena);

    (*map)[key_type(32)] = value_type(42);
    EXPECT_EQ(42, (*map)[key_type(32)]);
    EXPECT_EQ(0, (*map)[key_type(42)]);
    EXPECT_EQ(2, map->count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map->capacity());

    map->release();
}

TYPED_TEST(MapTest, multi_insert) {
    using key_type = typename TypeParam::key_type;
    using value_type = typename TypeParam::value_type;

    auto map = TypeParam::create(&this->TestFixture::arena);

    uint64_t resize_threshold = static_cast<uint64_t>(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_MAX_CAPACITY);
    for (uint64_t i = 0; i <= resize_threshold; ++i) {
        (*map)[key_type(i)] = value_type(i);
    }
    EXPECT_EQ(resize_threshold+1, map->count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map->capacity());


    for (uint64_t i = 0; i < resize_threshold + 1; ++i) {
        EXPECT_EQ(value_type(i), (*map)[key_type(i)]);
    }

    (*map)[key_type(resize_threshold + 1)] = value_type(resize_threshold + 1);
    EXPECT_EQ(resize_threshold + 2, map->count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_GROW_MULTIPLIER, map->capacity());

    for (uint64_t i = 0; i < resize_threshold + 1; ++i) {
        EXPECT_EQ(value_type(i), (*map)[key_type(i)]);
    }

    map->release();
}

TYPED_TEST(MapTest, erase) {
    using key_type = typename TypeParam::key_type;
    using value_type = typename TypeParam::value_type;
    auto map = TypeParam::create(&this->TestFixture::arena);

    uint64_t resize_threshold = static_cast<uint64_t>(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_MAX_CAPACITY);
    for (uint64_t i = 0; i <= resize_threshold; ++i) {
        (*map)[key_type(i)] = value_type(i);
    }
    EXPECT_EQ(resize_threshold+1, map->count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map->capacity());


    for (uint64_t i = 0; i < resize_threshold + 1; ++i) {
        EXPECT_EQ(value_type(i), (*map)[key_type(i)]);
    }

    (*map)[key_type(resize_threshold + 1)] = value_type(resize_threshold + 1);
    EXPECT_EQ(resize_threshold + 2, map->count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_GROW_MULTIPLIER, map->capacity());

    for (uint64_t i = 0; i < resize_threshold + 1; ++i) {
        EXPECT_EQ(value_type(i), (*map)[key_type(i)]);
    }

    map->erase(key_type(10));
    EXPECT_EQ(0, (*map)[key_type(10)]);

    map->release();
}

TYPED_TEST(MapTest, range) {
    using key_type = typename TypeParam::key_type;
    using value_type = typename TypeParam::value_type;

    auto map = TypeParam::create(&this->TestFixture::arena);

    std::vector<value_type> result_vector;
    result_vector.resize(16);

    for (uint64_t i = 0; i < 16; ++i) {
        (*map)[key_type(i)] = value_type(i);
    }

    uint64_t old_allocations = TestFixture::num_allocations();

    for (auto& val: *map) {
        result_vector[val.first] = val.second;
    }

    // Ranging over a map should not have caused any allocations.
    EXPECT_EQ(old_allocations, TestFixture::num_allocations());

    for (uint64_t i = 0; i < 16; ++i) {
        EXPECT_EQ(result_vector[i], value_type(i));
    }

    map->release();
}

class non_movable_object {
public:
    non_movable_object() {
        size = 0;
        v = nullptr;
        arena = nullptr;
    }

    non_movable_object(arena_t* a, uint64_t i) {
        size = i;
        v = gapil_alloc(a, i, 1);
        arena = a;
    }

    non_movable_object& operator=(const non_movable_object& _other) {
        size = _other.size;
        arena = _other.arena;
        v = gapil_alloc(arena, size, 1);
        return *this;
    }

    ~non_movable_object() {
        if (arena != nullptr) {
            gapil_free(arena, v);
        }
    }
    non_movable_object(const non_movable_object& _other) {
        v = gapil_alloc(_other.arena, _other.size, 1);
    }

private:
    void * v;
    uint64_t size;
    arena_t* arena;
};

TEST(CppMapTest, non_movable_object) {
    core::Arena arena;
    auto map = gapil::Map<uint32_t, non_movable_object>::create(&arena);

    auto a = reinterpret_cast<arena_t*>(&arena);

    uint64_t resize_threshold = static_cast<uint64_t>(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_MAX_CAPACITY);
    for (uint64_t i = 0; i <= resize_threshold; ++i) {
        (*map)[i] = non_movable_object(a, i + 10);
    }
    EXPECT_EQ(resize_threshold+1, map->count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map->capacity());

    (*map)[resize_threshold + 1] = non_movable_object(a, 10 + resize_threshold + 1);
    EXPECT_EQ(resize_threshold + 2, map->count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_GROW_MULTIPLIER, map->capacity());

    map->release();
}

class movable_object {
public:
    movable_object() {
        size = 0;
        v = nullptr;
        arena = nullptr;
    }

    movable_object(arena_t* a, uint64_t i) {
        v = gapil_alloc(a, i, 1);
        size = i;
        arena = a;
    }

    movable_object& operator=(movable_object&& _other) {
        v = _other.v;
        size = _other.size;
        arena = _other.arena;
        _other.v = nullptr;
        _other.size = 0;
        _other.arena = nullptr;
        return *this;
    }

    ~movable_object() {
        if (arena != nullptr) {
            gapil_free(arena, v);
        }
    }
    movable_object(movable_object&& _other) {
        *this = std::move(_other);
    }

private:
    void * v;
    uint64_t size;
    arena_t* arena;
};

TEST(CppMapTest, movable_object) {
    core::Arena arena;
    auto map = gapil::Map<uint32_t, movable_object>::create(&arena);

    auto a = reinterpret_cast<arena_t*>(&arena);

    uint64_t resize_threshold = static_cast<uint64_t>(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_MAX_CAPACITY);
    for (uint64_t i = 0; i <= resize_threshold; ++i) {
        (*map)[i] = movable_object(a, i + 10);
    }
    EXPECT_EQ(resize_threshold+1, map->count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map->capacity());

    (*map)[resize_threshold + 1] = movable_object(a, 10 + resize_threshold + 1);
    EXPECT_EQ(resize_threshold + 2, map->count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_GROW_MULTIPLIER, map->capacity());

    map->release();
}
