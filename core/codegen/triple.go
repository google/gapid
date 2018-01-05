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

package codegen

import (
	"strings"

	"github.com/google/gapid/core/os/device"
)

// targetTriple returns an LLVM target triple for the given ABI in the form:
//   <arch><sub>-<vendor>-<os>-<abi>
//
// References:
// https://github.com/llvm-mirror/llvm/blob/master/lib/Support/Triple.cpp
// https://clang.llvm.org/docs/CrossCompilation.html
func targetTriple(dev *device.ABI) string {
	arch, vendor, os, abi := "unknown", "unknown", "unknown", "unknown"
	// Consult Triple.cpp for legal values for each of these.
	// arch:   parseArch() + parseSubArch()
	// vendor: parseVendor()
	// os:     parseOS()
	// abi:    parseEnvironment() + parseFormat()

	switch dev.Architecture {
	case device.ARMv7a:
		arch = "armv7"
	case device.ARMv8a:
		arch = "aarch64"
	case device.X86:
		arch = "i386"
	case device.X86_64:
		arch = "x86_64"
	}

	switch dev.OS {
	case device.Windows:
		vendor, os = "pc", "win32"
	case device.OSX:
		vendor, os = "apple", "darwin"
	case device.Linux:
		os = "linux"
	case device.Android:
		os, abi = "linux", "androideabi"
	}

	return strings.Join([]string{arch, vendor, os, abi}, "-")
}
