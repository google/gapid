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


#include "map.cpp"
#include "test_alloc_free.h"
#include <gtest/gtest.h>

template<typename T>
class MapTest : public ::testing::Test {
    void TearDown() {
        EXPECT_EQ(0, testing::allocated_bytes);
        EXPECT_EQ(0, testing::num_allocations);
    }
};

using MapTestTypes = ::testing::Types<
    gapil::Map<uint32_t, uint32_t>,
    gapil::Map<uint16_t, uint32_t>,
    gapil::Map<uint32_t, uint64_t>>;

TYPED_TEST_CASE(MapTest, MapTestTypes);

TYPED_TEST(MapTest, basic_insert) {
    using key_type = typename TypeParam::key_type;
    using value_type = typename TypeParam::value_type;
    TypeParam map;
    context_t* ctx = nullptr;

    map[std::make_pair(ctx, key_type(32))] = value_type(42);
    EXPECT_EQ(42, map[std::make_pair(ctx, key_type(32))]);
    EXPECT_EQ(0, map[std::make_pair(ctx, key_type(42))]);
    EXPECT_EQ(2, map.count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map.capacity());
}

TYPED_TEST(MapTest, multi_insert) {
    using key_type = typename TypeParam::key_type;
    using value_type = typename TypeParam::value_type;
    TypeParam map;
    context_t* ctx = nullptr;

    uint64_t resize_threshold = static_cast<uint64_t>(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_MAX_CAPACITY);
    for (uint64_t i = 0; i <= resize_threshold; ++i) {
        map[std::make_pair(ctx, key_type(i))] = value_type(i);
    }
    EXPECT_EQ(resize_threshold+1, map.count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map.capacity());


    for (uint64_t i = 0; i < resize_threshold + 1; ++i) {
        EXPECT_EQ(value_type(i), map[std::make_pair(ctx, key_type(i))]);
    }

    map[std::make_pair(ctx, key_type(resize_threshold + 1))] = value_type(resize_threshold + 1);
    EXPECT_EQ(resize_threshold + 2, map.count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_GROW_MULTIPLIER, map.capacity());

    for (uint64_t i = 0; i < resize_threshold + 1; ++i) {
        EXPECT_EQ(value_type(i), map[std::make_pair(ctx, key_type(i))]);
    }
}

TYPED_TEST(MapTest, erase) {
    using key_type = typename TypeParam::key_type;
    using value_type = typename TypeParam::value_type;
    TypeParam map;
    context_t* ctx = nullptr;

    uint64_t resize_threshold = static_cast<uint64_t>(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_MAX_CAPACITY);
    for (uint64_t i = 0; i <= resize_threshold; ++i) {
        map[std::make_pair(ctx, key_type(i))] = value_type(i);
    }
    EXPECT_EQ(resize_threshold+1, map.count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map.capacity());


    for (uint64_t i = 0; i < resize_threshold + 1; ++i) {
        EXPECT_EQ(value_type(i), map[std::make_pair(ctx, key_type(i))]);
    }

    map[std::make_pair(ctx, key_type(resize_threshold + 1))] = value_type(resize_threshold + 1);
    EXPECT_EQ(resize_threshold + 2, map.count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_GROW_MULTIPLIER, map.capacity());

    for (uint64_t i = 0; i < resize_threshold + 1; ++i) {
        EXPECT_EQ(value_type(i), map[std::make_pair(ctx, key_type(i))]);
    }

    map.erase(ctx, key_type(10));
    EXPECT_EQ(0, map[std::make_pair(ctx, key_type(10))]);
}

TYPED_TEST(MapTest, range) {
    using key_type = typename TypeParam::key_type;
    using value_type = typename TypeParam::value_type;
    TypeParam map;
    context_t* ctx = nullptr;

    std::vector<value_type> result_vector;
    result_vector.resize(16);

    for (uint64_t i = 0; i < 16; ++i) {
        map[std::make_pair(ctx, key_type(i))] = value_type(i);
    }

    uint64_t old_allocations = ::testing::num_allocations;

    for (auto& val: map) {
        result_vector[val.first] = val.second;
    }

    // Ranging over a map should not have caused any allocations.
    EXPECT_EQ(old_allocations, ::testing::num_allocations);


    for (uint64_t i = 0; i < 16; ++i) {
        EXPECT_EQ(result_vector[i], value_type(i));
    }
}

class non_movable_object {
    public:
    non_movable_object() {
        size = 0;
        v = nullptr;
    }

    non_movable_object(uint64_t i) {
        v = gapil_alloc(nullptr, i, 1);
        size = i;
    }

    non_movable_object& operator=(const non_movable_object& _other) {
        size = _other.size;
        v = gapil_alloc(nullptr, size, 1);
        return *this;
    }

    ~non_movable_object() {
        gapil_free(nullptr, v);
    }
    non_movable_object(const non_movable_object& _other) {
        v = gapil_alloc(nullptr, _other.size, 1);
    }
    private:
    void * v;
    uint64_t size;
};

TEST(CppMapTest, constructible_object) {
    gapil::Map<uint32_t, non_movable_object> map;
    context_t* ctx = nullptr;


    uint64_t resize_threshold = static_cast<uint64_t>(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_MAX_CAPACITY);
    for (uint64_t i = 0; i <= resize_threshold; ++i) {
        map[std::make_pair(ctx, i)] = non_movable_object(i + 10);
    }
    EXPECT_EQ(resize_threshold+1, map.count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map.capacity());



    map[std::make_pair(ctx, resize_threshold + 1)] = non_movable_object(10 + resize_threshold + 1);
    EXPECT_EQ(resize_threshold + 2, map.count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_GROW_MULTIPLIER, map.capacity());
}

class movable_object {
    public:
    movable_object() {
        size = 0;
        v = nullptr;
    }

    movable_object(uint64_t i) {
        v = gapil_alloc(nullptr, i, 1);
        size = i;
    }

    movable_object& operator=(movable_object&& _other) {
        size = _other.size;
        v = _other.v;
        _other.size = 0;
        _other.v = nullptr;
        return *this;
    }

    ~movable_object() {
        gapil_free(nullptr, v);
    }
    movable_object(movable_object&& _other) {
        *this = std::move(_other);
    }

    private:
    void * v;
    uint64_t size;
};

TEST(CppMapTest, movable_object) {
    gapil::Map<uint32_t, non_movable_object> map;
    context_t* ctx = nullptr;


    uint64_t resize_threshold = static_cast<uint64_t>(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_MAX_CAPACITY);
    for (uint64_t i = 0; i <= resize_threshold; ++i) {
        map[std::make_pair(ctx, i)] = non_movable_object(i + 10);
    }
    EXPECT_EQ(resize_threshold+1, map.count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE, map.capacity());



    map[std::make_pair(ctx, resize_threshold + 1)] = non_movable_object(10 + resize_threshold + 1);
    EXPECT_EQ(resize_threshold + 2, map.count());
    EXPECT_EQ(GAPIL_MIN_MAP_SIZE * GAPIL_MAP_GROW_MULTIPLIER, map.capacity());
}
