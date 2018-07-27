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

package executor

import (
	"unsafe"
)

// #include "cgo.h"
// #include "env.h"
import "C"

func (e *Env) call(cmds *C.cmd_data, count C.uint64_t, res *C.uint64_t) {
	ctx := e.cCtx
	m := e.Executor.module
	C.call(ctx, m, cmds, count, res)
}

func callbacks() *C.callbacks {
	return &C.callbacks{
		apply_reads:       C.applyReads,
		apply_writes:      C.applyWrites,
		resolve_pool_data: C.resolvePoolData,
		call_extern:       C.callExtern,
		copy_slice:        C.copySlice,
		cstring_to_slice:  C.cstringToSlice,
		store_in_database: C.storeInDatabase,
		make_pool:         C.makePool,
		free_pool:         C.freePool,
		clone_slice:       C.cloneSlice,
	}
}

//export applyReads
func applyReads(c *C.gapil_context) {
	env(c).applyReads()
}

//export applyWrites
func applyWrites(c *C.gapil_context) {
	env(c).applyWrites()
}

//export resolvePoolData
func resolvePoolData(c *C.gapil_context, pool *C.gapil_pool, ptr C.uint64_t, access C.gapil_data_access, size C.uint64_t) unsafe.Pointer {
	p := (*C.pool)(unsafe.Pointer(pool))
	return env(c).resolvePoolData(p, ptr, access, size)
}

//export copySlice
func copySlice(c *C.gapil_context, dst, src *C.gapil_slice) {
	env(c).copySlice(dst, src)
}

//export cstringToSlice
func cstringToSlice(c *C.gapil_context, ptr C.uint64_t, out *C.gapil_slice) {
	env(c).cstringToSlice(ptr, out)
}

//export storeInDatabase
func storeInDatabase(c *C.gapil_context, ptr unsafe.Pointer, size C.uint64_t, idOut *C.uint8_t) {
	env(c).storeInDatabase(ptr, size, idOut)
}

//export makePool
func makePool(c *C.gapil_context, size C.uint64_t) *C.gapil_pool {
	return &env(c).makePool().base
}

//export freePool
func freePool(pool *C.gapil_pool) {
	p := (*C.pool)(unsafe.Pointer(pool))
	envFromID(envID(p.env)).freePool(p)
}

//export callExtern
func callExtern(c *C.gapil_context, name *C.uint8_t, args, res unsafe.Pointer) {
	env(c).callExtern(name, args, res)
}

//export cloneSlice
func cloneSlice(c *C.gapil_context, dst, src *C.gapil_slice) {
	env(c).cloneSlice(dst, src)
}
