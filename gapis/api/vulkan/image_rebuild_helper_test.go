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

package vulkan

import (
	"testing"

	"github.com/google/gapid/core/assert"
)

func TestExtractBitsToUint32(t *testing.T) {
	valid := func(src []uint8, start, oneOverEnd, expected uint32) {
		u, _ := extractBitsToUin32(src, start, oneOverEnd)
		assert.To(t).For("Extract bits: [%d, %d) from %v", start, oneOverEnd, src).That(u).Equals(expected)
	}
	invalid := func(src []uint8, start, oneOverEnd uint32) {
		_, err := extractBitsToUin32(src, start, oneOverEnd)
		assert.To(t).For("Extract bits: [%d, %d) from %v", start, oneOverEnd, src).That(err).IsNotNil()
	}
	// Invalid cases
	invalid([]uint8{}, 0, 0)
	invalid([]uint8{}, 1, 0)
	invalid([]uint8{}, 0, 1)
	invalid([]uint8{}, 1, 1)
	invalid([]uint8{0xFF}, 0, 0)
	invalid([]uint8{0xFF}, 1, 0)
	invalid([]uint8{0xFF}, 1, 1)
	invalid([]uint8{0xFF}, 0, 10)

	// Single byte
	valid([]uint8{0xab}, 0, 1, 1)
	valid([]uint8{0xab}, 1, 2, 1)
	valid([]uint8{0xab}, 2, 3, 0)
	valid([]uint8{0xab}, 3, 4, 1)
	valid([]uint8{0xab}, 4, 5, 0)
	valid([]uint8{0xab}, 5, 6, 1)
	valid([]uint8{0xab}, 6, 7, 0)
	valid([]uint8{0xab}, 7, 8, 1)

	valid([]uint8{0xab}, 0, 2, 3)
	valid([]uint8{0xab}, 1, 3, 1)
	valid([]uint8{0xab}, 2, 4, 2)
	valid([]uint8{0xab}, 3, 5, 1)
	valid([]uint8{0xab}, 4, 6, 2)
	valid([]uint8{0xab}, 5, 7, 1)
	valid([]uint8{0xab}, 6, 8, 2)

	valid([]uint8{0xab}, 0, 3, 3)
	valid([]uint8{0xab}, 1, 4, 5)
	valid([]uint8{0xab}, 2, 5, 2)
	valid([]uint8{0xab}, 3, 6, 5)
	valid([]uint8{0xab}, 4, 7, 2)
	valid([]uint8{0xab}, 5, 8, 5)

	valid([]uint8{0xab}, 0, 4, 11)
	valid([]uint8{0xab}, 1, 5, 5)
	valid([]uint8{0xab}, 2, 6, 10)
	valid([]uint8{0xab}, 3, 7, 5)
	valid([]uint8{0xab}, 4, 8, 10)

	valid([]uint8{0xab}, 0, 5, 11)
	valid([]uint8{0xab}, 1, 6, 21)
	valid([]uint8{0xab}, 2, 7, 10)
	valid([]uint8{0xab}, 3, 8, 21)

	valid([]uint8{0xab}, 0, 6, 0x2b)
	valid([]uint8{0xab}, 1, 7, 0x15)
	valid([]uint8{0xab}, 2, 8, 0x2a)

	valid([]uint8{0xab}, 0, 7, 0x2b)
	valid([]uint8{0xab}, 1, 8, 0x55)

	valid([]uint8{0xab}, 0, 8, 0xab)

	// Double byte
	valid([]uint8{0xab, 0xcd}, 3, 13, 437)
	valid([]uint8{0xab, 0xcd}, 3, 14, 437)
	valid([]uint8{0xab, 0xcd}, 3, 15, 2485)
	valid([]uint8{0xab, 0xcd}, 3, 16, 6581)

	valid([]uint8{0xab, 0xcd}, 7, 13, 0x1b)
	valid([]uint8{0xab, 0xcd}, 7, 14, 0x1b)
	valid([]uint8{0xab, 0xcd}, 7, 15, 0x9b)
	valid([]uint8{0xab, 0xcd}, 7, 16, 411)

	valid([]uint8{0xab, 0xcd}, 0, 4, 0xb)
	valid([]uint8{0xab, 0xcd}, 4, 8, 0xa)
	valid([]uint8{0xab, 0xcd}, 8, 12, 0xd)
	valid([]uint8{0xab, 0xcd}, 12, 16, 0xc)

	valid([]uint8{0xab, 0xcd}, 0, 8, 0xab)
	valid([]uint8{0xab, 0xcd}, 8, 16, 0xcd)

	// Multiple bytes
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 0, 1, 0)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 1, 3, 1)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 2, 5, 4)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 3, 7, 2)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 4, 9, 1)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 5, 11, 32)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 6, 13, 80)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 7, 15, 104)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 8, 17, 52)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 9, 19, 794)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 10, 21, 1421)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 11, 23, 2758)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 12, 25, 1379)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 13, 27, 689)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 14, 29, 24920)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 15, 31, 61612)
	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 16, 33, 30806)

	valid([]uint8{0x12, 0x34, 0x56, 0x78, 0x9a}, 8, 40, 0x9a785634)
}

func TestToUint32(t *testing.T) {
	test := func(src []uint8, start, oneOverEnd uint32, sign signedness, expected uint32) {
		u, _ := unpackIntToUint32(src, start, oneOverEnd, sign)
		assert.To(t).For("%v: bits: [%d, %d), signedness: %v", src, start, oneOverEnd, sign).That(u).Equals(expected)
	}
	test([]uint8{0x0}, 0, 1, unsigned, 0)
	test([]uint8{0x0}, 0, 1, signed, 0)

	test([]uint8{0x1}, 0, 1, unsigned, 1)
	test([]uint8{0x1}, 0, 1, signed, 0xFFFFFFFF)

	test([]uint8{0xab, 0xcd, 0xef}, 13, 23, unsigned, 894)
	test([]uint8{0xab, 0xcd, 0xef}, 13, 23, signed, 4294967166)
}

func TestToFloat32(t *testing.T) {
	test := func(src []uint8, fracStart, oneOverFracEnd, expStart, oneOverExpEnd uint32, sign signedness, signBit uint32, expected float32) {
		f, _ := unpackFloatToFloat32(src, fracStart, oneOverFracEnd, expStart, oneOverExpEnd, sign, signBit)
		assert.To(t).For("%v: frac bits: [%d, %d), exp bits: [%d, %d), signedness: %v, sign bit: %d", src, fracStart, oneOverFracEnd, expStart, oneOverExpEnd, sign, signBit).That(f).Equals(expected)
	}
	test([]uint8{0x00}, 0, 4, 4, 8, unsigned, 0, 0.0)
	test([]uint8{0x00}, 0, 4, 4, 7, signed, 0, 0.0)

	// TODO: handle half precision and other custom float point types.

	test([]uint8{0x0, 0x0, 0xa0, 0x3f}, 0, 23, 23, 31, signed, 31, 1.25)
	test([]uint8{0x0, 0x0, 0xa0, 0xbf}, 0, 23, 23, 31, signed, 31, -1.25)
}
