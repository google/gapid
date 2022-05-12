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

#include "string.h"

#include "core/memory/arena/cc/arena.h"

#include <gtest/gtest.h>

TEST(StringTest, empty) {
  gapil::String str;

  EXPECT_EQ(str.length(), 0);

  EXPECT_EQ(str, gapil::String());

  EXPECT_STREQ(str.c_str(), "");
}

TEST(StringTest, constructors) {
  auto abc123 = "abc123";
  core::Arena arena;
  {
    gapil::String str;
    EXPECT_EQ(str.length(), 0);
    EXPECT_STREQ(str.c_str(), "");
  }
  {
    gapil::String strA(&arena, abc123);
    gapil::String strB(strA);
    EXPECT_EQ(strB.length(), 6);
    EXPECT_STREQ(strB.c_str(), abc123);
  }
  {
    gapil::String strA(&arena, abc123);
    gapil::String strB(std::move(strA));
    EXPECT_EQ(strB.length(), 6);
    EXPECT_STREQ(strB.c_str(), abc123);
  }
  {
    gapil::String str(&arena, abc123);
    EXPECT_EQ(str.length(), 6);
    EXPECT_STREQ(str.c_str(), abc123);
  }
  {
    gapil::String str(&arena, {'a', 'b', 'c'});
    EXPECT_EQ(str.length(), 3);
    EXPECT_STREQ(str.c_str(), "abc");
  }
  {
    gapil::String str(&arena, &abc123[2], &abc123[4]);
    EXPECT_EQ(str.length(), 2);
    EXPECT_STREQ(str.c_str(), "c1");
  }
  {
    gapil::String str(&arena, &abc123[2], 3);
    EXPECT_EQ(str.length(), 3);
    EXPECT_STREQ(str.c_str(), "c12");
  }

  EXPECT_EQ(arena.num_allocations(), 0);  // nothing leaked
}

TEST(StringTest, allocs) {
  core::Arena arena;

  {
    gapil::String str(&arena, "hello world");

    EXPECT_EQ(arena.num_allocations(), 1);  // str owns 1 allocation

    EXPECT_EQ(str.length(), 11);

    EXPECT_EQ(str, gapil::String(&arena, "hello world"));

    EXPECT_EQ(arena.num_allocations(), 1);  // temporary has been freed

    EXPECT_STREQ(str.c_str(), "hello world");
  }

  EXPECT_EQ(arena.num_allocations(), 0);  // str has fallen out of scope
}

TEST(StringTest, assignment) {
  core::Arena arena;

  {
    gapil::String strA(&arena, "hello world");

    EXPECT_EQ(arena.num_allocations(), 1);  // strA owns 1 allocation

    gapil::String strB;

    EXPECT_EQ(arena.num_allocations(), 1);  // strB has made no allocations

    strB = strA;

    EXPECT_EQ(arena.num_allocations(), 1);  // strB is sharing strA's allocation

    strA = gapil::String();

    EXPECT_EQ(arena.num_allocations(),
              1);  // strB now exclusively owns the allocation

    strA = strB;

    EXPECT_EQ(arena.num_allocations(), 1);

    EXPECT_STREQ(strB.c_str(), "hello world");
  }

  EXPECT_EQ(arena.num_allocations(), 0);
}

TEST(StringTest, compare) {
  core::Arena arena;

  gapil::String strA(&arena, "meow");
  gapil::String strB(&arena, "woof");

  EXPECT_FALSE(strA == strB);
  EXPECT_TRUE(strA != strB);
  EXPECT_TRUE(strA < strB);
  EXPECT_TRUE(strA <= strB);
  EXPECT_TRUE(strB > strA);
  EXPECT_TRUE(strB >= strA);
}

TEST(StringTest, clear) {
  core::Arena arena;

  gapil::String str(&arena, "hello");

  EXPECT_EQ(arena.num_allocations(), 1);

  str.clear();

  EXPECT_EQ(arena.num_allocations(), 0);
  EXPECT_STREQ(str.c_str(), "");
}
