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

// TargetTriple returns an LLVM target triple for the given ABI in the form:
//   <arch><sub>-<vendor>-<os>-<abi>
//
// References:
// https://github.com/llvm-mirror/llvm/blob/master/lib/Support/Triple.cpp
// https://clang.llvm.org/docs/CrossCompilation.html
func TargetTriple(dev *device.ABI) Triple {
	out := Triple{"unknown", "unknown", "unknown", "unknown"}
	// Consult Triple.cpp for legal values for each of these.
	// arch:   parseArch() + parseSubArch()
	// vendor: parseVendor()
	// os:     parseOS()
	// abi:    parseEnvironment() + parseFormat()

	switch dev.Architecture {
	case device.ARMv7a:
		out.arch = "armv7"
	case device.ARMv8a:
		out.arch = "aarch64"
	case device.X86:
		out.arch = "i386"
	case device.X86_64:
		out.arch = "x86_64"
	}

	switch dev.OS {
	case device.Windows:
		out.vendor, out.os, out.abi = "w64", "windows", "gnu"
	case device.OSX:
		out.vendor, out.os = "apple", "darwin"
	case device.Linux:
		out.os = "linux"
	case device.Android:
		out.os, out.abi = "linux", "androideabi"
	}

	return out
}

// Triple represents an LLVM target triple.
type Triple struct {
	arch, vendor, os, abi string
}

// NewTriple returns a new Triple.
func NewTriple(arch, vendor, os, abi string) Triple {
	return Triple{arch, vendor, os, abi}
}

func (t Triple) String() string {
	return strings.Join([]string{t.arch, t.vendor, t.os, t.abi}, "-")
}
