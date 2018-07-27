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

#include "gapil/runtime/cc/runtime.h"

#ifdef __cplusplus
extern "C" {
#endif

#define CMD_FLAGS_HAS_READS 1
#define CMD_FLAGS_HAS_WRITES 2

typedef struct pool_t {
  gapil_pool base;
  uint32_t env;  // env identifier that owns this pool.
} pool;

typedef struct cmd_data_t {
  uint32_t api_idx;
  uint32_t cmd_idx;
  void* args;
  uint64_t id;
  uint64_t flags;
  uint64_t thread;
} cmd_data;

typedef void gapil_extern(gapil_context*, void* args, void* res);

gapil_context* create_context(gapil_module*, arena*);
void destroy_context(gapil_module*, gapil_context*);
void call(gapil_context*, gapil_module*, cmd_data* cmds, uint64_t count,
          uint64_t* res);
gapil_api_module* get_api_module(gapil_module*, uint32_t api_idx);
void register_c_extern(const char* name, gapil_extern* fn);

typedef struct callbacks_t {
  void* apply_reads;
  void* apply_writes;
  void* resolve_pool_data;
  void* call_extern;
  void* copy_slice;
  void* cstring_to_slice;
  void* store_in_database;
  void* make_pool;
  void* free_pool;

  void* clone_slice;
} callbacks;

void set_callbacks(callbacks*);

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus
