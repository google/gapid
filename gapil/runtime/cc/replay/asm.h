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

#ifndef __GAPIL_RUNTIME_REPLAY_ASM_H__
#define __GAPIL_RUNTIME_REPLAY_ASM_H__

#include "gapil/runtime/cc/runtime.h"

typedef enum gapil_replay_asm_type_t {
  GAPIL_REPLAY_ASM_TYPE_BOOL,
  GAPIL_REPLAY_ASM_TYPE_INT8,
  GAPIL_REPLAY_ASM_TYPE_INT16,
  GAPIL_REPLAY_ASM_TYPE_INT32,
  GAPIL_REPLAY_ASM_TYPE_INT64,
  GAPIL_REPLAY_ASM_TYPE_UINT8,
  GAPIL_REPLAY_ASM_TYPE_UINT16,
  GAPIL_REPLAY_ASM_TYPE_UINT32,
  GAPIL_REPLAY_ASM_TYPE_UINT64,
  GAPIL_REPLAY_ASM_TYPE_FLOAT,
  GAPIL_REPLAY_ASM_TYPE_DOUBLE,
  GAPIL_REPLAY_ASM_TYPE_ABSOLUTE_POINTER,
  GAPIL_REPLAY_ASM_TYPE_CONSTANT_POINTER,
  GAPIL_REPLAY_ASM_TYPE_VOLATILE_POINTER,
  GAPIL_REPLAY_ASM_TYPE_OBSERVED_POINTER_NAMESPACE_0,  // namespaces increment
                                                       // from here
} gapil_replay_asm_type;

typedef enum gapil_replay_asm_inst_t {
  GAPIL_REPLAY_ASM_INST_CALL,
  GAPIL_REPLAY_ASM_INST_PUSH,
  GAPIL_REPLAY_ASM_INST_POP,
  GAPIL_REPLAY_ASM_INST_COPY,
  GAPIL_REPLAY_ASM_INST_CLONE,
  GAPIL_REPLAY_ASM_INST_LOAD,
  GAPIL_REPLAY_ASM_INST_STORE,
  GAPIL_REPLAY_ASM_INST_STRCPY,
  GAPIL_REPLAY_ASM_INST_RESOURCE,
  GAPIL_REPLAY_ASM_INST_POST,
  GAPIL_REPLAY_ASM_INST_ADD,
  GAPIL_REPLAY_ASM_INST_LABEL,
  GAPIL_REPLAY_ASM_INST_SWITCHTHREAD,
} gapil_replay_asm_inst;

typedef struct gapil_replay_asm_value_t {
  uint64_t data;
  gapil_replay_asm_type data_type;
} gapil_replay_asm_value;

// call is an instruction to call a VM registered function.
// This instruction will pop the parameters from the VM stack starting with the
// first parameter. If push_return is true, then the return value of the
// function call will be pushed to the top of the VM stack.
typedef struct gapil_replay_asm_call_t {
  GAPIL_BOOL push_return;
  uint8_t api_index;     // The index of the API this call belongs to
  uint16_t function_id;  // The function id registered with the VM to invoke.
} gapil_replay_asm_call;

// push is an instruction to push value to the top of the VM stack.
typedef struct gapil_replay_asm_push_t {
  gapil_replay_asm_value value;  // The value to push on to the VM stack.
} gapil_replay_asm_push;

// pop is an instruction that discards Count values from the top of the VM
// stack.
typedef struct gapil_replay_asm_pop_t {
  uint32_t count;
} gapil_replay_asm_pop;

// copy is an instruction that pops the target address and then the source
// address from the top of the VM stack, and then copies count bytes from
// source to target.
typedef struct gapil_replay_asm_copy_t {
  uint64_t count;
} gapil_replay_asm_copy;

// clone is an instruction that makes a copy of the the n-th element from the
// top of the VM stack and pushes the copy to the top of the VM stack.
typedef struct gapil_replay_asm_clone_t {
  uint32_t n;
} gapil_replay_asm_clone;

// load is an instruction that loads the value of type type from source
// and pushes the loaded value to the top of the VM stack.
typedef struct gapil_replay_asm_load_t {
  gapil_replay_asm_type data_type;
  gapil_replay_asm_value source;
} gapil_replay_asm_load;

// store is an instruction that pops the value from the top of the VM stack and
// writes the value to Destination.
typedef struct gapil_replay_asm_store_t {
  gapil_replay_asm_value dst;
} gapil_replay_asm_store;

// strcpy is an instruction that pops the target address then the source address
// from the top of the VM stack, and then copies at most max_count-1 bytes from
// source to target. If the max_count is greater than the source string length,
// then the target will be padded with 0s. The destination buffer will always be
// 0-terminated.
typedef struct gapil_replay_asm_strcpy_t {
  uint64_t max_count;
} gapil_replay_asm_strcpy;

// resource is an instruction that loads the resource with index and writes the
// resource to dest.
typedef struct gapil_replay_asm_resource_t {
  uint32_t index;
  gapil_replay_asm_value dest;
} gapil_replay_asm_resource;

// post is an instruction that posts size bytes from source to the server.
typedef struct gapil_replay_asm_post_t {
  gapil_replay_asm_value source;
  uint64_t size;
} gapil_replay_asm_post;

// add is an instruction that pops and sums the top count stack values, pushing
// the result to the top of the stack. Each summed value must have the same
// type.
typedef struct gapil_replay_asm_add_t {
  uint32_t count;
} gapil_replay_asm_add;

// label is an instruction that holds a marker value, used for debugging.
typedef struct gapil_replay_asm_label_t {
  uint32_t value;
} gapil_replay_asm_label;

// switchthread is an instruction that changes execution to a different thread.
typedef struct gapil_replay_asm_switchthread_t {
  uint32_t index;
} gapil_replay_asm_switchthread;

#endif  // __GAPIL_RUNTIME_REPLAY_ASM_H__
