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

package codegen_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
)

func TestTargetTriple(t *testing.T) {
	ctx := log.Testing(t)
	for _, t := range []struct {
		name     string
		abi      *device.ABI
		expected codegen.Triple
	}{
		{"win-x64", device.WindowsX86_64, codegen.NewTriple("x86_64", "w64", "windows", "gnu")},
		{"osx-x64", device.OSXX86_64, codegen.NewTriple("x86_64", "apple", "darwin", "unknown")},
		{"linux-x64", device.LinuxX86_64, codegen.NewTriple("x86_64", "unknown", "linux", "unknown")},
		{"android-arm64", device.AndroidARM64v8a, codegen.NewTriple("aarch64", "unknown", "linux", "androideabi")},
		{"android-armv7a", device.AndroidARMv7a, codegen.NewTriple("armv7", "unknown", "linux", "androideabi")},
		{"android-x86", device.AndroidX86, codegen.NewTriple("i386", "unknown", "linux", "androideabi")},
		{"android-x64", device.AndroidX86_64, codegen.NewTriple("x86_64", "unknown", "linux", "androideabi")},
	} {
		assert.For(ctx, t.name).That(codegen.TargetTriple(t.abi)).Equals(t.expected)
	}
}
