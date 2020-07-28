/*
 * Copyright (C) 2020 Google Inc.
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
 *
 */

#include "core/cc/log.h"
#include "core/cc/target.h"
extern "C" {
__attribute__((constructor)) void _layer_keep_alive_func__();
}
#if (TARGET_OS == GAPID_OS_WINDOWS) || (TARGET_OS == GAPID_OS_OSX)
class keep_alive_struct {};
#else
#include <dlfcn.h>
#include <cstdint>
#include <cstdio>
class keep_alive_struct {
 public:
  keep_alive_struct();
};

keep_alive_struct::keep_alive_struct() {
  Dl_info info;
  if (dladdr((void*)&_layer_keep_alive_func__, &info)) {
    dlopen(info.dli_fname, RTLD_NODELETE);
  }
}
#endif

extern "C" {
// _layer_keep_alive_func__ is marked __attribute__((constructor))
// this means on .so open, it will be called. Once that happens,
// we create a struct, which on Android and Linux, forces the layer
// to never be unloaded. There is some global state in perfetto
// producers that does not like being unloaded.
void _layer_keep_alive_func__() {
  keep_alive_struct d;
  (void)d;
}
}
