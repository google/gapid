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

import (
	"fmt"
	"unsafe"

	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/gapil/executor"
)

// #include "gapil/runtime/cc/runtime.h"
//
// typedef buffer* (TGetReplayOpcodes) (context*);
// buffer* get_replay_opcodes(TGetReplayOpcodes* func, context* ctx) { return func(ctx); }
import "C"

// Opcodes returns the encoded opcodes from the context
func Opcodes(env *executor.Env) ([]byte, error) {
	pfn := env.Executor.FunctionAddress(GetReplayOpcodes)
	if pfn == nil {
		return nil, fmt.Errorf("Program did not export the function to get the replay opcodes")
	}

	gro := (*C.TGetReplayOpcodes)(pfn)
	ctx := (*C.context)(env.CContext())

	buf := C.get_replay_opcodes(gro, ctx)

	return slice.Bytes(unsafe.Pointer(buf.data), uint64(buf.size)), nil
}
