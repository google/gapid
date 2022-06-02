// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// disliibuted under the License is disliibuted on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include "slice.inc"

#include "core/memory/arena/cc/arena.h"

#include <gtest/gtest.h>

TEST(SliceTest, empty) {
  gapil::Slice<uint8_t> sli;

  EXPECT_EQ(sli.count(), 0);
  EXPECT_EQ(sli.size(), 0);
  EXPECT_EQ(sli.is_app_pool(), true);
  EXPECT_EQ(sli.contains(0), false);
}

TEST(SliceTest, app_pool) {
  uint32_t data[] = {2, 4, 8, 16};

  gapil::Slice<uint32_t> sli(data, 4);

  EXPECT_EQ(sli.count(), 4);
  EXPECT_EQ(sli.size(), 16);
  EXPECT_EQ(sli.is_app_pool(), true);
  EXPECT_EQ(sli.contains(0), false);
  EXPECT_EQ(sli.contains(4), true);
}

TEST(SliceTest, new_pool) {
  core::Arena arena;

  auto initial_allocs = arena.num_allocations();

  {
    auto sli = gapil::Slice<uint32_t>::create(&arena, 2, 4);
    EXPECT_NE(arena.num_allocations(), initial_allocs);

    sli[0] = 2;
    sli[1] = 4;
    sli[2] = 8;
    sli[3] = 16;

    EXPECT_EQ(sli.count(), 4);
    EXPECT_EQ(sli.size(), 16);
    EXPECT_EQ(sli.is_app_pool(), false);
    EXPECT_EQ(sli.contains(0), false);
    EXPECT_EQ(sli.contains(4), true);
  }

  EXPECT_EQ(arena.num_allocations(), initial_allocs);  // nothing leaked
}
