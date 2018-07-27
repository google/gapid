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

package test

// #include "gapil/runtime/cc/replay/replay.h"
//
// uint64_t test_remapped_key(gapil_context* ctx, void* v) { return *(uint32_t*)v; }
// static void register_remap_funcs() {
//   gapil_replay_register_remap_func("test", "remapped", &test_remapped_key);
// }
import "C"

import (
	"github.com/google/gapid/gapis/api"
)

func init() {
	C.register_remap_funcs()
}

func (i Remapped) remap(cmd api.Cmd, s *api.GlobalState) (interface{}, bool) {
	return i, true
}
