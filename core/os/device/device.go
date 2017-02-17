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

package device

const (
	UnknownArchitecture = Architecture_UnknownArchitecture
	ARMv7a              = Architecture_ARMv7a
	ARMv8a              = Architecture_ARMv8a
	X86                 = Architecture_X86
	X86_64              = Architecture_X86_64
	MIPS                = Architecture_MIPS
	MIPS64              = Architecture_MIPS64
)

const (
	UnknownEndian = Endian_UnknownEndian
	BigEndian     = Endian_BigEndian
	LittleEndian  = Endian_LittleEndian
)

const (
	UnknownOS = OSKind_UnknownOS
	Windows   = OSKind_Windows
	OSX       = OSKind_OSX
	Linux     = OSKind_Linux
	Android   = OSKind_Android
)

var (
	Little32 = &MemoryLayout{
		PointerAlignment: 4,
		PointerSize:      4,
		IntegerSize:      4,
		SizeSize:         4,
		U64Alignment:     8,
		Endian:           LittleEndian,
	}
	Big32 = &MemoryLayout{
		PointerAlignment: 4,
		PointerSize:      4,
		IntegerSize:      4,
		SizeSize:         4,
		U64Alignment:     8,
		Endian:           BigEndian,
	}
	Big64 = &MemoryLayout{
		PointerAlignment: 8,
		PointerSize:      8,
		IntegerSize:      4,
		SizeSize:         8,
		U64Alignment:     8,
		Endian:           BigEndian,
	}
)

// Clone returns a new MemoryLayout copied from m.
func (m *MemoryLayout) Clone() *MemoryLayout {
	c := *m
	return &c
}

// SameAs returns true if the MemoryLayouts are equal.
func (m *MemoryLayout) SameAs(o *MemoryLayout) bool {
	return *m == *o
}
