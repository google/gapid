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

#include "env.h"

#include "gapil/runtime/cc/cloner/cloner.h"

#include <stdio.h>
#include <string>
#include <unordered_map>

namespace {

std::unordered_map<std::string, gapil_extern*> externs;
gapil_runtime_callbacks go_callbacks;

}  // anonymous namespace

extern "C" {

gapil_context* create_context(gapil_module* m, arena* a) {
  return m->create_context(a);
}

void destroy_context(gapil_module* m, gapil_context* ctx) {
  m->destroy_context(ctx);
}

gapil_api_module* get_api_module(gapil_module* m, uint32_t api_idx) {
  if (api_idx >= m->num_apis) {
    return 0;
  }
  return &m->apis[api_idx];
}

void call(gapil_context* ctx, gapil_module* m, cmd_data_t* cmds, uint64_t count,
          uint64_t* res) {
  for (uint64_t i = 0; i < count; i++) {
    auto cmd = cmds[i];
    gapil_api_module* api_module = get_api_module(m, cmd.api_idx);
    if (api_module == nullptr) {
      fprintf(stderr, "no module for api[%d]\n", int(cmd.api_idx));
      return;
    }
    auto fptr = api_module->cmds[cmd.cmd_idx];
    if (fptr == nullptr) {
      fprintf(stderr, "no function to call for api[%d].cmd[%d] (%p)\n",
              int(cmd.api_idx), int(cmd.cmd_idx),
              &api_module->cmds[cmd.cmd_idx]);
      return;
    }

    ctx->thread = cmd.thread;
    ctx->cmd_id = cmd.id;
    ctx->cmd_idx = i;
    ctx->cmd_args = cmd.args;
    ctx->cmd_flags = cmd.flags;

    res[i] = 0;

    try {
      fptr(ctx);
    } catch (uint32_t err) {
      res[i] = err;
    }
  }
}

void call_extern(gapil_context* ctx, uint8_t* name, void* args, void* res) {
  auto it = externs.find(reinterpret_cast<const char*>(name));
  if (it != externs.end()) {
    it->second(ctx, args, res);
    return;
  }
  go_callbacks.call_extern(ctx, name, args, res);
}

void apply_reads(gapil_context* ctx) {
  if (ctx->cmd_flags & CMD_FLAGS_HAS_READS) {
    go_callbacks.apply_reads(ctx);
  }
}

void apply_writes(gapil_context* ctx) {
  if (ctx->cmd_flags & CMD_FLAGS_HAS_WRITES) {
    go_callbacks.apply_writes(ctx);
  }
}

void set_callbacks(callbacks* cgo) {
  gapil_runtime_callbacks runtime = {0};

#define CAST_FPTR(cb, func) \
  cb.func = reinterpret_cast<decltype(cb.func)>(cgo->func)

  runtime.apply_reads = &apply_reads;
  runtime.apply_writes = &apply_writes;
  runtime.call_extern = &call_extern;
  CAST_FPTR(go_callbacks, apply_reads);
  CAST_FPTR(go_callbacks, apply_writes);
  CAST_FPTR(go_callbacks, call_extern);

  CAST_FPTR(runtime, resolve_pool_data);
  CAST_FPTR(runtime, copy_slice);
  CAST_FPTR(runtime, cstring_to_slice);
  CAST_FPTR(runtime, store_in_database);
  CAST_FPTR(runtime, make_pool);
  CAST_FPTR(runtime, free_pool);
  gapil_set_runtime_callbacks(&runtime);

  gapil_cloner_callbacks cloner = {0};
  CAST_FPTR(cloner, clone_slice);
  gapil_set_cloner_callbacks(&cloner);

#undef CAST_FPTR
}

void register_c_extern(const char* name, gapil_extern* fn) {
  externs[name] = fn;
}

void dump_module(gapil_module* m) {
  fprintf(stderr, "Module:                        %p\n", m);
  fprintf(stderr, "Module.num_apis:               %d (%p)\n", int(m->num_apis),
          &m->num_apis);
  fprintf(stderr, "Module.apis:                   %p (%p)\n", m->apis,
          &m->apis);
  for (uint64_t i = 0; i < m->num_apis; i++) {
    auto api = &m->apis[i];
    fprintf(stderr, "Module.api[%d]:                %p (%p)\n", int(i), m->apis,
            &m->apis);
    fprintf(stderr, "Module.api[%d].globals_offset: %d (%p)\n", int(i),
            int(api->globals_offset), &api->globals_offset);
    fprintf(stderr, "Module.api[%d].globals_size:   %d (%p)\n", int(i),
            int(api->globals_size), &api->globals_size);
    fprintf(stderr, "Module.api[%d].num_cmds:       %d (%p)\n", int(i),
            int(api->num_cmds), &api->num_cmds);
    fprintf(stderr, "Module.api[%d].cmds:           %p (%p)\n", int(i),
            api->cmds, &api->cmds);
    for (uint64_t j = 0; j < api->num_cmds; j++) {
      fprintf(stderr, "Module.api[%d].cmds[%d]:     %p (%p)\n", int(i), int(j),
              api->cmds[j], &api->cmds[j]);
    }
  }
  fprintf(stderr, "Module.num_symbols:            %d (%p)\n",
          int(m->num_symbols), &m->num_symbols);
  fprintf(stderr, "Module.globals_size:           %d (%p)\n",
          int(m->globals_size), &m->globals_size);
}

}  // extern "C"