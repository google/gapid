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

#include "stdint.h"

#include "gapil/runtime/cc/runtime.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct cmd_ints_t {
  uint64_t thread;
  uint8_t a;
  int8_t b;
  uint16_t c;
  int16_t d;
  uint32_t e;
  int32_t f;
  uint64_t g;
  int64_t h;
} cmd_ints;

typedef struct cmd_intsCall_t {
  int64_t result;
} cmd_intsCall;

typedef struct cmd_floats_t {
  uint64_t thread;
  float a;
  double b;
} cmd_floats;

typedef struct cmd_enums_t {
  uint64_t thread;
  uint32_t e;
  int64_t e_s64;
} cmd_enums;

typedef struct cmd_arrays_t {
  uint64_t thread;
  uint8_t a[1];
  int32_t b[2];
  float c[3];
} cmd_arrays;

typedef struct cmd_pointers_t {
  uint64_t thread;
  uint8_t* a;
  int32_t* b;
  float* c;
} cmd_pointers;

void cmd__cmd_ints__encode(cmd_ints* cmd, context* ctx, uint8_t is_group);
void cmd__cmd_intsCall__encode(cmd_intsCall* cmd, context* ctx,
                               uint8_t is_group);
void cmd__cmd_floats__encode(cmd_floats* cmd, context* ctx, uint8_t is_group);
void cmd__cmd_enums__encode(cmd_enums* cmd, context* ctx, uint8_t is_group);
void cmd__cmd_arrays__encode(cmd_arrays* cmd, context* ctx, uint8_t is_group);
void cmd__cmd_pointers__encode(cmd_pointers* cmd, context* ctx,
                               uint8_t is_group);

typedef struct int_types_t {
  uint8_t a;
  int8_t b;
  uint16_t c;
  int16_t d;
  uint32_t f;
  int32_t g;
} int_types;

#define INT_TYPES_SIZE sizeof(int_types)

typedef struct basic_types_t {
  uint8_t a;
  int8_t b;
  uint16_t c;
  int16_t d;
  float e;
  uint32_t f;
  int32_t g;
  double h;
  uint64_t i;
  int64_t j;
  uint8_t k;
  uint32_t l;
  uint32_t* m;
  string* n;
} basic_types;

typedef struct inner_class_t {
  basic_types a;
} inner_class;

typedef struct nested_classes_t {
  inner_class a;
} nested_classes;

typedef struct map_types_t {
  map* a;
  map* b;
  map* c;
  map* d;
} map_types;

typedef struct ref_types_t {
  ref* a;
  ref* b;
  ref* c;
  ref* d;
} ref_types;

typedef struct slice_types_t {
  slice a;
  slice b;
  slice c;
} slice_types;

void basic_types__encode(basic_types* c, context* ctx, uint8_t is_group);
void nested_classes__encode(nested_classes* c, context* ctx, uint8_t is_group);
void map_types__encode(map_types* c, context* ctx, uint8_t is_group);
void ref_types__encode(ref_types* c, context* ctx, uint8_t is_group);
void slice_types__encode(slice_types* c, context* ctx, uint8_t is_group);

// Test helper functions.
void create_map_u32(arena*, map**);
void insert_map_u32(map* m, uint32_t k, uint32_t v);
void create_map_string(arena*, map**);
void insert_map_string(map* m, const char* k, const char* v);

basic_types* create_basic_types_ref(arena*, ref**);
inner_class* create_inner_class_ref(arena*, ref**);

context* create_context(arena* arena);
void destroy_context(context* ctx);

#ifdef __cplusplus
}  // extern "C"
#endif
