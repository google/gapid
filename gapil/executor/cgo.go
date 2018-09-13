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

func callbacks() *C.callbacks {
	return &C.callbacks{
		apply_reads:       C.applyReads,
		apply_writes:      C.applyWrites,
		resolve_pool_data: C.resolvePoolData,
		call_extern:       C.callExtern,
		store_in_database: C.storeInDatabase,
	}
}

//export applyReads
func applyReads(c *C.context) {
	env(c).applyReads()
}

//export applyWrites
func applyWrites(c *C.context) {
	env(c).applyWrites()
}

//export resolvePoolData
func resolvePoolData(c *C.context, pool *C.pool, ptr C.uint64_t, access C.gapil_data_access, size *C.uint64_t) unsafe.Pointer {
	return env(c).resolvePoolData(pool, ptr, access, size)
}

//export storeInDatabase
func storeInDatabase(c *C.context, ptr unsafe.Pointer, size C.uint64_t, idOut *C.uint8_t) {
	env(c).storeInDatabase(ptr, size, idOut)
}

//export callExtern
func callExtern(c *C.context, name *C.uint8_t, args, res unsafe.Pointer) {
	env(c).callExtern(name, args, res)
}
