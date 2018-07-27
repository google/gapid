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

namespace {

struct Pool : gapil_pool {
  uint64_t size;
  uint8_t* buffer;
  arena_t* arena;
};

}  // anonymous namespace

extern "C" {

void* resolve_pool_data(gapil_context* ctx, gapil_pool* pool, uint64_t ptr,
                        gapil_data_access access, uint64_t size) {
  auto p = static_cast<Pool*>(pool);
  return (p == nullptr) ? reinterpret_cast<void*>(static_cast<uintptr_t>(ptr))
                        : &p->buffer[ptr];
}

gapil_pool* make_pool(gapil_context* ctx, uint64_t size) {
  auto arena = reinterpret_cast<core::Arena*>(ctx->arena);
  auto pool = arena->create<Pool>();
  static uint64_t next_pool_id = 1;
  auto id = next_pool_id++;
  pool->ref_count = 1;
  pool->id = id;
  pool->size = size;
  pool->buffer = reinterpret_cast<uint8_t*>(arena->allocate(size, 16));
  pool->arena = ctx->arena;
  return pool;
}

void free_pool(gapil_pool* pool) {
  auto p = static_cast<Pool*>(pool);
  auto arena = reinterpret_cast<core::Arena*>(p->arena);
  arena->free(p->buffer);
  arena->free(p);
}

void register_runtime_callbacks() {
  gapil_runtime_callbacks cb = {0};
  cb.resolve_pool_data = &resolve_pool_data;
  cb.make_pool = &make_pool;
  cb.free_pool = &free_pool;
  gapil_set_runtime_callbacks(&cb);
}

}  // extern "C"

class SliceTest : public ::testing::Test {
 public:
  virtual void SetUp() override { register_runtime_callbacks(); }
};

TEST_F(SliceTest, empty) {
  gapil::Slice<uint8_t> sli;

  EXPECT_EQ(sli.count(), 0);
  EXPECT_EQ(sli.size(), 0);
  EXPECT_EQ(sli.is_app_pool(), true);
  EXPECT_EQ(sli.contains(0), false);
}

TEST_F(SliceTest, app_pool) {
  uint32_t data[] = {2, 4, 8, 16};

  gapil::Slice<uint32_t> sli(data, 4);

  EXPECT_EQ(sli.count(), 4);
  EXPECT_EQ(sli.size(), 16);
  EXPECT_EQ(sli.is_app_pool(), true);
  EXPECT_EQ(sli.contains(0), false);
  EXPECT_EQ(sli.contains(4), true);
}

TEST_F(SliceTest, new_pool) {
  core::Arena arena;
  gapil_context_t ctx;
  ctx.arena = reinterpret_cast<arena_t*>(&arena);

  auto initial_allocs = arena.num_allocations();

  {
    auto sli = gapil::Slice<uint32_t>::create(&ctx, 4);
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
