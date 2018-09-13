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

package testutils

import (
	"unsafe"

	"github.com/google/gapid/gapil/executor"
)

//#include "gapil/runtime/cc/runtime.h"
//
// typedef struct extern_a_args_t {
//   uint64_t   i;
//   float      f;
//   GAPIL_BOOL b;
// } extern_a_args;
//
// typedef struct extern_b_args_t {
//   string* s;
// } extern_b_args;
import "C"

var (
	// ExternA will be called whenever the test_extern_a extern function is
	// called.
	ExternA func(*executor.Env, uint64, float32, bool) uint64

	// ExternB will be called whenever the test_extern_b extern function is
	// called.
	ExternB func(*executor.Env, string) bool
)

func init() {
	executor.RegisterGoExtern("test.extern_a", externA)
	executor.RegisterGoExtern("test.extern_b", externB)
}

func externA(env *executor.Env, args, out unsafe.Pointer) {
	a := (*C.extern_a_args)(args)
	o := (*uint64)(out)
	b := false
	if a.b != 0 {
		b = true
	}
	*o = ExternA(env, uint64(a.i), float32(a.f), b)
}

func externB(env *executor.Env, args, out unsafe.Pointer) {
	a := (*C.extern_b_args)(args)
	o := (*bool)(out)
	*o = ExternB(env, C.GoString((*C.char)((unsafe.Pointer)(&a.s.data[0]))))
}
