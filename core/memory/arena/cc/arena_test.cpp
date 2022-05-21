/*
 * Copyright (C) 2018 Google Inc.
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

#include <gmock/gmock.h>
#include <gtest/gtest.h>
#include <random>
#include <unordered_set>
#include <vector>

#include "core/memory/arena/cc/arena.h"

namespace core {
namespace test {

TEST(allocate_memory, various_sizes) {
  Arena a;
  void* v1 = a.allocate(1, 1);
  void* v2 = a.allocate(2, 1);
  void* v16 = a.allocate(16, 1);
  void* v31 = a.allocate(31, 1);
  void* v32 = a.allocate(32, 1);
  void* v33 = a.allocate(33, 1);
  void* v1024 = a.allocate(1024, 1);
  void* v4096 = a.allocate(4096, 1);
  void* v10241024 = a.allocate(1024 * 1024, 1);
  void* v102410241 = a.allocate(1024 * 1024 + 1, 1);
  void* v10241024m1 = a.allocate(1024 * 1024 - 1, 1);

  EXPECT_EQ(11, a.num_allocations());
  EXPECT_NE(0, a.num_bytes_allocated());

  a.free(v10241024m1);
  a.free(v102410241);
  a.free(v10241024);
  a.free(v4096);
  a.free(v1024);
  a.free(v33);
  a.free(v32);
  a.free(v31);
  a.free(v16);
  a.free(v2);
  a.free(v1);

  EXPECT_EQ(0, a.num_allocations());
  EXPECT_EQ(0, a.num_bytes_allocated());
}

TEST(allocate_memory, overflowing_block) {
  Arena a;
  for (size_t i = 0; i < 1024 * 1024; ++i) {
    void* v = a.allocate(32, 32);
    EXPECT_EQ(0, reinterpret_cast<uintptr_t>(v) & 0x1F);
  }
}

TEST(allocate_memory, random_alloc_free) {
  std::default_random_engine generator;
  Arena a;
  std::unordered_set<void*> allocations;

  std::uniform_int_distribution<int> distribution(16, 8182);
  for (size_t i = 0; i < 2048; ++i) {
    allocations.insert(a.allocate(distribution(generator), 32));
  }

  EXPECT_EQ(2048, a.num_allocations());
  size_t old_allocated = a.num_bytes_allocated();
  EXPECT_GE(old_allocated, 2048 * 16);

  for (size_t i = 0; i < 1024; ++i) {
    auto it = allocations.begin();
    a.free(*it);
    allocations.erase(it);
  }

  EXPECT_EQ(1024, a.num_allocations());
  EXPECT_GE(a.num_bytes_allocated(), 1024 * 16);
  EXPECT_LT(a.num_bytes_allocated(), old_allocated);

  for (size_t i = 0; i < 1024; ++i) {
    auto it = allocations.begin();
    a.free(*it);
    allocations.erase(it);
  }

  EXPECT_EQ(0, a.num_allocations());
  EXPECT_EQ(0, a.num_bytes_allocated());
}

using reallocate_memory_tests = ::testing::TestWithParam<uint32_t>;

TEST_P(reallocate_memory_tests, reallocate) {
  std::vector<uint8_t> pattern;
  for (size_t i = 0; i < GetParam(); i++) {
    pattern.push_back(static_cast<uint8_t>(i));
  }

  Arena a;
  auto p = reinterpret_cast<uint8_t*>(a.allocate(pattern.size(), 16));
  memcpy(p, &pattern[0], pattern.size());

  p = reinterpret_cast<uint8_t*>(a.reallocate(p, pattern.size() * 2, 16));
  auto got = std::vector<uint8_t>(p, p + pattern.size());
  EXPECT_EQ(got, pattern);
}

INSTANTIATE_TEST_SUITE_P(many_values, reallocate_memory_tests,
                         ::testing::Values(1, 15, 16, 31, 32, 44, 1024, 4093));

using memory_protection_tests = ::testing::TestWithParam<uint32_t>;

TEST_P(memory_protection_tests, protect_memory) {
  Arena a;
  uint8_t* x = static_cast<uint8_t*>(a.allocate(GetParam(), 1));
  *x = 4;
  a.protect();
  EXPECT_DEATH({ *x = 32; }, "");
}

TEST_P(memory_protection_tests, free_memory) {
  Arena a;
  uint8_t* x = static_cast<uint8_t*>(a.allocate(GetParam(), 1));
  *x = 4;
  a.protect();
}

TEST_P(memory_protection_tests, unprotect_memory) {
  Arena a;
  uint8_t* x = static_cast<uint8_t*>(a.allocate(GetParam(), 1));
  *x = 4;
  a.protect();
  a.unprotect();
  *x = 5;
}

// DO NOT add another death test.
// There is some wonderful weirdness about how this will handle
// the offsets.
INSTANTIATE_TEST_SUITE_P(many_values, memory_protection_tests,
                         ::testing::Values(1, 31, 32, 44, 1024, 4093));

}  // namespace test
}  // namespace core