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
import "C"

var (
	// ExternA will be called whenever the test_extern_a extern function is
	// called.
	ExternA func(*executor.Env, uint64, float32, bool) uint64

	// ExternB will be called whenever the test_extern_b extern function is
	// called.
	ExternB func(*executor.Env, string) bool
)

//export test_extern_a
func test_extern_a(ctx unsafe.Pointer, i uint64, f float32, b bool, out *uint64) {
	env := executor.GetEnv(ctx)
	*out = ExternA(env, i, f, b)
}

//export test_extern_b
func test_extern_b(ctx unsafe.Pointer, s *C.string, out *bool) {
	env := executor.GetEnv(ctx)
	*out = ExternB(env, C.GoString((*C.char)((unsafe.Pointer)(&s.data[0]))))
}
