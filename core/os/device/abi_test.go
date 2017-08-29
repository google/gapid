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
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/os/device"
)

var abiTestData = []struct {
	abi         *device.ABI
	name        string
	width       int
	align       int32
	pointerSize int32
	intSize     int32
	endian      device.Endian
}{
	{device.UnknownABI, "unknown", 0, 0, 0, 0, device.UnknownEndian},
	{device.AndroidARMv7a, "armeabi-v7a", 32, 4, 4, 4, device.LittleEndian},
	{device.AndroidARM64v8a, "arm64-v8a", 64, 8, 8, 8, device.LittleEndian},
	{device.AndroidX86, "x86", 32, 4, 4, 4, device.LittleEndian},
	{device.AndroidX86_64, "x86-64", 64, 8, 8, 8, device.LittleEndian},
	{device.AndroidMIPS, "mips", 32, 4, 4, 4, device.LittleEndian},
	{device.AndroidMIPS64, "mips64", 64, 8, 8, 8, device.LittleEndian},
}

func TestABI(t *testing.T) {
	assert := assert.To(t)
	for _, test := range abiTestData {
		name := fmt.Sprintf("%s ABI=%s", test.name, test.abi.Name)
		abi := device.ABIByName(test.name)
		// TODO: there's a collision between the x86-64 ABIs for OSX and Android.
		// Should the ABI even include the OS?
		// assert.For(ctx, "ABIByName").That(abi).Equals(test.abi)
		assert.For("%s ABI.Architecture.Bitness", name).That(abi.Architecture.Bitness()).Equals(test.width)
		assert.For("%s ABI.MemoryLayout.PointerAlignment", name).That(abi.GetMemoryLayout().GetPointer().GetAlignment()).Equals(test.align)
		assert.For("%s ABI.MemoryLayout.PointerSize", name).That(abi.GetMemoryLayout().GetPointer().GetSize()).Equals(test.pointerSize)
		assert.For("%s ABI.MemoryLayout.IntegerSize", name).That(abi.GetMemoryLayout().GetInteger().GetSize()).Equals(test.intSize)
		assert.For("%s ABI.MemoryLayout.Endian", name).That(abi.GetMemoryLayout().GetEndian()).Equals(test.endian)
	}
}

func TestABIByName(t *testing.T) {
	assert := assert.To(t)
	abi := device.ABIByName("invalid")
	assert.For("ABI.Name").That(abi.Name).Equals("invalid")
	assert.For("ABI.Architecture").That(abi.Architecture).Equals(device.UnknownArchitecture)
	assert.For("ABI.OS").That(abi.OS).Equals(device.UnknownOS)
}
