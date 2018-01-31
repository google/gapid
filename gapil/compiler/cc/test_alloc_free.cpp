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

#include "builtins.h"
#include <unordered_map>
#include <stdlib.h>
#include <assert.h>

namespace testing {
static uint64_t num_allocations = 0;
static uint64_t allocated_bytes = 0;

static std::unordered_map<void*, uint64_t> allocations;
}

void* gapil_alloc(context_t*, uint64_t count, uint64_t) {
  testing::allocated_bytes += count;
  testing::num_allocations += 1;
  void* alloc = malloc(count);
  testing::allocations[alloc] = count;
  return alloc;
}

void gapil_free(context_t*, void* v) {
  testing::num_allocations -= 1;

  assert(testing::allocations.count(v));
  testing::allocated_bytes -= testing::allocations[v];
  testing::allocations.erase(v);
  return free(v);
}


void* gapil_realloc(context* ctx, void* ptr, uint64_t size, uint64_t align) {
    assert(testing::allocations.count(ptr));

    void* retptr = realloc(ptr, size);
    testing::allocated_bytes += size;
    testing::allocated_bytes -= testing::allocations[ptr];
    testing::allocations.erase(ptr);
    testing::allocations[retptr] = size;
    return retptr;
}
