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

#include "ref.inc"

#include "core/memory/arena/cc/arena.h"

#include <gtest/gtest.h>

template <typename T>
class RefTest : public ::testing::Test {};

using RefTestTypes =
    ::testing::Types<gapil::Ref<uint32_t>, gapil::Ref<uint16_t>,
                     gapil::Ref<uint64_t>>;

TYPED_TEST_SUITE(RefTest, RefTestTypes);

TYPED_TEST(RefTest, null) {
  auto ref = TypeParam();

  EXPECT_EQ(ref, ref);
  EXPECT_EQ(ref, TypeParam());

  EXPECT_EQ(ref.get(), nullptr);
}

TYPED_TEST(RefTest, create) {
  core::Arena arena;

  {
    auto ref = TypeParam::create(&arena);

    *ref = 5;

    EXPECT_EQ(ref, ref);
    EXPECT_NE(ref, TypeParam());

    EXPECT_EQ(1,
              arena.num_allocations());  // single allocation for the ref object
  }

  // ref has now fallen out of scope
  EXPECT_EQ(0, arena.num_allocations());
  EXPECT_EQ(0, arena.num_bytes_allocated());
}

TYPED_TEST(RefTest, assignment) {
  core::Arena arena;

  {
    auto refA = TypeParam::create(&arena);

    EXPECT_EQ(arena.num_allocations(), 1);  // refA owns 1 allocation

    TypeParam refB;

    EXPECT_EQ(arena.num_allocations(), 1);  // refB has made no allocations

    refB = refA;

    EXPECT_EQ(arena.num_allocations(), 1);  // refB is sharing refA's allocation

    refA = TypeParam();

    EXPECT_EQ(arena.num_allocations(),
              1);  // refB now exclusively owns the allocation

    refA = refB;

    EXPECT_EQ(arena.num_allocations(), 1);
  }

  EXPECT_EQ(arena.num_allocations(), 0);
}
