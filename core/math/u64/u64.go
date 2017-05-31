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

package u64

// Min returns the minimum value of a and b.
func Min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum value of a and b.
func Max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// AlignUp returns the result of aligning up the given value to the given alignment.
func AlignUp(value, alignment uint64) uint64 {
	if value%alignment != 0 {
		return value + alignment - (value % alignment)
	}
	return value
}

func Byte(i uint64) byte {
	if i < 0 {
		return 0
	}
	if i > 255 {
		return 255
	}
	return byte(i)
}

func Expand4to8(v uint64) uint64 {
	v &= 0xF
	return (v << 4) | v
}

func Expand5to8(v uint64) uint64 {
	v &= 0x1F
	return (v << 3) | (v >> 2)
}

func Expand6to8(v uint64) uint64 {
	v &= 0x3F
	return (v << 2) | (v >> 4)
}

func Expand7to8(v uint64) uint64 {
	v &= 0x7F
	return (v << 1) | (v >> 6)
}
