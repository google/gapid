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

package replay

import "github.com/google/gapid/core/codegen"

//#define QUOTE(x) #x
//#define DECL_GAPIL_REPLAY_FUNC(RETURN, NAME, ...) \
//	const char* NAME##_sig = QUOTE(RETURN NAME(__VA_ARGS__));
//#include "gapil/runtime/cc/replay/replay.h"
import "C"

// callbacks are the runtime functions used to build the replay instructions.
type callbacks struct {
	initData        *codegen.Function
	termData        *codegen.Function
	reserveMemory   *codegen.Function
	allocateMemory  *codegen.Function
	getRemapFunc    *codegen.Function
	addResource     *codegen.Function
	addConstant     *codegen.Function
	addRemapping    *codegen.Function
	lookupRemapping *codegen.Function
}

func (r *replayer) parseCallbacks() {
	// Function typedef
	r.T.Alias("gapil_replay_remap_func", r.T.Function(r.T.Uint64, r.T.CtxPtr, r.T.VoidPtr))

	r.callbacks.initData = r.M.ParseFunctionSignature(C.GoString(C.gapil_replay_init_data_sig))
	r.callbacks.termData = r.M.ParseFunctionSignature(C.GoString(C.gapil_replay_term_data_sig))
	r.callbacks.reserveMemory = r.M.ParseFunctionSignature(C.GoString(C.gapil_replay_reserve_memory_sig))
	r.callbacks.allocateMemory = r.M.ParseFunctionSignature(C.GoString(C.gapil_replay_allocate_memory_sig))
	r.callbacks.addResource = r.M.ParseFunctionSignature(C.GoString(C.gapil_replay_add_resource_sig))
	r.callbacks.addConstant = r.M.ParseFunctionSignature(C.GoString(C.gapil_replay_add_constant_sig))
	r.callbacks.getRemapFunc = r.M.ParseFunctionSignature(C.GoString(C.gapil_replay_get_remap_func_sig))
	r.callbacks.addRemapping = r.M.ParseFunctionSignature(C.GoString(C.gapil_replay_add_remapping_sig))
	r.callbacks.lookupRemapping = r.M.ParseFunctionSignature(C.GoString(C.gapil_replay_lookup_remapping_sig))
}
