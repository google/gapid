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

package codegen

import (
	"github.com/google/gapid/core/os/device"
)

// DataLayout returns an LLVM data layout string for the given ABI, or an empty
// string if there is no known data layout for the given ABI.
// Reference: https://llvm.org/docs/LangRef.html#langref-datalayout
func DataLayout(abi *device.ABI) string {
	switch abi.Architecture {
	case device.ARMv7a:
		// clang -target armv7-none-linux-androideabi -march=armv7-a
		return "e-m:e-p:32:32-i64:64-v128:64:128-a:0:32-n32-S64"
	case device.ARMv8a:
		// clang -target aarch64-none-linux-androideabi -march=armv8
		return "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128"
	case device.X86_64:
		return "e-m:e-i64:64-f80:128-n8:16:32:64-S128"
	}
	return ""
}
