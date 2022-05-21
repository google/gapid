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

#include "maker.h"

#include "core/memory/arena/cc/arena.h"

#include <gtest/gtest.h>

class MakerTest : public ::testing::Test {
  void TearDown() {
    EXPECT_EQ(0, arena.num_allocations());
    EXPECT_EQ(0, arena.num_bytes_allocated());
  }

 public:
  core::Arena arena;
};

TEST_F(MakerTest, make_pointer) {
  printf("arena: %p\n", &this->MakerTest::arena);
  EXPECT_EQ(nullptr, gapil::make<void*>(&this->MakerTest::arena));
  EXPECT_EQ(nullptr, gapil::make<const void*>(&this->MakerTest::arena));
  EXPECT_EQ(nullptr, gapil::make<int*>(&this->MakerTest::arena));
  EXPECT_EQ(nullptr, gapil::make<const int*>(&this->MakerTest::arena));
}

template <typename T>
class MakerTestInteger : public MakerTest {};

using MakerTestIntegerTypes =
    ::testing::Types<uint8_t, int8_t, uint16_t, int16_t, uint32_t, int32_t,
                     uint64_t, int64_t>;

TYPED_TEST_SUITE(MakerTestInteger, MakerTestIntegerTypes);

TYPED_TEST(MakerTestInteger, make_integer) {
  EXPECT_EQ(0, gapil::make<TypeParam>(&this->MakerTest::arena));
}
