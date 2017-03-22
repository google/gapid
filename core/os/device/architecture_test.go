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

package device_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
)

var architectureTestData = []struct {
	architecture device.Architecture
	name         string
	bitness      int
	align        int32
	pointerSize  int32
	intSize      int32
	endian       device.Endian
}{
	{device.UnknownArchitecture, "unknown", 0, 0, 0, 0, device.LittleEndian},
	{device.ARMv7a, "ARMv7a", 32, 4, 4, 4, device.LittleEndian},
	{device.ARMv8a, "ARMv8a", 64, 8, 8, 8, device.LittleEndian},
	{device.X86, "X86", 32, 4, 4, 4, device.LittleEndian},
	{device.X86_64, "X86_64", 64, 8, 8, 8, device.LittleEndian},
	{device.MIPS, "MIPS", 32, 4, 4, 4, device.LittleEndian},
	{device.MIPS64, "MIPS64", 64, 8, 8, 8, device.LittleEndian},
}

func TestArchitecture(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range architectureTestData {
		ctx := log.Enter(ctx, test.name)
		a := test.architecture
		m := a.MemoryLayout()
		ctx = log.V{"Architecture": a}.Bind(ctx)
		assert.For(ctx, "Architecture.Bitness()").That(a.Bitness()).Equals(test.bitness)
		assert.For(ctx, "Architecture.MemoryLayout().PointerAlignment").That(m.GetPointerAlignment()).Equals(test.align)
		assert.For(ctx, "Architecture.MemoryLayout().PointerSize").That(m.GetPointerSize()).Equals(test.pointerSize)
		assert.For(ctx, "Architecture.MemoryLayout().IntegerSize").That(m.GetIntegerSize()).Equals(test.intSize)
		assert.For(ctx, "Architecture.MemoryLayout().Endian").That(m.GetEndian()).Equals(test.endian)
	}
}

func TestArchitectureByName(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range architectureTestData {
		ctx := log.Enter(ctx, test.name)
		architecture := device.ArchitectureByName(test.name)
		assert.With(ctx).That(architecture).Equals(test.architecture)
	}
}

func TestArchitectureGOARCH(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		name         string
		architecture device.Architecture
	}{
		{"invalid", device.UnknownArchitecture},
		{"386", device.X86},
		{"amd64", device.X86_64},
		{"arm", device.ARMv7a},
	} {
		ctx := log.Enter(ctx, test.name)
		architecture := device.ArchitectureByName(test.name)
		assert.With(ctx).That(architecture).Equals(test.architecture)
	}
}
